package worker

import (
	"context"
	"log"
	"time"

	"github.com/blueberry-adii/tickr/internal/scheduler"
)

type Worker struct {
	ID        int
	Scheduler *scheduler.Scheduler
}

func NewWorker(id int, s *scheduler.Scheduler) *Worker {
	return &Worker{
		ID:        id,
		Scheduler: s,
	}
}

func (w *Worker) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		job, err := w.Scheduler.PopReadyQueue(ctx)
		if err != nil {
			log.Printf("Couldnt execute job!!!: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}
		if job == nil {
			continue
		}
		log.Printf("worker %v took job %v", w.ID, job.ID)
		exec := Executor{worker: w}
		exec.ExecuteJob(job)
	}
}
