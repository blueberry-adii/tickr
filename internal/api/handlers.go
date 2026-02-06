package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/blueberry-adii/tickr/internal/database"
	"github.com/blueberry-adii/tickr/internal/enums"
	"github.com/blueberry-adii/tickr/internal/jobs"
	"github.com/blueberry-adii/tickr/internal/scheduler"
)

/*
response struct: HTTP Response must always be
returned in this format
*/
type response struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Data    any    `json:"data"`
	Success bool   `json:"success"`
}

/*
Handler Struct which depends on scheduler and repository
through dependency injection
*/
type Handler struct {
	scheduler  *scheduler.Scheduler
	repository *database.MySQLRepository
}

/*Handler Constructor*/
func NewHandler(s *scheduler.Scheduler, r *database.MySQLRepository) *Handler {
	return &Handler{
		scheduler:  s,
		repository: r,
	}
}

/*API Health Endpoint Handler*/
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response{
		Status:  http.StatusOK,
		Message: "REST API Up and Working!!!",
		Data:    nil,
		Success: true,
	})
}

/*API Endpoint to submit jobs*/
func (h *Handler) SubmitJob(w http.ResponseWriter, r *http.Request) {
	/*http request body struct*/
	var body struct {
		JobType string          `json:"jobtype"`
		Payload json.RawMessage `json:"payload"`
		Delay   int             `json:"delay"`
	}

	/*Decode JSON Req Body into variable body*/
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid Body Format", http.StatusBadRequest)
		return
	}

	now := time.Now()
	scheduledAt := now.Add(time.Duration(body.Delay) * time.Second)

	/*Create a New Job with the given data from request body*/
	job := jobs.Job{
		JobType:     body.JobType,
		Payload:     body.Payload,
		Status:      enums.Pending,
		Attempt:     0,
		MaxAttempts: 3,
		CreatedAt:   now,
		ScheduledAt: scheduledAt,
	}

	/*Save Job in MySQL Database using Repository*/
	var err error
	job.ID, err = h.repository.SaveJob(r.Context(), job)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	/*
		If job is scheduled/given delay, push the job into waiting queue,
		otherwise push into ready queue
	*/
	if body.Delay > 0 {
		h.scheduler.PushWaitingQueue(r.Context(), job.ID, job.ScheduledAt)
	} else {
		h.scheduler.PushReadyQueue(r.Context(), &job)
	}

	/*
		Set response content type into application json
		response status code - 200 (OK)
		encode Go struct and send response in json format
	*/
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response{
		Status:  http.StatusOK,
		Message: "Job Submitted!!!",
		Data: map[string]any{
			"jobID":       job.ID,
			"status":      job.Status,
			"scheduledAt": job.ScheduledAt,
		},
		Success: true,
	})
}
