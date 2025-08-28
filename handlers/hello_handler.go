package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"

	dbhandling "github.com/hannahkm/gopherconus-2025/db_handling"
)

var (
	InstrumentationMethod string
	db                    *sql.DB
	provider              *trace.TracerProvider
)

func SetupEnv() {
	var ok bool
	InstrumentationMethod, ok = os.LookupEnv("INSTRUMENTATION")
	if !ok {
		InstrumentationMethod = "default"
	}
	slog.Info("Environment setup completed",
		"instrumentation_method", InstrumentationMethod)
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

func GetServiceName() string {
	slog.Debug("GetServiceName called",
		"OTEL_SERVICE_NAME_env", os.Getenv("OTEL_SERVICE_NAME"),
		"InstrumentationMethod", InstrumentationMethod)

	// Check for service name override from environment first
	if envServiceName := os.Getenv("OTEL_SERVICE_NAME"); envServiceName != "" {
		slog.Debug("Using service name from OTEL_SERVICE_NAME env var", "service_name", envServiceName)
		return envServiceName
	}

	// Otherwise, use instrumentation method to determine service name
	var serviceName string
	switch InstrumentationMethod {
	case "manual":
		serviceName = "gopherconus-manual"
	case "orchestrion":
		serviceName = "gopherconus-orchestrion"
	case "ebpf":
		serviceName = "gopherconus-ebpf"
	default:
		serviceName = "gopherconus-default"
	}

	slog.Debug("Using service name based on instrumentation method",
		"service_name", serviceName,
		"instrumentation_method", InstrumentationMethod)
	return serviceName
}

func SetupTraceProvider(serviceName string) func(context.Context) error {
	ctx := context.Background()

	// Check for service name override from environment
	if envServiceName := os.Getenv("OTEL_SERVICE_NAME"); envServiceName != "" {
		serviceName = envServiceName
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		slog.Error("Failed to create OTel resource", "error", err)
		os.Exit(1)
	}
	slog.Info("Created OTel resource",
		"service_name", serviceName,
		"service_version", "1.0.0")

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	slog.Debug("Set OTel text map propagators")

	// Get endpoint from environment variable or use default
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:4318"
	}
	slog.Info("Configuring OTel exporter",
		"endpoint", endpoint,
		"timeout", "30s",
		"insecure", true)

	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithTimeout(30*time.Second),
	)
	if err != nil {
		slog.Error("Failed to create trace exporter",
			"endpoint", endpoint,
			"error", err)
		os.Exit(1)
	}
	slog.Info("Successfully created OTel trace exporter", "endpoint", endpoint)

	provider = trace.NewTracerProvider(
		trace.WithBatcher(exporter,
			trace.WithBatchTimeout(5*time.Second),
			trace.WithMaxExportBatchSize(100),
			trace.WithMaxQueueSize(1000),
		),
		trace.WithResource(res),
	)
	slog.Info("Created OTel TracerProvider",
		"batch_timeout", "5s",
		"max_batch_size", 100,
		"max_queue_size", 1000)

	otel.SetTracerProvider(provider)
	slog.Info("Successfully set global OTel TracerProvider")

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
		slog.Debug("Injecting random error for testing", "instrumentation", instrumentation)
	}

	slog.Debug("Processing request",
		"method", r.Method,
		"path", r.URL.Path,
		"instrumentation", instrumentation,
		"error_injection", isErr)

	errPOST := dbhandling.POST(r.Context(), db, instrumentation, false)
	if errPOST != nil {
		slog.Warn("Database POST operation failed", "error", errPOST)
	}

	_, errGET := dbhandling.GET(r.Context(), db, 5)
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
		slog.Info("Responding with injected error", "status", status)
	} else if errPOST != nil || errGET != nil {
		status = http.StatusBadRequest
		slog.Info("Responding with database error", "status", status)
	} else {
		slog.Debug("Responding with success", "status", status)
	}
	w.WriteHeader(status)

	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		slog.Error("Failed to encode JSON response", "error", err)
		http.Error(w, "Failed to get response", http.StatusInternalServerError)
	}
}
