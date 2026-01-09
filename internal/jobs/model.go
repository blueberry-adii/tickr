package jobs

import (
	"encoding/json"
	"time"
)

type Job struct {
	ID        string          `json:"id"`
	JobType   string          `json:"jobtype"`
	Payload   json.RawMessage `json:"payload"`
	Status    string          `json:"status"`
	CreatedAt time.Time       `json:"createdAt"`
}
