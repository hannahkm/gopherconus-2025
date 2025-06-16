package loadtesting

import (
	"net/http"
	"net/http/httptest"

	"github.com/hannahkm/gopherconus-2025/handlers"
)

func SetupTestingEndpoints() *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/hello", handlers.HelloHandler)

	return httptest.NewServer(mux)
}
