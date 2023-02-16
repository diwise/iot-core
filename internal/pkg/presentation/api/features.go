package api

import (
	"encoding/json"
	"net/http"

	"github.com/diwise/iot-core/internal/pkg/application/features"
)

func NewQueryFeaturesHandler(registry features.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		features, _ := registry.Find(r.Context(), features.MatchAll())
		b, _ := json.MarshalIndent(features, "  ", "  ")

		w.WriteHeader(http.StatusOK)
		w.Write(b)
	}
}
