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

/*
Scheduler depends on redis client,
has a job channel to notify when a job is pushed to ready queue,
has a waiting queue channel to notify when a new job is pushed to waiting queue
*/
type Scheduler struct {
	redis *Redis
	JobCh chan *jobs.Job
	wqCh  chan int
}

/*
Scheduler constructor, accept redis client,
create a job channel and a wq channel
*/
func NewScheduler(r *Redis) *Scheduler {
	return &Scheduler{
		redis: r,
		JobCh: make(chan *jobs.Job),
		wqCh:  make(chan int),
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
	defer close(s.JobCh)
	defer close(s.wqCh)
	go s.PopReadyQueue(ctx)
	for {
		log.Printf("Scheduler Idle")
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
			log.Printf("Killing Scheduler")
			return
		case <-s.wqCh:
			log.Printf("New Job in WQ")
			continue
		case <-timer:
			jobs, _ := s.PopWaitingQueue(ctx)
			for _, job := range jobs {
				log.Printf("moving job %v from waiting to ready queue", job.ID)
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
func (s *Scheduler) PushReadyQueue(ctx context.Context, job *jobs.Job) error {
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
			log.Printf("Error popping from ready queue: %v", err)
			time.Sleep(time.Second)
			continue
		}

		/*
			Create a new empty job, and fill all the fields
			from job's json data fetched from redis into empty job
		*/
		var job *jobs.Job = new(jobs.Job)
		if err := json.Unmarshal([]byte(res[1]), job); err != nil {
			log.Printf("Error unmarshalling job: %v", err)
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
func (s *Scheduler) PushWaitingQueue(ctx context.Context, job *jobs.Job, delay int) error {
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}

	/*Calculates the time when the job needs to be moved from waiting queue to ready queue*/
	executeAt := time.Now().Add(time.Second * time.Duration(delay)).Unix()

	/*Adds the job into Redis ZSet sorted according to their delay*/
	err = s.redis.client.ZAdd(ctx, "tickr:queue:waiting", &redis.Z{
		Score:  float64(executeAt),
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

func (s *Scheduler) PopWaitingQueue(ctx context.Context) ([]*jobs.Job, error) {
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

	var readyJobs []*jobs.Job

	/*
		iterate over each job's json data and unmarshal it into Go struct and
		push it into readyJobs slice, and remove from Redis set (waiting queue)
	*/
	for _, item := range res {
		var job *jobs.Job
		if err := json.Unmarshal([]byte(item), &job); err != nil {
			continue
		}

		readyJobs = append(readyJobs, job)

		s.redis.client.ZRem(ctx, "tickr:queue:waiting", item)
	}

	/*Return all the readyJobs to be pushed into ready queue*/
	return readyJobs, nil
}
