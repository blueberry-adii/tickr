package main

import (
	"context"
	"log"
	"net/http"

	"github.com/blueberry-adii/tickr/internal/api"
)

var ctx = context.Background()

func main() {
	mux := http.NewServeMux()

	mux.Handle("/api/v1/health/", api.Logging(api.Health))

	log.Fatal(http.ListenAndServe(":8080", mux))
}
