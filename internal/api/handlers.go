package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/blueberry-adii/tickr/internal/enums"
	"github.com/blueberry-adii/tickr/internal/jobs"
	"github.com/blueberry-adii/tickr/internal/scheduler"
)

/*
HTTP Response is always returned in this structure
*/
type response struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Data    any    `json:"data"`
	Success bool   `json:"success"`
}

/*
Handler Struct responsible for handling API Requests
*/
type Handler struct {
	scheduler scheduler.Queue
}

/*
Returns a new instance of Handler
*/
func NewHandler(s scheduler.Queue) *Handler {
	return &Handler{
		scheduler: s,
	}
}

/*
Returns Api Health status
*/
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

/*
Takes job from http request,
calculates time when job needs to be moved from waiting queue to ready queue,
creates a New Job with the given data from request body, saves job into Database
pushes the job onto waiting queue if delayed otherwise ready queue
*/
func (h *Handler) SubmitJob(w http.ResponseWriter, r *http.Request) {
	var body struct {
		JobType string          `json:"jobtype"`
		Payload json.RawMessage `json:"payload"`
		Delay   int             `json:"delay"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid Body Format", http.StatusBadRequest)
		return
	}

	now := time.Now()
	scheduledAt := now.Add(time.Duration(body.Delay) * time.Second)

	job := jobs.Job{
		JobType:     body.JobType,
		Payload:     body.Payload,
		Status:      enums.Pending,
		Attempt:     0,
		MaxAttempts: 3,
		CreatedAt:   now,
		ScheduledAt: scheduledAt,
	}

	var err error
	job.ID, err = h.scheduler.SaveJob(r.Context(), job)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	redisJob := &jobs.RedisJob{JobID: job.ID, ScheduledAt: scheduledAt}

	if body.Delay > 0 {
		h.scheduler.PushWaitingQueue(r.Context(), redisJob)
	} else {
		h.scheduler.PushReadyQueue(r.Context(), redisJob)
	}

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
