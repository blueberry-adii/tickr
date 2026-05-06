package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/blueberry-adii/tickr/internal/api"
	"github.com/blueberry-adii/tickr/internal/database"
	"github.com/blueberry-adii/tickr/internal/scheduler"
	"github.com/blueberry-adii/tickr/internal/worker"
)

/*
The Main function creates a context, and cancels it when all scheduler is ready to
shut down, causing a graceful shutdown of the app.
Initializes DB, Redis, Scheduler and API Handler and Dependency Injections
Spawns n Workers in n goroutines for concurrent background jobs
*/
func main() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(ch)

	go func() {
		<-ch
		log.Println("shutdown signal recieved")
		cancel()
	}()

	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASS")
	dbHost := os.Getenv("DB_HOST")
	dbPort, _ := strconv.Atoi(os.Getenv("DB_PORT"))
	dbName := os.Getenv("DB_NAME")
	port, _ := strconv.Atoi(os.Getenv("PORT"))
	redisAddr := os.Getenv("REDIS_ADDR")

	if dbPort == 0 {
		log.Fatal("DB_PORT env var is required")
	}
	if port == 0 {
		log.Fatal("PORT env var is required")
	}

	cfg := database.Config{
		User:     dbUser,
		Password: dbPass,
		Host:     dbHost,
		Port:     dbPort,
		Database: dbName,
	}
	db, err := database.ConnectDB(cfg)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	repository := database.NewMySQLRepository(db)
	mux := http.NewServeMux()

	redis := scheduler.NewRedis(redisAddr)
	scheduler := scheduler.NewScheduler(redis, repository)
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

	mux.Handle("GET /api/v2/health", api.Logging(handler.Health))
	mux.Handle("POST /api/v2/jobs", api.Logging(handler.SubmitJob))

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	go func() {
		log.Printf("listening on Port %d", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
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
