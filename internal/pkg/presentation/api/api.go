package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/diwise/iot-core/internal/pkg/application"
	"github.com/diwise/iot-core/internal/pkg/application/functions"
	"github.com/diwise/iot-core/internal/pkg/infrastructure/database/rules"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/service-chassis/pkg/infrastructure/net/http/router"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"go.opentelemetry.io/otel/trace"
)

const tracerName string = "iot-agent/api"

func RegisterHandlers(ctx context.Context, rootMux *http.ServeMux, app application.App) error {
	const apiPrefix string = "/api/v0"

	r := router.New(rootMux, router.WithPrefix(apiPrefix), router.WithTaggedRoutes(true))

	logger := logging.GetFromContext(ctx)
	r.Use(loggerMiddleware(logger))

	r.Post("/functions/messagereceived", NewMessageReceivedHandler(app))
	//r.Get("/functions", NewQueryFunctionsHandler(ctx, app.))
	//r.Get("/functions/{id}/history", NewQueryFunctionHistoryHandler(ctx, app.registry))

	// Rule endpoints
	r.Post("/rules", NewCreateRuleHandler(ctx, app))
	r.Get("/rules/device/{deviceId}", NewGetRulesByDeviceHandler(ctx, app))
	r.Get("/rules/{ruleId}", NewGetRuleHandler(ctx, app))
	r.Put("/rules/{ruleId}", NewUpdateRuleHandler(ctx, app))
	r.Delete("/rules/{ruleId}", NewDeleteRuleHandler(ctx, app))

	return nil
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
			http.Error(w, fmt.Sprintf("could not create rule: %s", err.Error()), http.StatusInternalServerError)
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

		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	}
}

func NewGetRulesByDeviceHandler(ctx context.Context, app application.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var err error
		ctx, endSpan := tracing.Start(r.Context(), tracerName, "get-rules-by-device", func() error { return err })
		defer endSpan()

		deviceID, _ := url.QueryUnescape(r.PathValue("deviceId"))
		if deviceID == "" {
			http.Error(w, "no device id is supplied", http.StatusBadRequest)
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

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	}
}

func NewGetRuleHandler(ctx context.Context, app application.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var err error
		ctx, endSpan := tracing.Start(r.Context(), tracerName, "get-rule", func() error { return err })
		defer endSpan()

		ruleID, _ := url.QueryUnescape(r.PathValue("ruleId"))
		if ruleID == "" {
			http.Error(w, "no rule id is supplied", http.StatusBadRequest)
			return
		}

		rule, err := app.GetRule(ctx, ruleID)
		if err != nil {
			http.Error(w, fmt.Sprintf("could not get rule: %s", err.Error()), http.StatusNotFound)
			return
		}

		b, err := json.Marshal(rule)
		if err != nil {
			http.Error(w, "could not marshal response", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
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

		ruleID, _ := url.QueryUnescape(r.PathValue("ruleId"))
		if ruleID == "" {
			logger.Error("no rule id is supplied in query")
			http.Error(w, "no rule id is supplied", http.StatusBadRequest)
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
			http.Error(w, fmt.Sprintf("could not update rule: %s", err.Error()), http.StatusInternalServerError)
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

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	}
}

func NewDeleteRuleHandler(ctx context.Context, app application.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var err error
		ctx, endSpan := tracing.Start(r.Context(), tracerName, "delete-rule", func() error { return err })
		defer endSpan()

		ruleID, _ := url.QueryUnescape(r.PathValue("ruleId"))
		if ruleID == "" {
			http.Error(w, "no rule id is supplied", http.StatusBadRequest)
			return
		}

		err = app.DeleteRule(ctx, ruleID)
		if err != nil {
			http.Error(w, fmt.Sprintf("could not delete rule: %s", err.Error()), http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func NewQueryFunctionsHandler(ctx context.Context, funcRegistry functions.FuncRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var err error
		ctx, endSpan := tracing.Start(r.Context(), tracerName, "retrieve-functions", func() error { return err })
		defer endSpan()

		logging.GetFromContext(ctx).Debug("functions requested")

		functions, _ := funcRegistry.Find(ctx, functions.MatchAll())
		b, _ := json.MarshalIndent(functions, "  ", "  ")

		w.WriteHeader(http.StatusOK)
		w.Write(b)
	}
}

func NewQueryFunctionHistoryHandler(ctx context.Context, funcRegistry functions.FuncRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var err error
		ctx, endSpan := tracing.Start(r.Context(), tracerName, "retrieve-function-history", func() error { return err })
		defer endSpan()

		logger := logging.GetFromContext(ctx)

		functionID, _ := url.QueryUnescape(r.PathValue("id"))
		if functionID == "" {
			err = fmt.Errorf("no function id is supplied in query")
			logger.Error("bad request", "err", err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		function, err := funcRegistry.Get(ctx, functionID)
		if err != nil {
			logger.Error("not found", "err", err.Error())
			w.WriteHeader(http.StatusNotFound)
			return
		}

		lastN := queryUnescapeQueryInt(r, "lastN")
		label := queryUnescapeQueryStr(r, "label")

		history, _ := function.History(ctx, label, lastN)
		st := time.Time{}
		et := time.Now().UTC()

		if len(history) > 0 {
			st = history[0].Timestamp
			et = history[len(history)-1].Timestamp
		}

		response := struct {
			ID      string          `json:"id"`
			History HistoryResponse `json:"history"`
		}{
			ID: function.ID(),
			History: HistoryResponse{
				StartTime: st,
				EndTime:   et,
				Values:    history,
			},
		}

		b, _ := json.MarshalIndent(response, "  ", "  ")

		w.WriteHeader(http.StatusOK)
		w.Write(b)
	}
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

func queryUnescapeQueryStr(r *http.Request, key string) string {
	q, err := url.QueryUnescape(r.URL.Query().Get(key))
	if err != nil {
		return ""
	}
	return q
}

func queryUnescapeQueryInt(r *http.Request, key string) int {
	q, err := url.QueryUnescape(r.URL.Query().Get(key))
	if err != nil {
		return 0
	}
	i, err := strconv.Atoi(q)
	if err != nil {
		return 0
	}
	return i
}

type HistoryResponse struct {
	StartTime time.Time            `json:"startTime"`
	EndTime   time.Time            `json:"endTime"`
	Values    []functions.LogValue `json:"values"`
}
