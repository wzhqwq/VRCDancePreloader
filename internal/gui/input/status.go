package input

type Status string

const (
	StatusUnknown Status = "unknown"
	StatusOk      Status = "ok"
	StatusError   Status = "error"
	StatusTesting Status = "testing"
	StatusRunning Status = "running"
)
