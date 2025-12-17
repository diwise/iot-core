package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/diwise/service-chassis/pkg/infrastructure/env"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

const rulesEndpointPrefix = "/api/v0/rules"
const rulesByDeviceEndpoint = "/api/v0/rules/device"

//go:generate moq -rm -out tests/rule_client_mock.go . RuleClient

var tracer = otel.Tracer("rules-client")

type Rule struct {
	ID              string     `json:"id"`
	MeasurementID   string     `json:"measurement_id"`
	DeviceID        string     `json:"device_id"`
	MeasurementType int        `json:"measurement_type"`
	ShouldAbort     bool       `json:"should_abort"`
	RuleValues      RuleValues `json:"rule_values"`
}

type RuleValues struct {
	V  *RuleV  `json:"v"`
	Vs *RuleVs `json:"vs"`
	Vb *RuleVb `json:"vb"`
}

type RuleV struct {
	MinValue *float64 `json:"min_value"`
	MaxValue *float64 `json:"max_value"`
}

type RuleVs struct {
	Value *string `json:"value"`
}

type RuleVb struct {
	Value *bool `json:"value"`
}

type RuleClient interface {
	CreateRule(ctx context.Context, rule Rule) (*CreateRuleResponse, error)
	GetRulesByDevice(ctx context.Context, deviceID string) ([]Rule, error)
	GetRule(ctx context.Context, ruleID string) (*Rule, error)
	UpdateRule(ctx context.Context, rule Rule) (*UpdateRuleResponse, error)
	DeleteRule(ctx context.Context, ruleID string) error
}

type ruleClient struct {
	url               string
	clientCredentials *clientcredentials.Config
	insecureURL       bool

	httpClient  http.Client
	debugClient bool
}

type CreateRuleResponse struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

type UpdateRuleResponse struct {
	Message string `json:"message"`
}

var ErrNotFound = errors.New("not found")

func NewRuleClient(
	ctx context.Context,
	rulesBaseURL, oauthTokenURL string,
	oauthInsecureURL bool,
	oauthClientID, oauthClientSecret string,
) (RuleClient, error) {
	return New(ctx, rulesBaseURL, oauthTokenURL, oauthInsecureURL, oauthClientID, oauthClientSecret)
}

func New(
	ctx context.Context,
	rulesBaseURL, oauthTokenURL string,
	oauthInsecureURL bool,
	oauthClientID, oauthClientSecret string,
) (RuleClient, error) {
	if strings.TrimSpace(oauthTokenURL) == "" {
		return nil, fmt.Errorf("oauth token url must be provided")
	}

	httpTransport := http.DefaultTransport
	if oauthInsecureURL {
		if trans, ok := httpTransport.(*http.Transport); ok {
			if trans.TLSClientConfig == nil {
				trans.TLSClientConfig = &tls.Config{}
			}
			trans.TLSClientConfig.InsecureSkipVerify = true
		}
	}

	httpClient := &http.Client{
		Transport: otelhttp.NewTransport(httpTransport),
		Timeout:   15 * time.Second,
	}

	c := &ruleClient{
		url:         strings.TrimRight(rulesBaseURL, "/"),
		insecureURL: oauthInsecureURL,
		clientCredentials: &clientcredentials.Config{
			ClientID:     oauthClientID,
			ClientSecret: oauthClientSecret,
			TokenURL:     oauthTokenURL,
		},
		httpClient:  *httpClient,
		debugClient: env.GetVariableOrDefault(ctx, "RULES_CLIENT_DEBUG", "false") == "true",
	}

	ctx = context.WithValue(ctx, oauth2.HTTPClient, httpClient)

	token, err := c.clientCredentials.Token(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get client credentials from %s: %w", c.clientCredentials.TokenURL, err)
	}

	if !token.Valid() {
		return nil, fmt.Errorf("an invalid token was returned from %s", oauthTokenURL)
	}

	return c, nil
}

func drainAndCloseResponseBody(r *http.Response) {
	defer r.Body.Close()
	_, _ = io.Copy(io.Discard, r.Body)
}

func (c *ruleClient) dumpRequestResponseIfNon200AndDebugEnabled(ctx context.Context, req *http.Request, resp *http.Response) {
	if c.debugClient && (resp.StatusCode >= http.StatusBadRequest && resp.StatusCode != http.StatusNotFound) {
		reqbytes, _ := httputil.DumpRequest(req, false)
		respbytes, _ := httputil.DumpResponse(resp, false)

		log := logging.GetFromContext(ctx)
		log.Debug("request failed", "request", string(reqbytes), "response", string(respbytes))
	}
}

