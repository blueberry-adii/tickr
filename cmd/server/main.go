package main

import (
	"context"
	"log"
	"net/http"

	"github.com/blueberry-adii/tickr/internal/api"
	"github.com/blueberry-adii/tickr/internal/queue"
	"github.com/blueberry-adii/tickr/internal/worker"
)

var ctx = context.Background()

func main() {
	mux := http.NewServeMux()
	redisQueue := queue.NewRedisQueue("localhost:6379")
	handler := api.NewHandler(redisQueue)

	for i := 0; i < 5; i++ {
		worker := worker.NewWorker(i+1, redisQueue)
		go worker.Run(ctx)
	}

	mux.Handle("/api/v1/health/", api.Logging(handler.Health))
	mux.Handle("/api/v1/jobs/", api.Logging(handler.SubmitJob))

	log.Fatal(http.ListenAndServe(":8080", mux))
}
