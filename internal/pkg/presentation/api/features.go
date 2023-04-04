package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/diwise/iot-core/internal/pkg/application/features"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("iot-core/api")

func NewQueryFeaturesHandler(ctx context.Context, registry features.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		features, _ := registry.Find(r.Context(), features.MatchAll())
		b, _ := json.MarshalIndent(features, "  ", "  ")

		w.WriteHeader(http.StatusOK)
		w.Write(b)
	}
}

func NewQueryFeatureHistoryHandler(ctx context.Context, registry features.Registry) http.HandlerFunc {
	logger := logging.GetFromContext(ctx)

	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		ctx, span := tracer.Start(r.Context(), "retrieve-feature-history")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		_, ctx, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

		featureID, _ := url.QueryUnescape(chi.URLParam(r, "id"))
		if featureID == "" {
			err = fmt.Errorf("no feature id is supplied in query")
			log.Error().Err(err).Msg("bad request")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		feature, err := registry.Get(ctx, featureID)
		b, _ := json.MarshalIndent(feature, "  ", "  ")

		w.WriteHeader(http.StatusOK)
		w.Write(b)
	}
}
