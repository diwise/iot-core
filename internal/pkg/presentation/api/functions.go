package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/diwise/iot-core/internal/pkg/application/functions"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("iot-core/api")

func NewQueryFunctionsHandler(ctx context.Context, registry functions.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		functions, _ := registry.Find(r.Context(), functions.MatchAll())
		b, _ := json.MarshalIndent(functions, "  ", "  ")

		w.WriteHeader(http.StatusOK)
		w.Write(b)
	}
}

func NewQueryFunctionHistoryHandler(ctx context.Context, registry functions.Registry) http.HandlerFunc {
	logger := logging.GetFromContext(ctx)

	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		ctx, span := tracer.Start(r.Context(), "retrieve-function-history")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		_, ctx, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

		functionID, _ := url.QueryUnescape(chi.URLParam(r, "id"))
		if functionID == "" {
			err = fmt.Errorf("no function id is supplied in query")
			log.Error().Err(err).Msg("bad request")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		function, err := registry.Get(ctx, functionID)
		b, _ := json.MarshalIndent(function, "  ", "  ")

		w.WriteHeader(http.StatusOK)
		w.Write(b)
	}
}
