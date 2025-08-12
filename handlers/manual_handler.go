package handlers

import (
	"encoding/json"
	"math/rand/v2"
	"net/http"

	"go.opentelemetry.io/otel"
)

func ManualHandler(w http.ResponseWriter, r *http.Request) {
	// Give a 1/10 chance for the handler to respond with an error
	instrumentation := InstrumentationMethod
	if rand.IntN(10) == 0 {
		instrumentation = "WRONG"
	}

	tracer := otel.Tracer("manual")
	_, span := tracer.Start(r.Context(), "hello")
	defer span.End()

	response := HelloResponse{
		Message:    "Hello, " + instrumentation + " instrumentation!",
		SystemInfo: getSystemStats(),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, "Failed to get response", http.StatusInternalServerError)
	}
}
