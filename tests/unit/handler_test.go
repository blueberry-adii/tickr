package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/blueberry-adii/tickr/internal/api"
	"github.com/blueberry-adii/tickr/internal/jobs"
	"github.com/blueberry-adii/tickr/internal/scheduler"
)

type MockScheduler struct {
	readyQueue   []*jobs.RedisJob
	waitingQueue []*jobs.RedisJob
}

func (q *MockScheduler) SaveJob(ctx context.Context, job jobs.Job) (int64, error) {
	return 0, nil
}
func (q *MockScheduler) PushWaitingQueue(ctx context.Context, job *jobs.RedisJob) error {
	q.waitingQueue = append(q.waitingQueue, job)
	return nil
}
func (q *MockScheduler) PushReadyQueue(ctx context.Context, job *jobs.RedisJob) error {
	q.readyQueue = append(q.readyQueue, job)
	return nil
}

var _ scheduler.Queue = &MockScheduler{}

func TestSubmitJobHandler(t *testing.T) {

	tests := []struct {
		name               string
		body               string
		expectedStatusCode int
		expectedWaitingLen int
		expectedReadyLen   int
	}{
		{
			name:               "empty body",
			body:               "",
			expectedStatusCode: http.StatusBadRequest,
			expectedWaitingLen: 0,
			expectedReadyLen:   0,
		},
		{
			name:               "Invalid JSON",
			body:               `"jobtype":"email", "payload":""`,
			expectedStatusCode: http.StatusBadRequest,
			expectedWaitingLen: 0,
			expectedReadyLen:   0,
		},
		{
			name:               "Non Delayed Job",
			body:               `{"jobtype":"email", "payload":""}`,
			expectedStatusCode: http.StatusOK,
			expectedWaitingLen: 0,
			expectedReadyLen:   1,
		},
		{
			name:               "Job Delayed by few seconds",
			body:               `{"jobtype":"email", "payload":"", "delay":5}`,
			expectedStatusCode: http.StatusOK,
			expectedWaitingLen: 1,
			expectedReadyLen:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &MockScheduler{}
			handler := api.NewHandler(s)
			req := httptest.NewRequest(http.MethodPost, "/api/v2/jobs", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()

			route := http.HandlerFunc(handler.SubmitJob)
			route.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatusCode {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.expectedStatusCode)
			}

			if len(s.readyQueue) != tt.expectedReadyLen {
				t.Errorf("expected %d job in ready queue, got %d", tt.expectedReadyLen, len(s.readyQueue))
			}

			if len(s.waitingQueue) != tt.expectedWaitingLen {
				t.Errorf("expected %d job in waiting queue, got %d", tt.expectedWaitingLen, len(s.waitingQueue))
			}
		})
	}
}
