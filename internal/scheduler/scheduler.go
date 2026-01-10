package scheduler

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"time"

	"github.com/blueberry-adii/tickr/internal/jobs"
	"github.com/go-redis/redis/v8"
)

type Scheduler struct {
	redis *Redis
	JobCh chan *jobs.Job
	wqCh  chan int
}

func NewScheduler(r *Redis) *Scheduler {
	return &Scheduler{
		redis: r,
		JobCh: make(chan *jobs.Job),
		wqCh:  make(chan int),
	}
}

func (s *Scheduler) Run(ctx context.Context) {
	defer close(s.JobCh)
	defer close(s.wqCh)
	go s.PopReadyQueue(ctx)
	for {
		log.Printf("Scheduler Woke Up")
		nextExec, err := s.nextExecutionTime(ctx)

		var timer <-chan time.Time
		if err == nil {
			wait := time.Until(time.Unix(nextExec, 0))
			if wait < 0 {
				wait = 0
			}
			timer = time.After(wait)
		}

		select {
		case <-ctx.Done():
			return
		case <-s.wqCh:
			continue
		case <-timer:
			jobs, _ := s.PopWaitingQueue(ctx)
			for _, job := range jobs {
				s.PushReadyQueue(ctx, job)
			}
		case <-s.JobCh:
			go s.PopReadyQueue(ctx)
		}
	}
}

func (s *Scheduler) nextExecutionTime(ctx context.Context) (int64, error) {
	res, err := s.redis.client.ZRangeWithScores(
		ctx,
		"tickr:queue:waiting",
		0,
		0,
	).Result()

	if err != nil || len(res) == 0 {
		return 0, redis.Nil
	}

	return int64(res[0].Score), nil
}

func (s *Scheduler) PushReadyQueue(ctx context.Context, job *jobs.Job) error {
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}

	return s.redis.client.LPush(ctx, "tickr:queue:ready", data).Err()
}

func (s *Scheduler) PopReadyQueue(ctx context.Context) (*jobs.Job, error) {
	res, err := s.redis.client.BRPop(ctx, 0, "tickr:queue:ready").Result()

	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var job *jobs.Job = new(jobs.Job)
	if err := json.Unmarshal([]byte(res[1]), job); err != nil {
		return nil, err
	}

	s.JobCh <- job
	return job, nil
}

func (s *Scheduler) PushWaitingQueue(ctx context.Context, job *jobs.Job, delay int) error {
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}

	executeAt := time.Now().Add(time.Second * time.Duration(delay)).Unix()

	err = s.redis.client.ZAdd(ctx, "tickr:queue:waiting", &redis.Z{
		Score:  float64(executeAt),
		Member: data,
	}).Err()

	if err == nil {
		select {
		case s.wqCh <- 1:
		default:
		}
	}

	return err
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
