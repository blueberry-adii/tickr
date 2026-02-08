package jobs

import (
	"encoding/json"
	"time"

	"github.com/blueberry-adii/tickr/internal/enums"
)

/*
Holds JobID and ScheduledAt of the Job field
This is the structure of job to be stored in redis queues
*/
type RedisJob struct {
	JobID       int64     `json:"job_id"`
	ScheduledAt time.Time `json:"scheduledAt"`
}

/*
Job struct defines what fields each Job
must have
Payload is the data needed to execute the Job

This is the structure of job to be stored in MySQL
*/
type Job struct {
	ID      int64           `json:"id"`
	JobType string          `json:"jobtype"`
	Payload json.RawMessage `json:"payload"`

	Status      enums.Status `json:"status"`
	Attempt     int          `json:"attempt"`
	MaxAttempts int          `json:"maxAttempts"`

	ScheduledAt time.Time  `json:"scheduledAt"`
	CreatedAt   time.Time  `json:"createdAt"`
	StartedAt   *time.Time `json:"startedAt"`
	FinishedAt  *time.Time `json:"finishedAt"`

	LastError *string `json:"lastError"`
	WorkerID  *int    `json:"workerID"`
}
