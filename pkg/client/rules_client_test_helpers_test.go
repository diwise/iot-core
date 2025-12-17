package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newMockOAuthServer(tb testing.TB) *httptest.Server {
	tb.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/token" {
			tb.Fatalf("unexpected oauth path %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"test-token","expires_in":300,"token_type":"Bearer"}`))
	}))

	return server
}

func newRuleClientForTest(tb testing.TB, handler http.HandlerFunc) RuleClient {
	tb.Helper()

	apiServer := httptest.NewServer(handler)
	tb.Cleanup(apiServer.Close)

	oauthServer := newMockOAuthServer(tb)
	tb.Cleanup(oauthServer.Close)

	client, err := NewRuleClient(
		context.Background(),
		apiServer.URL,
		oauthServer.URL+"/token",
		false,
		"client-id",
		"client-secret",
	)
	if err != nil {
		tb.Fatalf("NewRuleClient: %v", err)
	}

	return client
}
