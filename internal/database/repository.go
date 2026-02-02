package database

import (
	"context"
	"database/sql"

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
		"INSERT INTO jobs (job_type, payload, status, created_at) VALUES (?, ?, ?);",
		job.JobType,
		job.Payload,
		job.Status,
		job.CreatedAt,
	)

	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}