func (c *ruleClient) refreshToken(ctx context.Context) (token *oauth2.Token, err error) {
	ctx, span := tracer.Start(ctx, "refresh-token")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	ctx = context.WithValue(ctx, oauth2.HTTPClient, &c.httpClient)
	token, err = c.clientCredentials.Token(ctx)
	return
}

func (c *ruleClient) newJSONRequest(ctx context.Context, method, endpoint string, body any) (*http.Request, error) {
	url := c.url + endpoint

	var reader io.Reader
	if body != nil {
		requestBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reader = bytes.NewReader(requestBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	if body != nil {
		req.Header.Add("Content-Type", "application/json")
	}

	if c.clientCredentials != nil {
		token, err := c.refreshToken(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get client credentials from %s: %w", c.clientCredentials.TokenURL, err)
		}
		req.Header.Add("Authorization", "Bearer "+token.AccessToken)
	}

	return req, nil
}

func (c *ruleClient) doAndUnmarshal(ctx context.Context, req *http.Request, expectedStatus int, treatNotFoundAsErr bool, out any) (err error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer drainAndCloseResponseBody(resp)

	c.dumpRequestResponseIfNon200AndDebugEnabled(ctx, req, resp)

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("request failed, not authorized")
	}

	if treatNotFoundAsErr && resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}

	if resp.StatusCode != expectedStatus {
		return fmt.Errorf("request failed with status code %d", resp.StatusCode)
	}

	if out == nil {
		return nil
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return nil
}

func (c *ruleClient) CreateRule(ctx context.Context, rule Rule) (*CreateRuleResponse, error) {
	var err error
	ctx, span := tracer.Start(ctx, "create-rule")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	req, err := c.newJSONRequest(ctx, http.MethodPost, rulesEndpointPrefix, rule)
	if err != nil {
		return nil, err
	}

	out := CreateRuleResponse{}
	err = c.doAndUnmarshal(ctx, req, http.StatusCreated, false, &out)
	if err != nil {
		err = fmt.Errorf("failed to create rule: %w", err)
		return nil, err
	}

	return &out, nil
}

func (c *ruleClient) GetRulesByDevice(ctx context.Context, deviceID string) ([]Rule, error) {
	var err error
	ctx, span := tracer.Start(ctx, "get-rules-by-device")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	escaped := url.PathEscape(deviceID)
	req, err := c.newJSONRequest(ctx, http.MethodGet, path.Join(rulesByDeviceEndpoint, escaped), nil)
	if err != nil {
		return nil, err
	}

	var rules []Rule
	err = c.doAndUnmarshal(ctx, req, http.StatusOK, false, &rules)
	if err != nil {
		err = fmt.Errorf("failed to get rules by device: %w", err)
		return nil, err
	}

	return rules, nil
}

func (c *ruleClient) GetRule(ctx context.Context, ruleID string) (*Rule, error) {
	var err error
	ctx, span := tracer.Start(ctx, "get-rule")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	escaped := url.PathEscape(ruleID)
	req, err := c.newJSONRequest(ctx, http.MethodGet, path.Join(rulesEndpointPrefix, escaped), nil)
	if err != nil {
		return nil, err
	}

	rule := Rule{}
	err = c.doAndUnmarshal(ctx, req, http.StatusOK, true, &rule)
	if err != nil {
		err = fmt.Errorf("failed to get rule: %w", err)
		return nil, err
	}

	return &rule, nil
}

func (c *ruleClient) UpdateRule(ctx context.Context, rule Rule) (*UpdateRuleResponse, error) {
	var err error
	ctx, span := tracer.Start(ctx, "update-rule")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	if strings.TrimSpace(rule.ID) == "" {
		err = fmt.Errorf("rule ID must be set when updating a rule")
		return nil, err
	}

	escaped := url.PathEscape(rule.ID)
	req, err := c.newJSONRequest(ctx, http.MethodPut, path.Join(rulesEndpointPrefix, escaped), rule)
	if err != nil {
		return nil, err
	}

	out := UpdateRuleResponse{}
	err = c.doAndUnmarshal(ctx, req, http.StatusOK, false, &out)
	if err != nil {
		err = fmt.Errorf("failed to update rule: %w", err)
		return nil, err
	}

	return &out, nil
}

func (c *ruleClient) DeleteRule(ctx context.Context, ruleID string) error {
	var err error
	ctx, span := tracer.Start(ctx, "delete-rule")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	escaped := url.PathEscape(ruleID)
	req, err := c.newJSONRequest(ctx, http.MethodDelete, path.Join(rulesEndpointPrefix, escaped), nil)
	if err != nil {
		return err
	}

	err = c.doAndUnmarshal(ctx, req, http.StatusNoContent, true, nil)
	if err != nil {
		return fmt.Errorf("failed to delete rule: %w", err)
	}

	return nil
}
