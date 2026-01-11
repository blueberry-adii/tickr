package jobs

import (
	"encoding/json"
	"time"
)

/*
Job struct defines what fields each Job
must have
Payload is the data needed to execute the Job
*/
type Job struct {
	ID        string          `json:"id"`
	JobType   string          `json:"jobtype"`
	Payload   json.RawMessage `json:"payload"`
	Status    string          `json:"status"`
	CreatedAt time.Time       `json:"createdAt"`
}
