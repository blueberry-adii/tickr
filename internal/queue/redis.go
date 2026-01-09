package queue

import (
	"context"
	"encoding/json"

	"github.com/blueberry-adii/tickr/internal/jobs"
	"github.com/go-redis/redis/v8"
)

type RedisQueue struct {
	client *redis.Client
}

func NewRedisQueue(addr string) *RedisQueue {
	return &RedisQueue{
		client: redis.NewClient(&redis.Options{Addr: addr}),
	}
}

func (q *RedisQueue) Enqueue(ctx context.Context, job *jobs.Job) error {
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}

	return q.client.LPush(ctx, "tickr:queue", data).Err()
}
