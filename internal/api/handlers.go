package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/blueberry-adii/tickr/internal/jobs"
	"github.com/blueberry-adii/tickr/internal/queue"
	"github.com/google/uuid"
)

type response struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Data    any    `json:"data"`
	Success bool   `json:"success"`
}

type Handler struct {
	queue *queue.RedisQueue
}

func NewHandler(q *queue.RedisQueue) *Handler {
	return &Handler{
		queue: q,
	}
}

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

func (h *Handler) SubmitJob(w http.ResponseWriter, r *http.Request) {
	var body struct {
		JobType string          `json:"jobtype"`
		Payload json.RawMessage `json:"payload"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid Body Format", http.StatusBadRequest)
		return
	}

	job := jobs.Job{
		ID:        uuid.NewString(),
		JobType:   body.JobType,
		Payload:   body.Payload,
		Status:    "pending",
		CreatedAt: time.Now(),
	}

	h.queue.PushReadyQueue(r.Context(), &job)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response{
		Status:  http.StatusOK,
		Message: "Job Submitted!!!",
		Data:    nil,
		Success: true,
	})
}
