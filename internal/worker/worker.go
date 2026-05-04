package worker

import (
	"context"
	"log"
	"time"

	"github.com/blueberry-adii/tickr/internal/enums"
	"github.com/blueberry-adii/tickr/internal/jobs"
)

/*
Dispatcher is the interface the worker needs from the scheduler:
a channel to receive jobs from, DB read/write access, and retry queuing.
Defined here so the worker package has no import dependency on scheduler.
*/
type Dispatcher interface {
	Jobs() <-chan *jobs.RedisJob
	GetJob(ctx context.Context, jobID int64) (*jobs.Job, error)
	UpdateJob(ctx context.Context, job *jobs.Job) error
	PushWaitingQueue(ctx context.Context, job *jobs.RedisJob) error
}

/*
Defines Worker struct with ID
and scheduler which assigns it the jobs
*/
type Worker struct {
	ID        int
	Scheduler Dispatcher
}

/*
Worker Constructor
*/
func NewWorker(id int, s Dispatcher) *Worker {
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
		case redisJob, ok := <-w.Scheduler.Jobs():
			if !ok {
				log.Printf("worker %d shutting down", w.ID)
				return
			}
			log.Printf("worker %v took job %v", w.ID, redisJob.JobID)

			/*Get Job by ID from database*/
			job, err := w.Scheduler.GetJob(ctx, redisJob.JobID)
			if err != nil {
				log.Printf("failed to fetch job %d: %v", redisJob.JobID, err)
				continue
			}

			/*Time at which job starts*/
			now := time.Now()
			job.StartedAt = &now
			job.FinishedAt = nil

			/*Set Job Status to executing and set worker ID to worker that took the job*/
			job.Status = enums.Executing
			job.WorkerID = &w.ID

			/*Update the job details into database*/
			w.Scheduler.UpdateJob(ctx, job)

			/*Create a new Executor and Execute the job assigned to this worker*/
			exec := Executor{worker: w}
			err = exec.ExecuteJob(job)
			jobCtx := context.Background()

			/*time at which job finishes execution*/
			end := time.Now()
			job.FinishedAt = &end

			/*Increment job attempt by 1*/
			job.Attempt = job.Attempt + 1
			if err != nil {
				log.Printf("error: %v", err.Error())
				errMsg := err.Error()
				job.LastError = &errMsg
				if job.Attempt < job.MaxAttempts {
					log.Printf("retry: attempt %d of job %d failed, sending back to waiting queue", job.Attempt, job.ID)
					job.Status = enums.Retrying
					w.Scheduler.UpdateJob(jobCtx, job)
					delay := end.Add(time.Second * 10 * time.Duration(job.Attempt))
					w.Scheduler.PushWaitingQueue(jobCtx, &jobs.RedisJob{JobID: job.ID, ScheduledAt: delay})
				} else {
					log.Printf("failed: attempt %d of job %d failed with max 3 attempts", job.Attempt, job.ID)
					job.Status = enums.Failed
					w.Scheduler.UpdateJob(jobCtx, job)
				}
			} else {
				log.Printf("success: attempt %d of job %d was successful", job.Attempt, job.ID)
				job.LastError = nil
				job.Status = enums.Completed
				w.Scheduler.UpdateJob(jobCtx, job)
			}
		}
	}
}
