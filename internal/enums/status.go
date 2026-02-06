package enums

type Status string

const (
	Pending   Status = "pending"
	Waiting   Status = "waiting"
	Ready     Status = "ready"
	Executing Status = "executing"
	Completed Status = "completed"
	Failed    Status = "failed"
)
