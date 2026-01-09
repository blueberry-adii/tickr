package main

import (
	"context"
	"log"
	"net/http"

	"github.com/blueberry-adii/tickr/internal/api"
	"github.com/blueberry-adii/tickr/internal/queue"
)

var ctx = context.Background()

func main() {
	mux := http.NewServeMux()
	redisQueue := queue.NewRedisQueue("localhost:6379")
	handler := api.NewHandler(redisQueue)

	mux.Handle("/api/v1/health/", api.Logging(handler.Health))

	log.Fatal(http.ListenAndServe(":8080", mux))
}
