package jobs

import (
	"time"
)

type Job struct {
	ID        string    `json:"id"`
	JobType   string    `json:"jobtype"`
	Payload   string    `json:"payload"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}
