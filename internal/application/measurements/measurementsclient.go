package measurements

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/diwise/iot-core/internal/infrastructure/cache"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

var tracer = otel.Tracer("measurements-client")

type measurementsClient struct {
	url              string
	httpClient       http.Client
	baseTransport    *http.Transport
	c                *cache.Cache
	stopCacheCleanup func()
}

type MeasurementsClient interface {
	MaxValueFinder
	CountBoolValueFinder
	Close()
}

type MaxValueFinder interface {
	GetMaxValue(ctx context.Context, measurmentID string) (float64, error)
}

type CountBoolValueFinder interface {
	GetCountTrueValues(ctx context.Context, measurmentID string, timeAt, endTimeAt time.Time) (float64, error)
}

type meta struct {
	TotalRecords uint64  `json:"totalRecords"`
	Offset       *uint64 `json:"offset,omitempty"`
	Limit        *uint64 `json:"limit,omitempty"`
	Count        *uint64 `json:"count,omitempty"`
}

type jsonApiResponse struct {
	Meta *meta           `json:"meta,omitempty"`
	Data json.RawMessage `json:"data"`
}

type AggrResult struct {
	Average *float64 `json:"avg,omitempty"`
	Total   *float64 `json:"sum,omitempty"`
	Minimum *float64 `json:"min,omitempty"`
	Maximum *float64 `json:"max,omitempty"`
	Count   *uint64  `json:"count,omitempty"`
}

func NewMeasurementsClient(ctx context.Context, url, oauthTokenURL, oauthClientID, oauthClientSecret string, oauthInsecureURL bool) (MeasurementsClient, error) {

	// configure transport
	baseTransport := http.DefaultTransport.(*http.Transport).Clone()
	baseTransport.MaxIdleConns = 100
	baseTransport.MaxIdleConnsPerHost = 20
	baseTransport.IdleConnTimeout = 90 * time.Second

	// skip TLS verification if configured (e.g. for local testing with self-signed certs)
	if oauthInsecureURL {
		baseTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	// client for OAuth tokens
	oauthClient := &http.Client{
		Transport: otelhttp.NewTransport(baseTransport),
		Timeout:   10 * time.Second,
	}

	// Create OAuth context that will be reused for all token operations
	oauthCtx := context.WithValue(context.Background(), oauth2.HTTPClient, oauthClient)

	oauthConfig := &clientcredentials.Config{
		ClientID:     oauthClientID,
		ClientSecret: oauthClientSecret,
		TokenURL:     oauthTokenURL,
	}

	// maby we should use oauthConfig.Client(oauthCtx) instead of creating our own transport?
	ts := oauthConfig.TokenSource(oauthCtx)

	// ts only to be able to wrap the transport with otelhttp
	apiTransport := &oauth2.Transport{
		Source: ts,
		Base:   otelhttp.NewTransport(baseTransport),
	}

	// client for API requests, using the OAuth transport to automatically add tokens and refresh them as needed
	apiClient := &http.Client{
		Transport: apiTransport,
		Timeout:   30 * time.Second,
	}

	// fail fast if token cannot be retrieved with the provided credentials
	token, err := ts.Token()
	if err != nil {
		baseTransport.CloseIdleConnections()
		return nil, fmt.Errorf("failed to get client credentials from %s: %w", oauthConfig.TokenURL, err)
	}

	if !token.Valid() {
		baseTransport.CloseIdleConnections()
		return nil, fmt.Errorf("an invalid token was returned from %s", oauthTokenURL)
	}

	c := cache.NewCache()
	stopCacheCleanup := c.Cleanup(5 * time.Minute)

	return &measurementsClient{
		url:              strings.TrimSuffix(url, "/"),
		httpClient:       *apiClient,
		baseTransport:    baseTransport,
		c:                c,
		stopCacheCleanup: stopCacheCleanup,
	}, nil
}

// Close stops the cache cleanup goroutine and closes idle HTTP
// connections on the owned base transport. Calling CloseIdleConnections
// on the api client alone is not enough: its immediate transport is the
// oauth2 transport, which does not implement the method, so the shared
// base transport would stay open. Safe to call more than once.
func (c *measurementsClient) Close() {
	if c.stopCacheCleanup != nil {
		c.stopCacheCleanup()
	}
	c.httpClient.CloseIdleConnections()
	if c.baseTransport != nil {
		c.baseTransport.CloseIdleConnections()
	}
}

func (c measurementsClient) GetMaxValue(ctx context.Context, measurmentID string) (float64, error) {
	aggrResult, err := c.getAggrValue(ctx, measurmentID, "max")
	if err != nil {
		return 0.0, err
	}

	if aggrResult.Maximum == nil {
		return 0.0, fmt.Errorf("no maximum value found")
	}

	return *aggrResult.Maximum, nil
}

func (c measurementsClient) GetCountTrueValues(ctx context.Context, measurmentID string, timeAt, endTimeAt time.Time) (float64, error) {
	params := url.Values{}
	params.Add("id", measurmentID)
	params.Add("aggrMethods", "count")
	params.Add("timeAt", timeAt.Format(time.RFC3339))
	params.Add("endTimeAt", endTimeAt.Format(time.RFC3339))

	jar, err := c.getApiResponse(ctx, params)
	if err != nil {
		return 0.0, err
	}

	var aggrResult AggrResult
	err = json.Unmarshal(jar.Data, &aggrResult)
	if err != nil {
		return 0.0, err
	}

	if aggrResult.Count == nil {
		return 0.0, nil
	}

	return float64(*aggrResult.Count), nil
}

func (c measurementsClient) getAggrValue(ctx context.Context, measurmentID string, aggrMethods ...string) (*AggrResult, error) {
	params := url.Values{}
	params.Add("id", measurmentID)
	params.Add("aggrMethods", strings.Join(aggrMethods, ","))

	jar, err := c.getApiResponse(ctx, params)
	if err != nil {
		return nil, err
	}

	var aggrResult AggrResult
	err = json.Unmarshal(jar.Data, &aggrResult)
	if err != nil {
		return nil, err
	}

	return &aggrResult, nil
}

func (c measurementsClient) getApiResponse(ctx context.Context, params url.Values) (*jsonApiResponse, error) {
	var err error
	ctx, span := tracer.Start(ctx, "get-measurement-values")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	log := logging.GetFromContext(ctx)

	url := fmt.Sprintf("%s/%s?%s", c.url, "api/v0/measurements", params.Encode())

	cachedItem, found := c.c.Get(url)
	if found {
		jar, ok := cachedItem.(jsonApiResponse)
		if ok {
			return &jar, nil
		}

		log.Warn(fmt.Sprintf("found response for %s in cache but could not cast to JsonApiResponse", url))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		err = fmt.Errorf("failed to create http request: %w", err)
		return nil, err
	}

	req.Header.Add("Accept", "application/vnd.api+json")

	// Authorization is attached by the oauth2 transport wrapping
	// httpClient. Fetching a token here as well would only add a
	// redundant token request on the global default transport.
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		err = fmt.Errorf("request failed, not authorized")
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("not found")
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("request failed with status code %d", resp.StatusCode)
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("failed to read response body: %w", err)
		return nil, err
	}

	log.Debug("received measurements response", "size", len(body))

	jar := jsonApiResponse{}
	err = json.Unmarshal(body, &jar)
	if err != nil {
		err = fmt.Errorf("failed to unmarshal response body: %w", err)
		return nil, err
	}

	c.c.Set(url, jar, 1*time.Minute)

	return &jar, nil
}
