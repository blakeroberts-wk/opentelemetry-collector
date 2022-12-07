package errors

import "time"

type ErrorRateLimited struct {
	Backoff time.Duration
}

func (e *ErrorRateLimited) Error() string {
	return "Too Many Requests"
}
