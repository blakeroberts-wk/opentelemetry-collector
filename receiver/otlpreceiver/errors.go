package otlpreceiver

import (
	"time"

	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/errors"
)

var ErrRateLimited error = &errors.ErrorRateLimited{}

func NewErrRateLimited(backoff time.Duration) error {
	return &errors.ErrorRateLimited{
		Backoff: backoff,
	}
}
