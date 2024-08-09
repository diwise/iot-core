package measurements

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"golang.org/x/oauth2/clientcredentials"
)

var tracer = otel.Tracer("measurements-client")

type measurementsClient struct {
	url               string
	clientCredentials *clientcredentials.Config
	httpClient        http.Client
	cache             *Cache
}

type MeasurementsClient interface {
	MaxValueFinder
}

type MaxValueFinder interface {
	GetMaxValue(ctx context.Context, measurmentID string) (float64, error)
}

type jsonApiResponse struct {
	Data json.RawMessage `json:"data"`
}

type AggrResult struct {
	Average *float64 `json:"avg,omitempty"`
	Total   *float64 `json:"sum,omitempty"`
	Minimum *float64 `json:"min,omitempty"`
	Maximum *float64 `json:"max,omitempty"`
}

func NewMeasurementsClient(ctx context.Context, url, oauthTokenURL, oauthClientID, oauthClientSecret string) (MeasurementsClient, error) {
	oauthConfig := &clientcredentials.Config{
		ClientID:     oauthClientID,
		ClientSecret: oauthClientSecret,
		TokenURL:     oauthTokenURL,
	}

	token, err := oauthConfig.Token(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get client credentials from %s: %w", oauthConfig.TokenURL, err)
	}

	if !token.Valid() {
		return nil, fmt.Errorf("an invalid token was returned from %s", oauthTokenURL)
	}

	c := NewCache()
	c.Cleanup(5 * time.Minute)

	return &measurementsClient{
		url:               strings.TrimSuffix(url, "/"),
		clientCredentials: oauthConfig,
		httpClient: http.Client{
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
		cache: c,
	}, nil
}

func (c measurementsClient) GetMaxValue(ctx context.Context, measurmentID string) (float64, error) {
	aggrResult, err := c.GetAggrValue(ctx, measurmentID, "max")
	if err != nil {
		return 0.0, err
	}

	if aggrResult.Maximum == nil {
		return 0.0, fmt.Errorf("no maximum value found")
	}

	return *aggrResult.Maximum, nil
}

func (c measurementsClient) GetAggrValue(ctx context.Context, measurmentID string, aggrMethods ...string) (*AggrResult, error) {
	jar, err := c.getAggrValues(ctx, measurmentID, aggrMethods...)
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

func (c measurementsClient) getAggrValues(ctx context.Context, measurmentID string, aggrMethods ...string) (*jsonApiResponse, error) {
	var err error
	ctx, span := tracer.Start(ctx, "get-aggr-values")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	log := logging.GetFromContext(ctx)

	params := url.Values{}
	params.Add("id", measurmentID)
	params.Add("aggrMethods", strings.Join(aggrMethods, ","))

	url := fmt.Sprintf("%s/%s?%s", c.url, "api/v0/measurements", params.Encode())

	cachedItem, found := c.cache.Get(url)
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

	if c.clientCredentials != nil {
		token, err := c.clientCredentials.Token(ctx)
		if err != nil {
			err = fmt.Errorf("failed to get client credentials from %s: %w", c.clientCredentials.TokenURL, err)
			return nil, err
		}

		req.Header.Add("Authorization", fmt.Sprintf("%s %s", token.TokenType, token.AccessToken))
	}

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

	log.Debug(fmt.Sprintf("response body: %s", string(body)))

	jar := jsonApiResponse{}
	err = json.Unmarshal(body, &jar)
	if err != nil {
		err = fmt.Errorf("failed to unmarshal response body: %w", err)
		return nil, err
	}

	c.cache.Set(url, jar, 1*time.Minute)

	return &jar, nil
}