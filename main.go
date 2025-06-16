package main

import (
	"net/http"

	"github.com/hannahkm/gopherconus-2025/handlers"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/hello", handlers.HelloHandler)
}
