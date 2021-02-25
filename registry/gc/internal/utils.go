//go:generate mockgen -package mocks -destination mocks/utils.go . Clock,Backoff

package internal

import (
	"time"

	"github.com/benbjohnson/clock"
)

// Clock represents the functions in the standard library time package.
type Clock interface {
	clock.Clock
}

// Backoff represents a back off generator.
type Backoff interface {
	// Reset resets the interval back to the initial retry interval.
	Reset()
	// NextBackOff calculates the next backoff interval.
	NextBackOff() time.Duration
}
