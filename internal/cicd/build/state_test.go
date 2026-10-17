package build

import "testing"

func TestBuildStatusTransitions(t *testing.T) {
	tests := []struct {
		name    string
		from    string
		to      string
		allowed bool
	}{
		{name: "queue dispatch", from: StatusQueued, to: StatusDispatching, allowed: true},
		{name: "running succeeds", from: StatusRunning, to: StatusSuccess, allowed: true},
		{name: "cancel requested", from: StatusRunning, to: StatusCancelling, allowed: true},
		{name: "cancel confirmed", from: StatusCancelling, to: StatusCancelled, allowed: true},
		{name: "terminal immutable", from: StatusSuccess, to: StatusRunning, allowed: false},
		{name: "cannot skip cancellation acknowledgement", from: StatusRunning, to: StatusCancelled, allowed: false},
		{name: "unknown target", from: StatusQueued, to: "unknown", allowed: false},
		{name: "unknown idempotent transition", from: "unknown", to: "unknown", allowed: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanTransition(tt.from, tt.to); got != tt.allowed {
				t.Fatalf("CanTransition(%q, %q) = %v, want %v", tt.from, tt.to, got, tt.allowed)
			}
		})
	}
}

func TestTerminalStatus(t *testing.T) {
	for _, status := range []string{StatusSuccess, StatusFailed, StatusTimeout, StatusCancelled} {
		if !IsTerminalStatus(status) {
			t.Fatalf("status %q should be terminal", status)
		}
	}
	if IsTerminalStatus(StatusCancelling) {
		t.Fatal("cancelling must not be terminal")
	}
}
