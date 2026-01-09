package jobs

import (
	"encoding/json"
	"time"
)

type Job struct {
	ID        string
	JobType   string
	Payload   json.RawMessage
	Status    string
	CreatedAt time.Time
}
