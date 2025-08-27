package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"math/rand/v2"
	"net/http"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/trace"

	dbhandling "github.com/hannahkm/gopherconus-2025/db_handling"
)

var InstrumentationMethod string
var db *sql.DB
var provider *trace.TracerProvider

func SetupEnv() {
	var ok bool
	InstrumentationMethod, ok = os.LookupEnv("INSTRUMENTATION")
	if !ok {
		InstrumentationMethod = "default"
	}
}

func SetupDB() error {
	var err error
	if InstrumentationMethod == "manual" {
		db, err = dbhandling.Manual_InitDB()
	} else {
		db, err = dbhandling.InitDB()
	}
	return err
}

func StopDB() error {
	if db != nil {
		return db.Close()
	}
	return nil
}

func SetupTraceProvider() func(context.Context) error {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:4318"
	}
	exporter, err := otlptracehttp.New(context.Background(),
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithEndpoint(endpoint),
	)
	if err != nil {
		log.Fatalf("Failed to create trace exporter: %v", err)
	}
	provider = trace.NewTracerProvider(trace.WithBatcher(exporter))
	otel.SetTracerProvider(provider)
	return provider.Shutdown
}

type HelloResponse struct {
	Status     string       `json:"status,omitempty"`
	Message    string       `json:"message"`
	SystemInfo *SystemStats `json:"system_info"`
}

func HelloHandler(w http.ResponseWriter, r *http.Request) {
	// Give a 1/10 chance for the handler to respond with an error
	instrumentation := InstrumentationMethod
	isErr := rand.IntN(10) == 0
	if isErr {
		instrumentation = "WRONG"
	}

	errPOST := dbhandling.POST(db, instrumentation, false)
	_, errGET := dbhandling.GET(db, 5)

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
