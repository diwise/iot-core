package api

import (
	"context"
	"net/http"

	"github.com/diwise/iot-core/internal/application/functions"
	"github.com/diwise/service-chassis/pkg/infrastructure/net/http/router"
)

type API interface {
	Router() *http.ServeMux
}

func New(ctx context.Context, registry functions.Registry) API {
	mux := http.NewServeMux()
	r := router.New(mux)

	// TODO: Introduce an authenticator to manage tenant access
	r.Get("/api/functions", NewQueryFunctionsHandler(ctx, registry))
	r.Get("/api/functions/{id}/history", NewQueryFunctionHistoryHandler(ctx, registry))

	// CORE-005: healthproberna bor på kontrollservern (servicerunner).
	// Den publika GET /health är borttagen; externa prober måste flytta.

	return &api{mux: mux}
}

type api struct {
	mux *http.ServeMux
}

func (a *api) Router() *http.ServeMux {
	return a.mux
}
