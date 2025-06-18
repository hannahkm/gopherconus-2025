package main

import (
	"net/http"

	"github.com/hannahkm/gopherconus-2025/handlers"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func main() {
	handlers.SetupEnv()

	if handlers.InstrumentationMethod == "manual" {
		ManualInstrument()
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/hello", handlers.HelloHandler)
	http.ListenAndServe(":8080", mux)
}

// Manually instrument our http call using the OTel SDK
// Code inspired by https://opentelemetry.io/docs/languages/go/getting-started
func ManualInstrument() {
	mux := http.NewServeMux()

	handleFunc := func(pattern string, handlerFunc func(http.ResponseWriter, *http.Request)) {
		// Configure the "http.route" for the HTTP instrumentation.
		handler := otelhttp.WithRouteTag(pattern, http.HandlerFunc(handlerFunc))
		mux.Handle(pattern, handler)
	}
	handleFunc("/hello", handlers.ManualHandler)

	handler := otelhttp.NewHandler(mux, "/")

	server := &http.Server{
		Addr:    ":8080",
		Handler: handler,
	}

	server.ListenAndServe()
}
