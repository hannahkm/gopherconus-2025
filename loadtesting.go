package loadtesting

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
)

type helloResponse struct {
	Status  string `json:"status,omitempty"`
	Message string `json:"message"`
}

func SetupEndpoints() *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/hello", helloHandler)

	return httptest.NewServer(mux)
}

func helloHandler(w http.ResponseWriter, _ *http.Request) {
	response := helloResponse{
		Message: "Hello World!",
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, "Failed to get response", http.StatusInternalServerError)
	}
}
