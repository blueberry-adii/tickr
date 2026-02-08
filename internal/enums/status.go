package enums

type Status string

/*
Enums for different job statuses
*/
const (
	Pending   Status = "pending"
	Executing Status = "executing"
	Completed Status = "completed"
	Failed    Status = "failed"
	Retrying  Status = "retrying"
)
