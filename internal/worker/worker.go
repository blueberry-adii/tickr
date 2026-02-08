package worker

import (
	"context"
	"log"

	"github.com/blueberry-adii/tickr/internal/enums"
	"github.com/blueberry-adii/tickr/internal/scheduler"
)

/*
Defines Worker struct with ID
and scheduler which assigns it the jobs
*/
type Worker struct {
	ID        int
	Scheduler *scheduler.Scheduler
}

/*
Worker Constructor
*/
func NewWorker(id int, s *scheduler.Scheduler) *Worker {
	return &Worker{
		ID:        id,
		Scheduler: s,
	}
}

/*
The Worker Run method is an infinite loop
which uses select case statements to block execution inside the loop
this avoids `polling and constantly running the loop to check for jobs`
and saves resources

It waits on multiple (2) channels, and when either channel provides a signal,
the block is executed and worker moves onto next iteration and blocks again
till the next signal
*/
func (w *Worker) Run(ctx context.Context) {
	for {
		log.Printf("worker %v idle", w.ID)
		select {
		/*
			Executes when the main context is cancelled
			to shutdown the worker and return from run
		*/
		case <-ctx.Done():
			log.Printf("worker %d shutting down", w.ID)
			return
		/*
			waits for a signal from job channel in scheduler.
			when there is a new job in ready queue, job channel notifies
			worker and this block is executed
		*/
		case redisJob, ok := <-w.Scheduler.JobCh:
			if !ok {
				log.Printf("worker %d shutting down", w.ID)
				return
			}
			log.Printf("worker %v took job %v", w.ID, redisJob.JobID)

			job, err := w.Scheduler.Repository.GetJob(ctx, redisJob.JobID)
			if err != nil {
				break
			}

			w.Scheduler.Repository.UpdateJobStatus(ctx, job, enums.Executing)

			/*Create a new Executor and Execute the job assigned to this worker*/
			exec := Executor{worker: w}
			err = exec.ExecuteJob(job)
			jobCtx := context.Background()

			if err != nil {
				log.Printf("error: %v", err.Error())
				w.Scheduler.Repository.UpdateJobStatus(jobCtx, job, enums.Failed)
			} else {
				w.Scheduler.Repository.UpdateJobStatus(jobCtx, job, enums.Completed)
			}
		}
	}
}
