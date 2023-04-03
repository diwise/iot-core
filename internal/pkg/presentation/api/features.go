package api

import (
	"encoding/json"
	"net/http"

	"github.com/diwise/iot-core/internal/pkg/application/functions"
)

func NewQueryFunctionsHandler(registry functions.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		functions, _ := registry.Find(r.Context(), functions.MatchAll())
		b, _ := json.MarshalIndent(functions, "  ", "  ")

		w.WriteHeader(http.StatusOK)
		w.Write(b)
	}
}
