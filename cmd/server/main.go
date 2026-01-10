package main

import (
	"context"
	"log"
	"net/http"

	"github.com/blueberry-adii/tickr/internal/api"
	"github.com/blueberry-adii/tickr/internal/scheduler"
	"github.com/blueberry-adii/tickr/internal/worker"
)

var ctx = context.Background()

func main() {
	mux := http.NewServeMux()
	redis := scheduler.NewRedis("localhost:6379")
	scheduler := scheduler.NewScheduler(redis)
	handler := api.NewHandler(scheduler)

	for i := 0; i < 5; i++ {
		worker := worker.NewWorker(i+1, scheduler)
		go worker.Run(ctx)
	}

	mux.Handle("/api/v1/health/", api.Logging(handler.Health))
	mux.Handle("/api/v1/jobs/", api.Logging(handler.SubmitJob))

	log.Println("Listening on Port 8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
