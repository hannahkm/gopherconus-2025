package main

import (
	"log"
	"net/http"

	"github.com/hannahkm/gopherconus-2025/handlers"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func main() {
	handlers.SetupEnv()

	err := handlers.SetupDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer handlers.StopDB()

	var handler http.Handler
	if handlers.InstrumentationMethod == "manual" {
		handler = ManualInstrument()
	} else {
		handler = DefaultInstrumentation()
	}

	server := &http.Server{
		Addr:    ":8080",
		Handler: handler,
	}

	server.ListenAndServe()
}

// Create a simple /hello endpoint. We can also use this as the entry
// point for autoinstrumenting using Orchestrion + OTel
func DefaultInstrumentation() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		handlers.HelloHandler(w, r)
	})

	return mux
}

// Manually instrument our http call using the OTel SDK
// Code inspired by https://opentelemetry.io/docs/languages/go/getting-started
func ManualInstrument() http.Handler {
	mux := http.NewServeMux()

	handleFunc := func(pattern string, handlerFunc func(http.ResponseWriter, *http.Request)) {
		// Configure the "http.route" for the HTTP instrumentation.
		handler := otelhttp.WithRouteTag(pattern, http.HandlerFunc(handlerFunc))
		mux.Handle(pattern, handler)
	}
	handleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		handlers.ManualHandler(w, r)
	})

	handler := otelhttp.NewHandler(mux, "/")

	return handler
}
