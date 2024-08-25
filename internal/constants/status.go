package constants

type Status string

const (
	Requesting  Status = "requesting"
	Downloading Status = "downloading"
	Downloaded  Status = "downloaded"
	Failed      Status = "failed"
	Pending     Status = "pending"
	Playing     Status = "playing"
	Ended       Status = "ended"
	Ignored     Status = "ignored"
)
