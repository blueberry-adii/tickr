package scheduler

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/blueberry-adii/tickr/internal/jobs"
	"github.com/go-redis/redis/v8"
)

type Scheduler struct {
	redis *Redis
}

func NewScheduler(r *Redis) *Scheduler {
	return &Scheduler{
		redis: r,
	}
}

func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			jobs, _ := s.PopWaitingQueue(ctx)
			for _, job := range jobs {
				s.PushReadyQueue(ctx, job)
			}
		}
	}
}

func (s *Scheduler) PushReadyQueue(ctx context.Context, job *jobs.Job) error {
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}

	return s.redis.client.LPush(ctx, "tickr:queue:ready", data).Err()
}

func (s *Scheduler) PopReadyQueue(ctx context.Context) (*jobs.Job, error) {
	res, err := s.redis.client.BRPop(ctx, time.Second*5, "tickr:queue:ready").Result()

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

func (s *Scheduler) PushWaitingQueue(ctx context.Context, job *jobs.Job, delay int) error {
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}

	executeAt := time.Now().Add(time.Second * time.Duration(delay)).Unix()

	return s.redis.client.ZAdd(ctx, "tickr:queue:waiting", &redis.Z{
		Score:  float64(executeAt),
		Member: data,
	}).Err()
}

func (s *Scheduler) PopWaitingQueue(ctx context.Context) ([]*jobs.Job, error) {
	now := time.Now().Unix()

	res, err := s.redis.client.ZRangeByScore(ctx, "tickr:queue:waiting", &redis.ZRangeBy{
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

		s.redis.client.ZRem(ctx, "tickr:queue:waiting", item)
	}

	return readyJobs, nil
}
