package handlers

import (
	"encoding/json"
	"log/slog"
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
	defer func() {
		span.End()
		slog.Debug("Ended span",
			"span_name", "hello",
			"trace_id", span.SpanContext().TraceID().String())
	}()
	*r = *r.WithContext(ctx)

	slog.Info("Started span",
		"span_name", "hello",
		"trace_id", span.SpanContext().TraceID().String(),
		"span_id", span.SpanContext().SpanID().String())

	errPOST := dbhandling.POST(ctx, db, instrumentation, false)
	if errPOST != nil {
		slog.Warn("Database POST operation failed", "error", errPOST)
	}

	_, errGET := dbhandling.GET(ctx, db, 5)
	if errGET != nil {
		slog.Warn("Database GET operation failed", "error", errGET)
	}

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
