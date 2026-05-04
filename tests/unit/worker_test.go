package tests

import (
	"context"
	"testing"
	"time"

	"github.com/blueberry-adii/tickr/internal/enums"
	"github.com/blueberry-adii/tickr/internal/jobs"
	"github.com/blueberry-adii/tickr/internal/worker"
)

type MockDispatcher struct {
	ch      chan *jobs.RedisJob
	job     *jobs.Job
	retried []*jobs.RedisJob
	updated []*jobs.Job
}

func (d *MockDispatcher) Jobs() <-chan *jobs.RedisJob {
	return d.ch
}
func (d *MockDispatcher) GetJob(ctx context.Context, jobID int64) (*jobs.Job, error) {
	return d.job, nil
}
func (d *MockDispatcher) UpdateJob(ctx context.Context, job *jobs.Job) error {
	if job.Status != enums.Executing {
		job.WorkerID = nil
	}
	d.updated = append(d.updated, job)
	return nil
}
func (d *MockDispatcher) PushWaitingQueue(ctx context.Context, job *jobs.RedisJob) error {
	d.retried = append(d.retried, job)
	return nil
}

func TestWorkerMaxAttemptsAndRetryLogic(t *testing.T) {
	tests := []struct {
		name            string
		job             *jobs.Job
		expectedStatus  enums.Status
		expectedRetries int
	}{
		{
			name: "status retrying after 1 retry",
			job: &jobs.Job{
				ID:          1,
				JobType:     "unknown",
				Status:      enums.Pending,
				Attempt:     0,
				MaxAttempts: 3,
				ScheduledAt: time.Now(),
			},
			expectedStatus:  enums.Retrying,
			expectedRetries: 1,
		},
		{
			name: "fails after max attempts",
			job: &jobs.Job{
				ID:          2,
				JobType:     "unknown",
				Status:      enums.Retrying,
				Attempt:     2,
				MaxAttempts: 3,
				ScheduledAt: time.Now(),
			},
			expectedStatus:  enums.Failed,
			expectedRetries: 0,
		},
		{
			name: "succeeds using http or email job type",
			job: &jobs.Job{
				ID:      3,
				JobType: "email",
				Status:  enums.Pending,
				Payload: []byte(`{
					"to":"luffy",
					"from":"shanks",
					"body":"return the straw hat"
				}`),
				Attempt:     0,
				MaxAttempts: 3,
				ScheduledAt: time.Now(),
			},
			expectedStatus:  enums.Completed,
			expectedRetries: 0,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			d := &MockDispatcher{
				ch:      make(chan *jobs.RedisJob, 1),
				job:     tt.job,
				retried: make([]*jobs.RedisJob, 0),
				updated: make([]*jobs.Job, 0),
			}
			w := worker.NewWorker(i+1, d)

			job := jobs.RedisJob{
				JobID:       d.job.ID,
				ScheduledAt: d.job.ScheduledAt,
			}

			d.ch <- &job
			close(d.ch)

			w.Run(ctx)

			finalStatus := d.updated[len(d.updated)-1].Status
			if finalStatus != tt.expectedStatus {
				t.Errorf("expected %v, got %v", tt.expectedStatus, finalStatus)
			}

			if len(d.retried) != tt.expectedRetries {
				t.Errorf("expected %d retries, got %d", tt.expectedRetries, len(d.retried))
			}
		})
	}
}
