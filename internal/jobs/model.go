package jobs

import (
	"encoding/json"
	"time"

	"github.com/blueberry-adii/tickr/internal/enums"
)

/*
Job struct defines what fields each Job
must have
Payload is the data needed to execute the Job
*/
type Job struct {
	ID        int64           `json:"id"`
	JobType   string          `json:"jobtype"`
	Payload   json.RawMessage `json:"payload"`
	Status    enums.Status    `json:"status"`
	CreatedAt time.Time       `json:"createdAt"`
}
