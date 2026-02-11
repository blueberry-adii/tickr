package worker

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
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
	case "http":
		return e.SendHttpRequest(job)
	default:
		return errors.New("unrecognized job")
	}
}

/*
Method to send an http request
*/
func (e *Executor) SendHttpRequest(job *jobs.Job) error {
	client := &http.Client{
		Timeout: time.Second * 10,
	}

	var request struct {
		Url     string          `json:"url"`
		Body    json.RawMessage `json:"body"`
		Headers json.RawMessage `json:"headers"`
		Method  string          `json:"method"`
	}

	var Obj struct {
		Data any `json:"data"`
	}

	defer func() {
		bytes, _ := json.Marshal(Obj)
		job.Result = bytes
	}()

	if err := json.Unmarshal([]byte(job.Payload), &request); err != nil {
		Obj.Data = "error: invalid http request"
		log.Printf("invalid http request format: %v", err)
		return err
	}

	req, _ := http.NewRequest(request.Method, request.Url, bytes.NewBuffer(request.Body))

	if len(request.Headers) > 0 {
		var headerMap map[string]string

		if err := json.Unmarshal(request.Headers, &headerMap); err != nil {
			Obj.Data = "error: failed to parse headers"
			return fmt.Errorf("failed to parse headers: %w", err)
		}

		for key, value := range headerMap {
			req.Header.Set(key, value)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		Obj.Data = err.Error()
		log.Printf("err: %v", err.Error())
		return err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		Obj.Data = err.Error()
		log.Printf("err: %v", err.Error())
		return err
	}

	Obj.Data = json.RawMessage(bodyBytes)

	return nil
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

	var Obj struct {
		Data string `json:"data"`
	}

	defer func() {
		bytes, _ := json.Marshal(Obj)
		job.Result = bytes
	}()

	if err := json.Unmarshal([]byte(job.Payload), &email); err != nil {
		Obj.Data = "error: invalid email"
		log.Printf("invalid email format: %v", err)
		return err
	}

	log.Printf("sending email from %s to %s", email.From, email.To)
	time.Sleep(time.Second * 5)
	log.Printf("sent email: %s", email.Body)
	Obj.Data = fmt.Sprintf("sent Email to %v successfully", email.To)

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

	var Obj struct {
		Data string `json:"data"`
	}

	defer func() {
		bytes, _ := json.Marshal(Obj)
		job.Result = bytes
	}()

	if err := json.Unmarshal([]byte(job.Payload), &report); err != nil {
		log.Printf("invalid report format: %v", err)
		Obj.Data = "error: invalid report format"
		return err
	}

	log.Printf("scheduled report for %d seconds", report.Time)
	time.Sleep(time.Second * time.Duration(report.Time))
	log.Printf("Title: %s | Body: %s", report.Title, report.Body)

	Obj.Data = "report successful"
	return nil
}
