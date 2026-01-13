package enums

type Status string

const (
	Waiting   Status = "waiting"
	Ready     Status = "ready"
	Executing Status = "executing"
	Completed Status = "completed"
	Failed    Status = "failed"
)
