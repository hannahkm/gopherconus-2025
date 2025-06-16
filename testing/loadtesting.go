package loadtesting

import (
	"net/http"
	"net/http/httptest"

	main "github.com/hannahkm/gopherconus-2025"
)

func SetupTestingEndpoints() *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/hello", main.HelloHandler)

	return httptest.NewServer(mux)
}
