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

/*
Scheduler depends on redis client,
has a job channel to notify when a job is pushed to ready queue,
has a waiting queue channel to notify when a new job is pushed to waiting queue,
has a recovering field, to let other goroutines know redis is in recovering state,
injects database repository to access DB Methods
*/
type Scheduler struct {
	recovering int32
	Repository *database.MySQLRepository
	redis      *Redis
	JobCh      chan *jobs.RedisJob
	wqCh       chan int
}

/*
Scheduler constructor, accept redis client,
create a job channel and a wq channel
*/
func NewScheduler(r *Redis, repo *database.MySQLRepository) *Scheduler {
	return &Scheduler{
		Repository: repo,
		redis:      r,
		JobCh:      make(chan *jobs.RedisJob),
		wqCh:       make(chan int),
	}
}

/*
Run is a scheduler method which runs an infinite loop
and serves many purposes:
1. closes all the scheduler channels when scheduler dies
2. run PopReadyQueue method as a goroutine
3. Block the infinite for loop and Listen to multiple channels
*/
func (s *Scheduler) Run(ctx context.Context) {
	/*Checks whether Redis lost data/state, if true, runs recovery to refill Redis queues*/
	if s.redisStateLost(ctx) {
		log.Println("important: redis state missing, rebuilding from MySQL")
		s.recoverFromMySQL(ctx)
	}
	defer close(s.JobCh)
	defer close(s.wqCh)
	go s.PopReadyQueue(ctx)
	for {
		log.Printf("scheduler idle")
		/*
			Calculate the time when the job with least delay
			needs to be moved from waiting queue to ready queue
		*/
		nextExec, err := s.nextExecutionTime(ctx)

		/*
			define a timer channel (receiving only)
		*/
		var timer <-chan time.Time
		if err == nil {
			/*
				Calculate the waiting time till nextExec
			*/
			wait := time.Until(time.Unix(nextExec, 0))
			/*If negative, set waiting time to 0 (immediate execution)*/
			if wait < 0 {
				wait = 0
			}

			/*
				Sets timer a channel which signals when the wait duration is elapsed
			*/
			timer = time.After(wait)
		}

		/*
			1. On context cancel, returns from the function, killing the scheduler
			2. when waiting channel signals, there is a new job in waiting queue,
				scheduler moves on to next iteration, to recalculate waiting time
				for the least delayed job
			3. when timer signals after least delayed job's waiting duration is elapsed
				it pops all the jobs from waiting queue and pushes them one by one in
				ready queue for the workers
		*/
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
method to Push job in ready queue
*/
func (s *Scheduler) PushReadyQueue(ctx context.Context, job *jobs.RedisJob) error {
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}

	return s.redis.client.LPush(ctx, "tickr:queue:ready", data).Err()
}

/*
Method to pop job from ready queue and put it into
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

		/*
			as long as context isn't cancelled, this block is executed every loop
			BRPop is a blocking call, which waits until there is a job pushed into ready queue
		*/
		res, err := s.redis.client.BRPop(ctx, 0, "tickr:queue:ready").Result()

		/*
			If job is nil then skip and move to next iteration
		*/
		if err == redis.Nil {
			continue
		}
		if err != nil {
			/*
				If context cancelled while waiting on BRPop,
				return from this method
			*/
			if ctx.Err() != nil {
				return
			}
			log.Printf("error popping from ready queue: %v", err)

			/*Calls watchRedis to wait till redis reconnects*/
			s.watchRedis(ctx)

			/*Runs recovery if redis state lost*/
			if s.redisStateLost(ctx) {
				s.triggerRecovery()
			}

			time.Sleep(time.Second)
			continue
		}

		/*
			Create a new empty job, and fill all the fields
			from job's json data fetched from redis into empty job
		*/
		var job *jobs.RedisJob = new(jobs.RedisJob)
		if err := json.Unmarshal([]byte(res[1]), job); err != nil {
			log.Printf("error unmarshalling job: %v", err)
			continue
		}

		/*
			Wait until either context cancelled or jobChannel is empty
			when jobChannel empty, put the job into jobChannel for worker
			to fetch from

			**IMPORTANT**: For example, all 5 workers have taken jobs and are busy,
			a new job is popped from ready queue to job channel
			No worker is idle to take the job from job channel.
			Then this all the other jobs wait until the job in job channel is taken by
			an idle worker
		*/
		select {
		case s.JobCh <- job:
		case <-ctx.Done():
			return
		}
	}
}

/*
Method to push a job in waiting queue, with duration the job stays in waiting queue
*/
func (s *Scheduler) PushWaitingQueue(ctx context.Context, job *jobs.RedisJob) error {
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}

	/*Adds the job into Redis ZSet sorted according to their time of execution*/
	err = s.redis.client.ZAdd(ctx, "tickr:queue:waiting", &redis.Z{
		Score:  float64(job.ScheduledAt.Unix()),
		Member: data,
	}).Err()

	/*
		If no error, then wait until the waiting queue channel is empty,
		when wqCh is empty, notify the scheduler that the waiting queue has been
		updated with a new job in the queue
	*/
	if err == nil {
		select {
		case s.wqCh <- 1:
		default:
		}
	}

	return err
}

func (s *Scheduler) PopWaitingQueue(ctx context.Context) ([]*jobs.RedisJob, error) {
	now := time.Now().Unix()

	/*
		Fetches all the jobs from waiting queue which have exceeded their waiting time
	*/
	res, err := s.redis.client.ZRangeByScore(ctx, "tickr:queue:waiting", &redis.ZRangeBy{
		Min: "-inf",
		Max: strconv.FormatInt(now, 10),
	}).Result()

	/*
		Return nil when there is an error or there are no jobs which have exceeded
		their waiting time
	*/
	if err != nil || len(res) == 0 {
		return nil, err
	}

	var readyJobs []*jobs.RedisJob

	/*
		iterate over each job's json data and unmarshal it into Go struct and
		push it into readyJobs slice, and remove from Redis set (waiting queue)
	*/
	for _, item := range res {
		var job *jobs.RedisJob
		if err := json.Unmarshal([]byte(item), &job); err != nil {
			continue
		}

		readyJobs = append(readyJobs, job)

		s.redis.client.ZRem(ctx, "tickr:queue:waiting", item)
	}

	/*Return all the readyJobs to be pushed into ready queue*/
	return readyJobs, nil
}

/*Function which checks whether redis lost state/data after crash*/
func (s *Scheduler) redisStateLost(ctx context.Context) bool {
	/*
		tickr:redis:epoch is a key which is assigned to redis in the start of the program,
		if it is missing -> redis state lost,
		return true otherwise false
	*/
	exists, err := s.redis.client.Exists(ctx, "tickr:redis:epoch").Result()
	if err != nil {
		return false
	}
	return exists == 0
}

/*
Function which sets recovering state to 1, to let other recovery functions to know
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
Function which fetches all pending jobs from mysql and
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
Function that constantly pings redis to check whether it's active,
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
