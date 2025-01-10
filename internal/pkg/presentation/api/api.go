package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/diwise/iot-core/internal/pkg/application"
	"github.com/diwise/iot-core/internal/pkg/presentation/api/auth"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

func RegisterHandlers(ctx context.Context, serviceName string, rootMux *http.ServeMux, app application.App, policies io.Reader) error {
	log := logging.GetFromContext(ctx)

	authenticator, err := auth.NewAuthenticator(ctx, policies)
	if err != nil {
		return fmt.Errorf("failed to create api authenticator: %w", err)
	}

	const apiPrefix string = "/api/v0"

	mux := http.NewServeMux()
	mux.HandleFunc("GET /functions", NewQueryFunctionsHandler(app, log))

	routeGroup := http.StripPrefix(apiPrefix, mux)
	rootMux.Handle("GET "+apiPrefix+"/", authenticator(routeGroup))

	return nil
}

type response struct {
	Data any `json:"data"`
}

func write(w http.ResponseWriter, statusCode int, data any) {
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response{Data: data})
}

func NewQueryFunctionsHandler(app application.App, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		ctx := r.Context()
		f, err := app.Query(ctx, nil)
		if err != nil {
			log.Error("failed to query functions", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		write(w, http.StatusOK, f)
	}
}

/*


import (
	"context"
	"net/http"

	"github.com/diwise/iot-core/internal/pkg/application/functions"
	"github.com/go-chi/chi/v5"
	"github.com/rs/cors"
)

type API interface {
	Router() *chi.Mux
}

func New(ctx context.Context, registry functions.Registry) API {
	api_ := &api{
		router: chi.NewRouter(),
	}

	api_.router.Use(cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		Debug:            false,
	}).Handler)

	// TODO: Introduce an authenticator to manage tenant access
	api_.router.Get("/api/functions", NewQueryFunctionsHandler(ctx, registry))
	api_.router.Get("/api/functions/{id}/history", NewQueryFunctionHistoryHandler(ctx, registry))

	api_.router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		w.WriteHeader(http.StatusOK)
	})

	return api_
}

type api struct {
	router *chi.Mux
}

func (a *api) Router() *chi.Mux {
	return a.router
}
*/
