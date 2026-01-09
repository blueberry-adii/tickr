package queue

import (
	"context"
	"encoding/json"
	"time"

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

	queueKey := q.getQueueKey(job.Priority)

	return q.client.LPush(ctx, queueKey, data).Err()
}

func (q *RedisQueue) getQueueKey(priority int) string {
	if priority >= 8 {
		return "tickr:queue:high"
	} else if priority >= 5 {
		return "tickr:queue:medium"
	} else {
		return "ticker:queue:low"
	}
}

func (q *RedisQueue) Dequeue(ctx context.Context) (*jobs.Job, error) {
	queues := []string{
		"tickr:queue:high",
		"tickr:queue:medium",
		"ticker:queue:low",
	}

	res, err := q.client.BRPop(ctx, time.Second*5, queues...).Result()

	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var job jobs.Job
	if err := json.Unmarshal([]byte(res[1]), &job); err != nil {
		return nil, err
	}
	return &job, nil
}
