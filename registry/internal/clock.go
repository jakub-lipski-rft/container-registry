//go:generate mockgen -package mocks -destination mocks/clock.go . Clock

package internal

import (
	"github.com/benbjohnson/clock"
)

// Clock represents the functions in the standard library time package.
type Clock interface {
	clock.Clock
}
