package handlers

import (
	"encoding/json"
	"net/http"
)

type HelloResponse struct {
	Status  string `json:"status,omitempty"`
	Message string `json:"message"`
}

func HelloHandler(w http.ResponseWriter, _ *http.Request) {
	response := HelloResponse{
		Message: "Hello, default instrumentation!",
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, "Failed to get response", http.StatusInternalServerError)
	}
}
