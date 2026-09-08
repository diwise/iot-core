package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/diwise/iot-core/internal/pkg/infrastructure/database"
)

// Router migration (chi -> service-chassis router): locks that the API
// routes resolve, including the {id} path parameter previously read via
// chi.URLParam.
func TestAPIRoutesAfterRouterMigration(t *testing.T) {
	is, dmClient, msgCtx := testSetup(t)

	fconf := bytes.NewBufferString("fid1;name;counter;overflow;internalID;false")
	_, api, err := initialize(context.Background(), dmClient, nil, msgCtx, fconf, &database.StorageMock{
		AddFnFunc: func(ctx context.Context, id, fnType, subType, tenant, source string, lat, lon float64) error {
			return nil
		},
		AddFunc: func(ctx context.Context, id, label string, value float64, timestamp time.Time) error {
			return nil
		},
		InitializeFunc: func(contextMoqParam context.Context) error {
			return nil
		},
		HistoryFunc: func(ctx context.Context, id, label string, lastN int) ([]database.LogValue, error) {
			return []database.LogValue{}, nil
		},
	})
	is.NoErr(err)

	server := httptest.NewServer(api.Router())
	defer server.Close()

	resp, _ := testRequest(server, http.MethodGet, "/api/functions", nil)
	is.Equal(resp.StatusCode, http.StatusOK)

	resp, body := testRequest(server, http.MethodGet, "/api/functions/fid1/history", nil)
	is.Equal(resp.StatusCode, http.StatusOK)
	is.True(strings.Contains(body, `"id": "fid1"`))

	resp, _ = testRequest(server, http.MethodGet, "/health", nil)
	is.Equal(resp.StatusCode, http.StatusOK)
}
