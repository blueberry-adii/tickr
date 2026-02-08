package enums

type Status string

const (
	Pending   Status = "pending"
	Executing Status = "executing"
	Completed Status = "completed"
	Failed    Status = "failed"
	Retrying  Status = "retrying"
)
