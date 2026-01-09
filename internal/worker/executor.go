package worker

import (
	"encoding/json"
	"log"
	"time"

	"github.com/blueberry-adii/tickr/internal/jobs"
)

type Executor struct {
	worker *Worker
}

func (e *Executor) ExecuteJob(job *jobs.Job) {
	switch job.JobType {
	case "email":
		e.handleEmail(job)
	case "report":
		e.handleReport(job)
	default:
		log.Printf("Unrecognized Job")
	}
}

func (e *Executor) handleEmail(job *jobs.Job) {
	var email struct {
		To   string `json:"to"`
		From string `json:"from"`
		Body string `json:"body"`
	}

	if err := json.Unmarshal([]byte(job.Payload), &email); err != nil {
		log.Printf("Invalid Email Format: %v", err)
		return
	}

	log.Printf("Sending email from %s to %s", email.From, email.To)
	time.Sleep(time.Second * 5)
	log.Printf("Sent Email: %s", email.Body)
}

func (e *Executor) handleReport(job *jobs.Job) {

}
