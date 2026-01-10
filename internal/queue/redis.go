package queue

import (
	"context"
	"encoding/json"
	"strconv"
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

func (q *RedisQueue) PushReadyQueue(ctx context.Context, job *jobs.Job) error {
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}

	return q.client.LPush(ctx, "tickr:queue:ready", data).Err()
}

func (q *RedisQueue) PopReadyQueue(ctx context.Context) (*jobs.Job, error) {
	res, err := q.client.BRPop(ctx, time.Second*5, "tickr:queue:ready").Result()

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

func (q *RedisQueue) PushWaitingQueue(ctx context.Context, job *jobs.Job, delay int) error {
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}

	executeAt := time.Now().Add(time.Second * time.Duration(delay)).Unix()

	return q.client.ZAdd(ctx, "tickr:queue:waiting", &redis.Z{
		Score:  float64(executeAt),
		Member: data,
	}).Err()
}

func (q *RedisQueue) PopWaitingQueue(ctx context.Context) ([]*jobs.Job, error) {
	now := time.Now().Unix()

	res, err := q.client.ZRangeByScore(ctx, "tickr:queue:waiting", &redis.ZRangeBy{
		Min: "-inf",
		Max: strconv.FormatInt(now, 10),
	}).Result()

	if err != nil || len(res) == 0 {
		return nil, err
	}

	var readyJobs []*jobs.Job

	for _, item := range res {
		var job *jobs.Job
		if err := json.Unmarshal([]byte(item), &job); err != nil {
			continue
		}

		readyJobs = append(readyJobs, job)

		q.client.ZRem(ctx, "tickr:queue:waiting", item)
	}

	return readyJobs, nil
}
