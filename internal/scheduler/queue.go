package scheduler

import (
	"context"

	"github.com/blueberry-adii/tickr/internal/jobs"
)

type Queue interface {
	SaveJob(ctx context.Context, job jobs.Job) (int64, error)
	PushWaitingQueue(ctx context.Context, job *jobs.RedisJob) error
	PushReadyQueue(ctx context.Context, job *jobs.RedisJob) error
}
