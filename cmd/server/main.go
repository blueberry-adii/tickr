package main

import (
	"context"
	"net/http"

	"github.com/blueberry-adii/tickr/internal/api"
)

var ctx = context.Background()

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/health", api.Health)

	http.ListenAndServe(":8080", mux)
}
