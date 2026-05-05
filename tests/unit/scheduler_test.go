package tests

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/blueberry-adii/tickr/internal/database"
	"github.com/blueberry-adii/tickr/internal/jobs"
	"github.com/blueberry-adii/tickr/internal/scheduler"
)

type MockRepository struct {
	pending []jobs.RedisJob
}

var _ database.Repository = &MockRepository{}

func (r *MockRepository) SaveJob(ctx context.Context, job jobs.Job) (int64, error) {
	return 0, nil
}

func (r *MockRepository) GetJob(ctx context.Context, jobID int64) (*jobs.Job, error) {
	return nil, nil
}

func (r *MockRepository) UpdateJob(ctx context.Context, job *jobs.Job) error {
	return nil
}

func (r *MockRepository) GetPendingJobs(ctx context.Context) ([]jobs.RedisJob, error) {
	return r.pending, nil
}

func newTestScheduler(t *testing.T, repo database.Repository) (*scheduler.Scheduler, *miniredis.Miniredis) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis %v", err)
	}

	r := scheduler.NewRedis(mr.Addr())
	return scheduler.NewScheduler(r, repo), mr
}

func TestPopWaitingQueue(t *testing.T) {
	tests := []struct {
		name        string
		jobs        []*jobs.RedisJob
		expectedLen int
	}{
		{
			name: "all jobs with past scheduled at",
			jobs: []*jobs.RedisJob{
				{JobID: 1, ScheduledAt: time.Now().Add(-time.Minute)},
				{JobID: 1, ScheduledAt: time.Now().Add(-time.Hour)},
			},
			expectedLen: 2,
		},
		{
			name:        "empty queue with no jobs",
			jobs:        []*jobs.RedisJob{},
			expectedLen: 0,
		},
		{
			name: "all jobs scheduled for future",
			jobs: []*jobs.RedisJob{
				{JobID: 1, ScheduledAt: time.Now().Add(time.Minute)},
				{JobID: 1, ScheduledAt: time.Now().Add(time.Hour)},
			},
			expectedLen: 0,
		},
		{
			name: "mixed jobs scheduled for future and past",
			jobs: []*jobs.RedisJob{
				{JobID: 1, ScheduledAt: time.Now().Add(-time.Minute)},
				{JobID: 1, ScheduledAt: time.Now().Add(time.Hour)},
			},
			expectedLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			sc, _ := newTestScheduler(t, &MockRepository{})

			for _, job := range tt.jobs {
				sc.PushWaitingQueue(ctx, job)
			}

			jobs, err := sc.PopWaitingQueue(ctx)
			if err != nil {
				t.Fatalf("unexpected error %v", err)
			}
			if len(jobs) != tt.expectedLen {
				t.Errorf("expected %d jobs, got %d", tt.expectedLen, len(jobs))
			}
		})
	}
}

func TestSchedulerRecovery(t *testing.T) {
	pendingJobs := []jobs.RedisJob{
		{JobID: 10, ScheduledAt: time.Now().Add(time.Hour)},
		{JobID: 11, ScheduledAt: time.Now().Add(2 * time.Hour)},
	}

	sc, mr := newTestScheduler(t, &MockRepository{
		pending: pendingJobs,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	sc.Run(ctx)

	if !mr.Exists("tickr:redis:epoch") {
		t.Errorf("expected epoch key to be set after recovery, but it was missing")
	}

	members, err := mr.ZMembers("tickr:queue:waiting")
	if err != nil {
		t.Fatalf("unexpected error reading waiting queue: %v", err)
	}

	if len(members) != len(pendingJobs) {
		t.Errorf("expected %d jobs in waiting queue, got %d", len(pendingJobs), len(members))
	}
}
