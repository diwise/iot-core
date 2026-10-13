package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/diwise/iot-core/internal/application"
	"github.com/diwise/iot-core/internal/infrastructure/database/rules"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/service-chassis/pkg/infrastructure/net/http/router"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/jackc/pgx/v5/pgconn"
	"go.opentelemetry.io/otel/trace"
)

const tracerName string = "iot-agent/api"

// RegisterRulesHandlers monterar develop:s regel- och
// messagereceived-endpoints vid sidan av labs-1.27:s funktions-API.
// labs-1.27:s New(ctx, registry) med /api/functions är normerande för
// funktionsroutes; dessa endpoints behåller develop:s /api/v0-prefix
// utan att ändra labs-routes.
func RegisterRulesHandlers(ctx context.Context, rootMux *http.ServeMux, app application.App) error {
	const apiPrefix string = "/api/v0"

	r := router.New(rootMux, router.WithPrefix(apiPrefix), router.WithTaggedRoutes(true))

	logger := logging.GetFromContext(ctx)
	r.Use(loggerMiddleware(logger))

	r.Post("/functions/messagereceived", NewMessageReceivedHandler(app))

	// Rule endpoints
	r.Post("/rules", NewCreateRuleHandler(ctx, app))
	r.Get("/rules/device/{deviceId}", NewGetRulesByDeviceHandler(ctx, app))
	r.Get("/rules/{ruleId}", NewGetRuleHandler(ctx, app))
	r.Put("/rules/{ruleId}", NewUpdateRuleHandler(ctx, app))
	r.Delete("/rules/{ruleId}", NewDeleteRuleHandler(ctx, app))

	return nil
}

// RegisterHandlers bevaras som alias för develop-kod som anropar det
// äldre namnet.
func RegisterHandlers(ctx context.Context, rootMux *http.ServeMux, app application.App) error {
	return RegisterRulesHandlers(ctx, rootMux, app)
}

func NewMessageReceivedHandler(app application.App) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var err error
		ctx, endSpan := tracing.Start(r.Context(), tracerName, "message-received", func() error { return err })
		defer endSpan()

		b, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "could not read body", http.StatusBadRequest)
			return
		}

		evt := events.MessageReceived{}

		err = json.Unmarshal(b, &evt)
		if err != nil {
			http.Error(w, "could not unmarshal body", http.StatusBadRequest)
			return
		}

		ma, err := app.MessageReceived(ctx, evt)
		if err != nil {
			http.Error(w, "could not handle message received", http.StatusInternalServerError)
			return
		}

		b, err = json.Marshal(ma)
		if err != nil {
			http.Error(w, "could not marshal message accepted", http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(b)
	}
}

func NewCreateRuleHandler(ctx context.Context, app application.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var err error
		ctx, endSpan := tracing.Start(r.Context(), tracerName, "create-rule", func() error { return err })
		defer endSpan()

		b, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "could not read body", http.StatusBadRequest)
			return
		}

		rule := rules.Rule{}
		err = json.Unmarshal(b, &rule)
		if err != nil {
			http.Error(w, "could not unmarshal body", http.StatusBadRequest)
			return
		}

		err = app.CreateRule(ctx, &rule)
		if err != nil {
			http.Error(w, fmt.Sprintf("could not create rule: %s", err.Error()), statusCodeForRuleError(err, http.StatusInternalServerError))
			return
		}

		response := map[string]string{
			"id":      rule.ID,
			"message": "Rule created successfully",
		}

		b, err = json.Marshal(response)
		if err != nil {
			http.Error(w, "could not marshal response", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write(b)
	}
}

