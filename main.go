package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/hannahkm/gopherconus-2025/handlers"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func setupLogger() *slog.Logger {
	logLevel := slog.LevelInfo
	if logLevelStr := os.Getenv("LOG_LEVEL"); logLevelStr != "" {
		switch strings.ToUpper(logLevelStr) {
		case "DEBUG":
			logLevel = slog.LevelDebug
		case "INFO":
			logLevel = slog.LevelInfo
		case "WARN":
			logLevel = slog.LevelWarn
		case "ERROR":
			logLevel = slog.LevelError
		}
	} else if os.Getenv("DEBUG") == "true" {
		logLevel = slog.LevelDebug
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
}

func main() {
	logger := setupLogger()
	slog.SetDefault(logger)
	slog.Info("Starting gopherconus application")

	handlers.SetupEnv()
	slog.Debug("Environment setup completed")

	err := handlers.SetupDB()
	if err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer func() {
		slog.Debug("Closing database connection")
		if err := handlers.StopDB(); err != nil {
			slog.Error("Error closing database", "error", err)
		}
	}()
	slog.Info("Database initialized successfully")

	var handler http.Handler
	var shutdown func(context.Context) error
	if handlers.InstrumentationMethod == "manual" {
		slog.Info("Using manual instrumentation")
		handler, shutdown = ManualInstrument()
		defer func() {
			slog.Debug("Shutting down OTel provider")
			if err := shutdown(context.Background()); err != nil {
				slog.Error("Error shutting down OTel provider", "error", err)
			}
		}()
	} else {
		slog.Info("Using default instrumentation")
		handler = DefaultInstrumentation()
	}

	server := &http.Server{
		Addr:    ":8080",
		Handler: handler,
	}

	slog.Info("Starting HTTP server", "addr", ":8080")
	err = server.ListenAndServe()
	if err != nil {
		slog.Error("HTTP server failed", "error", err)
		os.Exit(1)
	}
}

func DefaultInstrumentation() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		handlers.HelloHandler(w, r)
	})
	slog.Debug("Default instrumentation handler configured")
	return mux
}

// Manually instrument our http call using the OTel SDK
// Code inspired by https://opentelemetry.io/docs/languages/go/getting-started
func ManualInstrument() (http.Handler, func(context.Context) error) {
	// Determine service name based on instrumentation method
	var serviceName string
	switch handlers.InstrumentationMethod {
	case "manual":
		serviceName = "gopherconus-manual"
	case "orchestrion":
		serviceName = "gopherconus-orchestrion"
	case "ebpf":
		serviceName = "gopherconus-ebpf"
	default:
		serviceName = "gopherconus-default"
	}

	shutdown := handlers.SetupTraceProvider(serviceName)

	mux := http.NewServeMux()

	handleFunc := func(pattern string, handlerFunc func(http.ResponseWriter, *http.Request)) {
		handler := otelhttp.WithRouteTag(pattern, http.HandlerFunc(handlerFunc))
		mux.Handle(pattern, handler)
	}
	handleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		handlers.ManualHandler(w, r)
	})

	handler := otelhttp.NewHandler(mux, "/")
	slog.Debug("Manual instrumentation handler configured")

	return handler, shutdown
}
