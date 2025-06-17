package handlers

import (
	"encoding/json"
	"net/http"

	"go.opentelemetry.io/otel"
)

func ManualHandler(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("manual")
	_, span := tracer.Start(r.Context(), "hello")
	defer span.End()

	response := HelloResponse{
		Message: "Hello, manual instrumentation!",
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, "Failed to get response", http.StatusInternalServerError)
	}
}
