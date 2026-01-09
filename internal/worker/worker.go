package worker

import (
	"context"
	"log"
	"time"

	"github.com/blueberry-adii/tickr/internal/jobs"
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
		w.processJob(job)
	}
}

func (w *Worker) processJob(job *jobs.Job) {
	log.Printf("%v processing job %s", w.ID, job.ID)
}
