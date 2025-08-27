package handlers

import (
	"database/sql"
	"encoding/json"
	"math/rand/v2"
	"net/http"
	"os"

	dbhandling "github.com/hannahkm/gopherconus-2025/db_handling"
)

var InstrumentationMethod string
var db *sql.DB

func SetupEnv() {
	var ok bool
	InstrumentationMethod, ok = os.LookupEnv("INSTRUMENTATION")
	if !ok {
		InstrumentationMethod = "default"
	}
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
	}

	errPOST := dbhandling.POST(db, instrumentation, false)
	_, errGET := dbhandling.GET(db, 5)

	response := HelloResponse{
		Message:    "Hello, " + instrumentation + " instrumentation!",
		SystemInfo: getSystemStats(),
	}

	w.Header().Set("Content-Type", "application/json")
	status := http.StatusOK
	if isErr {
		status = http.StatusInternalServerError
	} else if errPOST != nil || errGET != nil {
		status = http.StatusBadRequest
	}
	w.WriteHeader(status)

	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, "Failed to get response", http.StatusInternalServerError)
	}
}
