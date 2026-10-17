package build

import (
	"testing"
	"time"
)

func TestRetryDelayHasMinimum(t *testing.T) {
	short := time.Millisecond
	if got := normalizeRetryAfter(short); got != time.Second {
		t.Fatalf("normalizeRetryAfter(%v) = %v, want one second", short, got)
	}
	long := 30 * time.Second
	if got := normalizeRetryAfter(long); got != long {
		t.Fatalf("normalizeRetryAfter(%v) = %v, want unchanged", long, got)
	}
}
