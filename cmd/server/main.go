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
	/*
		This starts the global context with the option to cancel it
	*/
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	/*
		Wait Group to wait for go routines before ending the main func
	*/
	var wg sync.WaitGroup

	/*
		Listen to interrupt signals like 'CTRL+C'
	*/
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(ch)

	/*
		Go routine that waits until an interrupt signal
		is received and then cancels the context created above
	*/
	go func() {
		<-ch
		log.Println("Shutdown signal recieved")
		cancel()
	}()

	/*
		Create Multiplexer Router to serve http Endpoints
	*/
	mux := http.NewServeMux()

	/*
		Create an instance of RedisClient and inject that into instance of scheduler.
		Inject the scheduler instance into http Handler
	*/
	redis := scheduler.NewRedis("localhost:6379")
	scheduler := scheduler.NewScheduler(redis)
	handler := api.NewHandler(scheduler)

	/*
		Run the scheduler as a goroutine and add it
		into the wait group as the main function shouldn't
		exit while scheduler is alive
	*/
	wg.Add(1)
	go func() {
		/*
			Remove scheduler from wait group when
			scheduler stops
		*/
		defer wg.Done()
		scheduler.Run(ctx)
	}()

	/*
		For Loop to create 5 Workers
		Replace 5 with 'n' to create 'n' concurrent
		workers
	*/
	for i := 0; i < 5; i++ {
		/*
			Create a new worker instance and add each
			worker instance to wait group
			Run worker as a goroutine and
			Remove worker from wait group when goroutine/worker ends
		*/
		worker := worker.NewWorker(i+1, scheduler)
		wg.Add(1)
		go func() {
			defer wg.Done()
			worker.Run(ctx)
		}()
	}

	/*
		API endpoints for
		1. API Health
		2. Submitting Job/Task to the application
	*/
	mux.Handle("/api/v1/health/", api.Logging(handler.Health))
	mux.Handle("/api/v1/jobs/", api.Logging(handler.SubmitJob))

	/*
		Configure the server to listen on port 8080
		and define the handler as the mux created above
	*/
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	/*
		Starting to listen the server as a goroutine
		Server returns an error when fails, except when closed
		through interrupt signal
	*/
	go func() {
		log.Println("Listening on Port 8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	/*
		Main func waits until context ctx defined is cancelled
		<-ctx.Done() is a blocking call, until ctx is cancelled
	*/
	<-ctx.Done()

	log.Println("shutting down http server")

	/*
		Create a new context shutdownCtx with timeout 5 seconds
		Cancels the shutdownCtx within 5 seconds
	*/
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	/*
		Shutdown the http server gracefully,
		This is a crucial step, as http server has to shutdown
		within 5 seconds or less, no new requests are accepted by the
		application, and no new jobs are issued
	*/
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown error: %v", err)
	}

	/*
		Wait for the scheduler and workers to finish
		existing jobs that are mid execution
	*/
	wg.Wait()
	log.Println("graceful shutdown complete")
}
