package worker

import (
	"context"
	"log"

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
		log.Printf("worker %v idle", w.ID)
		select {
		case <-ctx.Done():
			log.Printf("worker %d shutting down", w.ID)
			return
		case job, ok := <-w.Scheduler.JobCh:
			if !ok {
				log.Printf("worker %d shutting down", w.ID)
				return
			}
			log.Printf("worker %v took job %v", w.ID, job.ID)
			exec := Executor{worker: w}
			exec.ExecuteJob(job)
		}
	}
}
