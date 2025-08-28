package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/hannahkm/gopherconus-2025/handlers"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func getAvailablePort() (int, error) {
	if portEnv := os.Getenv("PORT"); portEnv != "" {
		port, err := strconv.Atoi(portEnv)
		if err != nil {
			return 0, fmt.Errorf("invalid PORT environment variable: %v", err)
		}
		return port, nil
	}

	// Find an available port dynamically
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	return port, nil
}

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

	port, err := getAvailablePort()
	if err != nil {
		slog.Error("Failed to get available port", "error", err)
		os.Exit(1)
	}

	addr := fmt.Sprintf(":%d", port)
	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	slog.Info("Starting HTTP server", "addr", addr, "port", port)

	// Output port for script consumption
	fmt.Printf("SERVER_PORT=%d\n", port)

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
	serviceName := handlers.GetServiceName()
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
