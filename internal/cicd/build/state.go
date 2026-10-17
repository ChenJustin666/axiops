package build

import (
	"errors"
	"fmt"
)

var (
	ErrRunNotFound       = errors.New("build run not found")
	ErrInvalidTransition = errors.New("invalid build status transition")
)

const (
	StatusQueued      = "queued"
	StatusDispatching = "dispatching"
	StatusRunning     = "running"
	StatusCancelling  = "cancelling"
	StatusSuccess     = "success"
	StatusFailed      = "failed"
	StatusTimeout     = "timeout"
	StatusCancelled   = "cancelled"
)

var allowedTransitions = map[string]map[string]struct{}{
	StatusQueued: {
		StatusDispatching: {},
		StatusCancelling:  {},
		StatusFailed:      {},
	},
	StatusDispatching: {
		StatusRunning:    {},
		StatusCancelling: {},
		StatusFailed:     {},
	},
	StatusRunning: {
		StatusSuccess:    {},
		StatusFailed:     {},
		StatusTimeout:    {},
		StatusCancelling: {},
	},
	StatusCancelling: {
		StatusCancelled: {},
		StatusFailed:    {},
	},
}

var validStatuses = map[string]struct{}{
	StatusQueued: {}, StatusDispatching: {}, StatusRunning: {}, StatusCancelling: {},
	StatusSuccess: {}, StatusFailed: {}, StatusTimeout: {}, StatusCancelled: {},
}

func CanTransition(from, to string) bool {
	if _, ok := validStatuses[from]; !ok {
		return false
	}
	if _, ok := validStatuses[to]; !ok {
		return false
	}
	if from == to {
		return true
	}
	_, ok := allowedTransitions[from][to]
	return ok
}

func ValidateTransition(from, to string) error {
	if CanTransition(from, to) {
		return nil
	}
	return fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, from, to)
}

func IsTerminalStatus(status string) bool {
	switch status {
	case StatusSuccess, StatusFailed, StatusTimeout, StatusCancelled:
		return true
	default:
		return false
	}
}
