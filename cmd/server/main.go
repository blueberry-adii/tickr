package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/blueberry-adii/tickr/internal/api"
	"github.com/blueberry-adii/tickr/internal/scheduler"
	"github.com/blueberry-adii/tickr/internal/worker"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(ch)

	go func() {
		<-ch
		log.Println("Shutdown signal recieved")
		cancel()
	}()

	mux := http.NewServeMux()
	redis := scheduler.NewRedis("localhost:6379")
	scheduler := scheduler.NewScheduler(redis)
	handler := api.NewHandler(scheduler)

	wg.Add(1)
	go func() {
		defer wg.Done()
		scheduler.Run(ctx)
	}()

	for i := 0; i < 5; i++ {
		worker := worker.NewWorker(i+1, scheduler)
		wg.Add(1)
		go func() {
			defer wg.Done()
			worker.Run(ctx)
		}()
	}

	mux.Handle("/api/v1/health/", api.Logging(handler.Health))
	mux.Handle("/api/v1/jobs/", api.Logging(handler.SubmitJob))

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		log.Println("Listening on Port 8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-ctx.Done()

	log.Println("shutting down http server")
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown error: %v", err)
	}

	wg.Wait()
	log.Println("graceful shutdown complete")
}
