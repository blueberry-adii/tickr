package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/blueberry-adii/tickr/internal/enums"
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

/*
Get Job Method to get job by job ID from database
*/
func (r MySQLRepository) GetJob(ctx context.Context, jobID int64) (*jobs.Job, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT 
			id,
			job_type,
			payload,
			status,
			attempt,
			max_attempts,
			scheduled_at,
			created_at,
			started_at,
			finished_at,
			last_error,
			worker_id 
		FROM jobs WHERE id = ?`,
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
		log.Printf("%v", err)
		if err == sql.ErrNoRows {
			return nil, errors.New(fmt.Sprintf("job with id %v not found", jobID))
		}
		return nil, err
	}

	return &job, nil
}

/*
Update job method to update job by ID in the database
*/
func (r MySQLRepository) UpdateJob(ctx context.Context, job *jobs.Job) error {
	/*if job isnt in executing status, clear worker and set it to nil in job instance*/
	if job.Status != enums.Executing {
		job.WorkerID = nil
	}

	_, err := r.db.ExecContext(
		ctx,
		"UPDATE jobs SET status = ?, worker_id = ?, attempt = ?, started_at = ?, finished_at = ?, last_error = ?, result = ? WHERE id = ?",
		job.Status,
		job.WorkerID,
		job.Attempt,
		job.StartedAt,
		job.FinishedAt,
		job.LastError,
		job.Result,
		job.ID,
	)
	if err != nil {
		return err
	}

	return nil
}

/*
Method to get the list of pending jobs including jobs which are in retrying state
*/
func (r MySQLRepository) GetPendingJobs(ctx context.Context) ([]jobs.RedisJob, error) {
	rows, err := r.db.QueryContext(
		ctx,
		"SELECT id, scheduled_at FROM jobs WHERE status IN ('pending', 'retrying')",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []jobs.RedisJob

	for rows.Next() {
		var job jobs.RedisJob
		if err := rows.Scan(&job.JobID, &job.ScheduledAt); err != nil {
			return nil, err
		}
		res = append(res, job)
	}

	return res, nil
}
