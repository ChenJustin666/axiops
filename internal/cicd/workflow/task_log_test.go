package workflow

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestTaskLogDeduplicatesAndBoundsHistory(t *testing.T) {
	now := time.Now()
	log := appendTaskLog("", "[apply] api submitted", now)
	if next := appendTaskLog(log, "[apply] api submitted", now.Add(time.Second)); next != log {
		t.Fatal("unchanged snapshot duplicated")
	}
	for i := 0; i < 200; i++ {
		log = appendTaskLog(log, strings.Repeat("检查", 600)+time.Unix(int64(i), 0).String(), now)
	}
	log = appendTaskLog(log, "[failed] ImagePullBackOff", now)
	if len(log) > 64*1024 || !utf8.ValidString(log) || !strings.Contains(log, "[apply] api submitted") || !strings.Contains(log, "earlier checks truncated") || !strings.Contains(log, "[failed] ImagePullBackOff") {
		t.Fatalf("invalid retained log len=%d", len(log))
	}
}

func TestSafeDiagnosticMasksCommonCredentials(t *testing.T) {
	result := safeDiagnostic(`password=abc token: xyz Authorization: Bearer private https://user:pass@registry/v2/`)
	for _, secret := range []string{"abc", "xyz", "private", "user:pass"} {
		if strings.Contains(result, secret) {
			t.Fatalf("leaked credential: %s", result)
		}
	}
}

func TestTaskErrorSummaryFitsLegacyColumn(t *testing.T) {
	message := truncateDiagnostic(strings.Repeat("镜像拉取失败", 1000), 1000)
	if len(message) > 1024 || !utf8.ValidString(message) || !strings.HasSuffix(message, "[truncated]") {
		t.Fatal("invalid error summary")
	}
}

func TestDeploymentObservationsPersistOnlyChangedFacts(t *testing.T) {
	now := time.Now()
	observation := "[check] cluster=2 namespace=qa\n[rollout] Deployment/api ready=false available=0\n[pod] api-1 phase=Pending container=api ready=false restarts=0\n[event] Pod/api-1 2026-10-02T01:00:00Z Pulling: pulling image\n[waiting] waiting for rollout"
	log := appendTaskLog("", observation, now)
	if got := appendTaskLog(log, observation, now.Add(time.Second)); got != log {
		t.Fatal("identical observation added more output")
	}
	next := observation + "\n[event] Pod/api-1 2026-10-02T01:00:01Z Failed: denied"
	log = appendTaskLog(log, next, now.Add(2*time.Second))
	if strings.Count(log, "[rollout]") != 1 || strings.Count(log, "Pulling:") != 1 || strings.Count(log, "Failed: denied") != 1 {
		t.Fatal("one new event replayed unchanged snapshot")
	}
	// Simulate worker restart: all deduplication state comes from the saved text.
	if got := appendTaskLog(log, next, now.Add(3*time.Second)); got != log {
		t.Fatal("restart replayed saved observations")
	}
	recovered := strings.ReplaceAll(observation, "ready=false available=0", "ready=true available=1")
	log = appendTaskLog(log, recovered, now.Add(4*time.Second))
	log = appendTaskLog(log, observation, now.Add(5*time.Second))
	if strings.Count(log, "[rollout]") != 3 {
		t.Fatal("recovery or recurring unhealthy state was lost")
	}
	failed := observation + "\n[failed] timeout\n[pod] api-1 phase=Running container=api waiting=CrashLoopBackOff previous=Error exitCode=1"
	log = appendTaskLog(log, failed, now.Add(6*time.Second))
	if !strings.Contains(log, "[failed] timeout") || !strings.Contains(log, "CrashLoopBackOff") || !strings.Contains(log, "exitCode=1") {
		t.Fatal("failure evidence lost")
	}
	// Arbitrary process output must not be interpreted as Kubernetes observations.
	if got := appendTaskLog(log, "application output\napplication output", now); strings.Count(got, "application output") != 2 {
		t.Fatal("process output was collapsed")
	}
}
