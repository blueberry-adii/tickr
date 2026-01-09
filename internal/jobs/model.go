package jobs

import (
	"encoding/json"
	"time"
)

type Job struct {
	ID          int64
	JobID       string
	JobType     string
	Payload     json.RawMessage
	Status      string
	Priority    int
	Attempts    int
	MaxAttempts int
	CreatedAt   time.Time
}
