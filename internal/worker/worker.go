package worker

import (
	"context"
	"log"
	"time"

	"github.com/blueberry-adii/tickr/internal/queue"
)

type Worker struct {
	ID    int
	Queue *queue.RedisQueue
}

func NewWorker(id int, q *queue.RedisQueue) *Worker {
	return &Worker{
		ID:    id,
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
		log.Printf("worker %v took job %v", w.ID, job.ID)
		exec := Executor{worker: w}
		exec.ExecuteJob(job)
	}
}
