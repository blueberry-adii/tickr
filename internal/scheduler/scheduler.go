package scheduler

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/blueberry-adii/tickr/internal/database"
	"github.com/blueberry-adii/tickr/internal/jobs"
	"github.com/go-redis/redis/v8"
)

type Scheduler struct {
	recovering int32
	Repository database.Repository
	redis      *Redis
	JobCh      chan *jobs.RedisJob
	wqCh       chan int
}

func NewScheduler(r *Redis, repo database.Repository) *Scheduler {
	return &Scheduler{
		Repository: repo,
		redis:      r,
		JobCh:      make(chan *jobs.RedisJob),
		wqCh:       make(chan int),
	}
}

/*
Run is a scheduler method which runs an infinite loop and serves many purposes:
Checks whether Redis lost data/state, if true, runs recovery to refill Redis queues and Calculate the time when
the job with least delay needs to be moved from waiting queue to ready queue, and Calculates the waiting time till nextExec
*/
func (s *Scheduler) Run(ctx context.Context) {
	if s.redisStateLost(ctx) {
		log.Println("important: redis state missing, rebuilding from MySQL")
		s.recoverFromMySQL(ctx)
	}
	defer close(s.JobCh)
	defer close(s.wqCh)
	go s.PopReadyQueue(ctx)
	for {
		log.Printf("scheduler idle")
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
			log.Printf("killing scheduler")
			return
		case <-s.wqCh:
			log.Printf("new job in waiting queue")
			continue
		case <-timer:
			jobs, _ := s.PopWaitingQueue(ctx)
			for _, job := range jobs {
				log.Printf("moving job %v from waiting to ready queue", job.JobID)
				s.PushReadyQueue(ctx, job)
			}
		}
	}
}

/*
Calculates the time when the least delayed job in waiting queue needs to be
moved from waiting queue to ready queue
*/
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

/*
Pushes job into Ready queue
*/
func (s *Scheduler) PushReadyQueue(ctx context.Context, job *jobs.RedisJob) error {
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}

	return s.redis.client.LPush(ctx, "tickr:queue:ready", data).Err()
}

/*
Pops job from ready queue and put it into
Scheduler's job channel.
runs an infinite for loop, which stops when context is cancelled
*/
func (s *Scheduler) PopReadyQueue(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		res, err := s.redis.client.BRPop(ctx, 0, "tickr:queue:ready").Result()

		if err == redis.Nil {
			continue
		}
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("error popping from ready queue: %v", err)

			s.watchRedis(ctx)

			if s.redisStateLost(ctx) {
				s.triggerRecovery()
			}

			time.Sleep(time.Second)
			continue
		}

		var job *jobs.RedisJob = new(jobs.RedisJob)
		if err := json.Unmarshal([]byte(res[1]), job); err != nil {
			log.Printf("error unmarshalling job: %v", err)
			continue
		}

		select {
		case s.JobCh <- job:
		case <-ctx.Done():
			return
		}
	}
}

/*
Pushes a job in waiting queue, with duration the job stays in waiting queue
*/
func (s *Scheduler) PushWaitingQueue(ctx context.Context, job *jobs.RedisJob) error {
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}

	err = s.redis.client.ZAdd(ctx, "tickr:queue:waiting", &redis.Z{
		Score:  float64(job.ScheduledAt.Unix()),
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

/*
Fetches all the jobs from waiting queue which have exceeded their waiting time
iterate over each job's json data and unmarshal it into Go struct and
push it into readyJobs slice, and remove from Redis set (waiting queue)
*/
func (s *Scheduler) PopWaitingQueue(ctx context.Context) ([]*jobs.RedisJob, error) {
	now := time.Now().Unix()

	res, err := s.redis.client.ZRangeByScore(ctx, "tickr:queue:waiting", &redis.ZRangeBy{
		Min: "-inf",
		Max: strconv.FormatInt(now, 10),
	}).Result()

	if err != nil || len(res) == 0 {
		return nil, err
	}

	var readyJobs []*jobs.RedisJob

	for _, item := range res {
		var job *jobs.RedisJob
		if err := json.Unmarshal([]byte(item), &job); err != nil {
			continue
		}

		readyJobs = append(readyJobs, job)

		s.redis.client.ZRem(ctx, "tickr:queue:waiting", item)
	}

	return readyJobs, nil
}

/*
checks whether redis lost state/data after crash

tickr:redis:epoch is a key which is assigned to redis in the start of the program,
if it is missing -> redis state lost,
return true otherwise false
*/
func (s *Scheduler) redisStateLost(ctx context.Context) bool {
	exists, err := s.redis.client.Exists(ctx, "tickr:redis:epoch").Result()
	if err != nil {
		return false
	}
	return exists == 0
}

/*
sets recovering state to 1, to let other recovery functions to know
that recovery is ongoing
*/
func (s *Scheduler) triggerRecovery() {
	if atomic.CompareAndSwapInt32(&s.recovering, 0, 1) {
		go func() {
			defer atomic.StoreInt32(&s.recovering, 0)
			s.recoverFromMySQL(context.Background())
		}()
	}
}

/*
fetches all pending jobs from mysql and
pushes them back onto waiting queue
*/
func (s *Scheduler) recoverFromMySQL(ctx context.Context) {
	log.Println("redis state lost, rebuilding queues")

	jobs, err := s.Repository.GetPendingJobs(ctx)
	if err != nil {
		log.Printf("recovery failed: %v", err)
		return
	}

	for _, job := range jobs {
		s.PushWaitingQueue(ctx, &job)
	}

	s.redis.client.Set(ctx, "tickr:redis:epoch", time.Now().Unix(), 0)
}

/*
constantly pings redis to check whether it's active,
if yes, then send a signal to scheduler's wqCh to start executing recovered
waiting queue jobs and return from this function, otherwise repeat the loop every
second
*/
func (s *Scheduler) watchRedis(ctx context.Context) {
	for {
		if err := s.redis.client.Ping(ctx).Err(); err == nil {
			select {
			case s.wqCh <- 1:
			default:
			}
			break
		}
		log.Printf("err: redis connection inactive!!")
		time.Sleep(time.Second)
	}
}

func (s *Scheduler) Jobs() <-chan *jobs.RedisJob {
	return s.JobCh
}

func (s *Scheduler) GetJob(ctx context.Context, jobID int64) (*jobs.Job, error) {
	return s.Repository.GetJob(ctx, jobID)
}

func (s *Scheduler) SaveJob(ctx context.Context, job jobs.Job) (int64, error) {
	return s.Repository.SaveJob(ctx, job)
}

func (s *Scheduler) UpdateJob(ctx context.Context, job *jobs.Job) error {
	return s.Repository.UpdateJob(ctx, job)
}
