package worker

import (
	"encoding/json"
	"log"
	"time"

	"github.com/blueberry-adii/tickr/internal/jobs"
)

/*
Executor struct needs a worker assigned through
dependency injection
*/
type Executor struct {
	worker *Worker
}

/*
ExecuteJob is a method which runs switch case conditions
to decide which job handler to run based on job type
*/
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

/*
Method to simulate email sending
*/
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

/*
Method to simulate report handling
*/
func (e *Executor) handleReport(job *jobs.Job) {
	var report struct {
		Title string `json:"title"`
		Body  string `json:"body"`
		Time  int    `json:"time"`
	}

	if err := json.Unmarshal([]byte(job.Payload), &report); err != nil {
		log.Printf("Invalid Report Format: %v", err)
		return
	}

	log.Printf("Scheduled report for %d seconds", report.Time)
	time.Sleep(time.Second * time.Duration(report.Time))
	log.Printf("Title: %s | Body: %s", report.Title, report.Body)
}
