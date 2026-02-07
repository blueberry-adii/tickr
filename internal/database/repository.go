package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/blueberry-adii/tickr/internal/jobs"
)

/*MySQLRepository depends on database DI*/
type MySQLRepository struct {
	db *sql.DB
}

/*MySQLRepository constructor*/
func NewMySQLRepository(db *sql.DB) *MySQLRepository {
	return &MySQLRepository{
		db,
	}
}

/*
Save Job Method to save the job in database and
return job ID
*/
func (r MySQLRepository) SaveJob(ctx context.Context, job jobs.Job) (int64, error) {
	res, err := r.db.ExecContext(
		ctx,
		"INSERT INTO jobs (job_type, payload, status, attempt, max_attempts, created_at, scheduled_at) VALUES (?, ?, ?, ?, ?, ?, ?);",
		job.JobType,
		job.Payload,
		job.Status,
		job.Attempt,
		job.MaxAttempts,
		job.CreatedAt,
		job.ScheduledAt,
	)

	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

func (r MySQLRepository) GetJob(ctx context.Context, jobID int64) (*jobs.Job, error) {
	row := r.db.QueryRowContext(
		ctx,
		"SELECT * FROM jobs WHERE job_id = ?",
		jobID,
	)

	var job jobs.Job
	err := row.Scan(
		&job.ID,
		&job.JobType,
		&job.Payload,

		&job.Status,
		&job.Attempt,
		&job.MaxAttempts,

		&job.ScheduledAt,
		&job.CreatedAt,
		&job.StartedAt,
		&job.FinishedAt,

		&job.LastError,
		&job.WorkerID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New(fmt.Sprintf("job with id %v not found", jobID))
		}
		return nil, err
	}

	return &job, nil
}
