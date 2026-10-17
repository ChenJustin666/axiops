package build

import "testing"

func TestMetricRate(t *testing.T) {
	if got := metricRate(8, 10); got != 0.8 {
		t.Fatalf("rate=%v", got)
	}
	if got := metricRate(0, 0); got != 0 {
		t.Fatalf("empty rate=%v", got)
	}
}

func TestValidateConcurrencyLimit(t *testing.T) {
	if err := validateConcurrencyLimit(4, 5); err != nil {
		t.Fatalf("capacity was rejected: %v", err)
	}
	if err := validateConcurrencyLimit(5, 5); err == nil {
		t.Fatal("full concurrency policy was accepted")
	}
	if err := validateConcurrencyLimit(0, 0); err == nil {
		t.Fatal("invalid concurrency policy was accepted")
	}
}
