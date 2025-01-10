package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/diwise/iot-core/internal/pkg/application"
	"github.com/diwise/iot-core/internal/pkg/application/functions"
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
	mux.HandleFunc("POST /functions", NewCreateFunctionHandler(app, log))

	routeGroup := http.StripPrefix(apiPrefix, mux)
	rootMux.Handle("GET "+apiPrefix+"/", authenticator(routeGroup))
	rootMux.Handle("POST "+apiPrefix+"/", authenticator(routeGroup))

	return nil
}

type response struct {
	Data any `json:"data"`
}

func write(w http.ResponseWriter, statusCode int, data any) {
	w.WriteHeader(statusCode)
	if data != nil {
		json.NewEncoder(w).Encode(response{Data: data})
	}
}

func NewQueryFunctionsHandler(app application.App, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		ctx := r.Context()

		f, err := app.Query(ctx, toParams(r))
		if err != nil {
			log.Error("failed to query functions", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		write(w, http.StatusOK, f)
	}
}

func NewCreateFunctionHandler(app application.App, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		ctx := r.Context()

		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Error("failed to read request body", "error", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		setting := functions.Setting{}
		err = json.Unmarshal(body, &setting)
		if err != nil {
			log.Error("failed to unmarshal request body", "error", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		err = app.Register(ctx, setting)
		if err != nil {
			log.Error("failed to register function", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		write(w, http.StatusOK, nil)
	}
}

func toParams(r *http.Request) map[string]any {
	params := make(map[string]any)

	for k, v := range r.URL.Query() {
		if len(v) == 1 {
			params[k] = v[0]
		} else {
			params[k] = v
		}
	}

	return params
}
