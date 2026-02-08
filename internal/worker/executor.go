package worker

import (
	"encoding/json"
	"errors"
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
func (e *Executor) ExecuteJob(job *jobs.Job) error {
	switch job.JobType {
	case "email":
		return e.handleEmail(job)
	case "report":
		return e.handleReport(job)
	default:
		return errors.New("unrecognized job")
	}
}

/*
Method to simulate email sending
*/
func (e *Executor) handleEmail(job *jobs.Job) error {
	var email struct {
		To   string `json:"to"`
		From string `json:"from"`
		Body string `json:"body"`
	}

	if err := json.Unmarshal([]byte(job.Payload), &email); err != nil {
		log.Printf("invalid email format: %v", err)
		return err
	}

	log.Printf("sending email from %s to %s", email.From, email.To)
	time.Sleep(time.Second * 5)
	log.Printf("sent email: %s", email.Body)
	return nil
}

/*
Method to simulate report handling
*/
func (e *Executor) handleReport(job *jobs.Job) error {
	var report struct {
		Title string `json:"title"`
		Body  string `json:"body"`
		Time  int    `json:"time"`
	}

	if err := json.Unmarshal([]byte(job.Payload), &report); err != nil {
		log.Printf("invalid report format: %v", err)
		return err
	}

	log.Printf("scheduled report for %d seconds", report.Time)
	time.Sleep(time.Second * time.Duration(report.Time))
	log.Printf("Title: %s | Body: %s", report.Title, report.Body)
	return nil
}
