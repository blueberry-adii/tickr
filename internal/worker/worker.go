package worker

import (
	"context"
	"log"
	"time"

	"github.com/blueberry-adii/tickr/internal/jobs"
	"github.com/blueberry-adii/tickr/internal/queue"
	"github.com/google/uuid"
)

type Worker struct {
	ID    string
	Queue *queue.RedisQueue
}

func NewWorker(q *queue.RedisQueue) *Worker {
	return &Worker{
		ID:    uuid.NewString(),
		Queue: q,
	}
}

func (w *Worker) Run(ctx context.Context) {
	for {
		job, err := w.Queue.Dequeue(ctx)
		if err != nil {
			log.Printf("Couldnt execute job!!!: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}
		if job == nil {
			continue
		}
		processJob(job)
	}
}

func processJob(job *jobs.Job) {
	log.Printf("Processing job %s", job.ID)
}
