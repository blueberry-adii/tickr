package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/blueberry-adii/tickr/internal/jobs"
	"github.com/blueberry-adii/tickr/internal/worker"
)

func TestExecutor(t *testing.T) {
	successServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer successServer.Close()

	errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer errorServer.Close()

	tests := []struct {
		name          string
		job           *jobs.Job
		expectedError bool
	}{
		{
			name: "valid email execution",
			job: &jobs.Job{
				JobType: "email",
				Payload: []byte(`{"to":"luffy","from":"aditya","body":"hello world"}`),
			},
			expectedError: false,
		},
		{
			name: "invalid email payload",
			job: &jobs.Job{
				JobType: "email",
				Payload: []byte(`not json`),
			},
			expectedError: true,
		},
		{
			name: "valid http execution",
			job: &jobs.Job{
				JobType: "http",
				Payload: []byte(`{"url":"` + successServer.URL + `","method":"GET"}`),
			},
			expectedError: false,
		},
		{
			name: "http non-2xx response returns error",
			job: &jobs.Job{
				JobType: "http",
				Payload: []byte(`{"url":"` + errorServer.URL + `","method":"GET"}`),
			},
			expectedError: true,
		},
		{
			name: "invalid http payload",
			job: &jobs.Job{
				JobType: "http",
				Payload: []byte(`not json`),
			},
			expectedError: true,
		},
		{
			name: "unknown job type returns error",
			job: &jobs.Job{
				JobType: "unknown",
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := worker.NewExecutor()

			err := e.ExecuteJob(tt.job)

			if err != nil && !tt.expectedError {
				t.Errorf("expected no error, got: %v", err)
			}
			if err == nil && tt.expectedError {
				t.Errorf("expected error, got nil")
			}
			if tt.job.Result == nil {
				t.Errorf("expected job.Result to be set, got nil")
			}
		})
	}
}
