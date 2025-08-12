package handlers

import (
	"encoding/json"
	"math/rand/v2"
	"net/http"
	"os"
)

var InstrumentationMethod string

func SetupEnv() {
	var ok bool
	InstrumentationMethod, ok = os.LookupEnv("INSTRUMENTATION")
	if !ok {
		InstrumentationMethod = "default"
	}
}

type HelloResponse struct {
	Status     string       `json:"status,omitempty"`
	Message    string       `json:"message"`
	SystemInfo *SystemStats `json:"system_info"`
}

func HelloHandler(w http.ResponseWriter, _ *http.Request) {
	// Give a 1/10 chance for the handler to respond with an error
	instrumentation := InstrumentationMethod
	if rand.IntN(10) == 0 {
		instrumentation = "WRONG"
	}
	response := HelloResponse{
		Message:    "Hello, " + instrumentation + " instrumentation!",
		SystemInfo: getSystemStats(),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, "Failed to get response", http.StatusInternalServerError)
	}
}
