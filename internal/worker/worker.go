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
		log.Printf("Worker %v woke up", w.ID)
		select {
		case <-ctx.Done():
			return
		case job, ok := <-w.Scheduler.JobCh:
			if !ok {
				return
			}
			log.Printf("worker %v took job %v", w.ID, job.ID)
			exec := Executor{worker: w}
			exec.ExecuteJob(job)
		}
	}
}