func NewGetRulesByDeviceHandler(ctx context.Context, app application.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var err error
		ctx, endSpan := tracing.Start(r.Context(), tracerName, "get-rules-by-device", func() error { return err })
		defer endSpan()

		deviceID := r.PathValue("deviceId")
		if deviceID == "" {
			http.Error(w, "no device id is supplied in path", http.StatusBadRequest)
			return
		}

		deviceRules, err := app.GetRulesByDevice(ctx, deviceID)
		if err != nil {
			http.Error(w, fmt.Sprintf("could not get rules: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		b, err := json.Marshal(deviceRules)
		if err != nil {
			http.Error(w, "could not marshal response", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(b)
	}
}

func NewGetRuleHandler(ctx context.Context, app application.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var err error
		ctx, endSpan := tracing.Start(r.Context(), tracerName, "get-rule", func() error { return err })
		defer endSpan()

		ruleID := r.PathValue("ruleId")
		if ruleID == "" {
			http.Error(w, "no rule id is supplied in path", http.StatusBadRequest)
			return
		}

		rule, err := app.GetRule(ctx, ruleID)
		if err != nil {
			http.Error(w, fmt.Sprintf("could not get rule: %s", err.Error()), statusCodeForRuleError(err, http.StatusInternalServerError))
			return
		}

		b, err := json.Marshal(rule)
		if err != nil {
			http.Error(w, "could not marshal response", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(b)
	}
}

func NewUpdateRuleHandler(ctx context.Context, app application.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var err error
		ctx, endSpan := tracing.Start(r.Context(), tracerName, "update-rule", func() error { return err })
		defer endSpan()

		logger := logging.GetFromContext(ctx)

		ruleID := r.PathValue("ruleId")
		if ruleID == "" {
			logger.Error("no rule id is supplied in path")
			http.Error(w, "no rule id is supplied in path", http.StatusBadRequest)
			return
		}

		b, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Error("could not read body", "err", err.Error())
			http.Error(w, "could not read body", http.StatusBadRequest)
			return
		}

		rule := rules.Rule{}
		err = json.Unmarshal(b, &rule)
		if err != nil {
			logger.Error("could not unmarshal body", "err", err.Error())
			http.Error(w, "could not unmarshal body", http.StatusBadRequest)
			return
		}

		rule.ID = ruleID

		err = app.UpdateRule(ctx, &rule)
		if err != nil {
			logger.Error("could not update rule", "err", err.Error())
			http.Error(w, fmt.Sprintf("could not update rule: %s", err.Error()), statusCodeForRuleError(err, http.StatusInternalServerError))
			return
		}

		response := map[string]string{
			"message": "Rule updated successfully",
		}

		b, err = json.Marshal(response)
		if err != nil {
			logger.Error("could not marshal response", "err", err.Error())
			http.Error(w, "could not marshal response", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(b)
	}
}

func NewDeleteRuleHandler(ctx context.Context, app application.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var err error
		ctx, endSpan := tracing.Start(r.Context(), tracerName, "delete-rule", func() error { return err })
		defer endSpan()

		ruleID := r.PathValue("ruleId")
		if ruleID == "" {
			http.Error(w, "no rule id is supplied in path", http.StatusBadRequest)
			return
		}

		err = app.DeleteRule(ctx, ruleID)
		if err != nil {
			http.Error(w, fmt.Sprintf("could not delete rule: %s", err.Error()), statusCodeForRuleError(err, http.StatusInternalServerError))
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func statusCodeForRuleError(err error, fallback int) int {
	if err == nil {
		return fallback
	}

	if errors.Is(err, rules.ErrNotFound) {
		return http.StatusNotFound
	}

	if errors.Is(err, rules.ErrorNoKindSet) ||
		errors.Is(err, rules.ErrorMultipleKindSet) ||
		errors.Is(err, rules.ErrorNotFloatValue) ||
		errors.Is(err, rules.ErrorVHasWrongOrder) {
		return http.StatusBadRequest
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return http.StatusConflict
	}

	return fallback
}

func loggerMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			_, ctx, _ = o11y.AddTraceIDToLoggerAndStoreInContext(
				trace.SpanFromContext(ctx),
				logger,
				ctx)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
