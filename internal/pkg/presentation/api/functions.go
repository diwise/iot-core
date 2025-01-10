package api

/*
import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/diwise/iot-core/internal/pkg/application/functions"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("iot-core/api")

func NewQueryFunctionsHandler(ctx context.Context, registry functions.Registry) http.HandlerFunc {
	logger := logging.GetFromContext(ctx)

	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		defer r.Body.Close()

		ctx, span := tracer.Start(r.Context(), "retrieve-functions")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		_, ctx, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

		log.Debug("functions requested")

		functions, _ := registry.Find(ctx, functions.MatchAll())
		b, _ := json.MarshalIndent(functions, "  ", "  ")

		w.WriteHeader(http.StatusOK)
		w.Write(b)
	}
}

func NewQueryFunctionHistoryHandler(ctx context.Context, registry functions.Registry) http.HandlerFunc {
	logger := logging.GetFromContext(ctx)

	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		defer r.Body.Close()

		ctx, span := tracer.Start(r.Context(), "retrieve-function-history")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		_, ctx, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

		functionID, _ := url.QueryUnescape(chi.URLParam(r, "id"))
		if functionID == "" {
			err = fmt.Errorf("no function id is supplied in query")
			log.Error("bad request", "err", err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		function, err := registry.Get(ctx, functionID)
		if err != nil {
			log.Error("not found", "err", err.Error())
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
*/
