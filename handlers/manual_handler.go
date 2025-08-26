package handlers

import (
	"encoding/json"
	"math/rand/v2"
	"net/http"

	dbhandling "github.com/hannahkm/gopherconus-2025/db_handling"
	"go.opentelemetry.io/otel"
)

func ManualHandler(w http.ResponseWriter, r *http.Request) {
	// Give a 1/10 chance for the handler to respond with an error
	instrumentation := InstrumentationMethod
	isErr := rand.IntN(10) == 0
	if isErr {
		instrumentation = "WRONG"
	}

	tracer := otel.Tracer("manual")
	ctx, span := tracer.Start(r.Context(), "hello")
	defer span.End()
	*r = *r.WithContext(ctx)

	_, dbSpan := tracer.Start(ctx, "database")
	errPOST := dbhandling.POST(db, instrumentation, false)
	_, errGET := dbhandling.GET(db, 5)
	dbSpan.End()

	response := HelloResponse{
		Message:    "Hello, " + instrumentation + " instrumentation!",
		SystemInfo: getSystemStats(),
	}

	w.Header().Set("Content-Type", "application/json")
	status := http.StatusOK
	if isErr {
		status = http.StatusInternalServerError
	} else if errPOST != nil || errGET != nil {
		status = http.StatusBadRequest
	}
	w.WriteHeader(status)

	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, "Failed to get response", http.StatusInternalServerError)
	}
}
