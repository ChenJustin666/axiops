package project

import (
	"testing"
	"time"
)

func TestInitializationOnlyCompletesOnExplicitFinalConfirmation(t *testing.T) {
	now := time.Now()
	p := &Project{InitializationStatus: InitializationStatusPending}
	status, step := InitializationStatusInProgress, 4
	if !applyInitializationUpdate(p, &UpdateProjectRequest{InitializationStatus: &status, InitializationStep: &step}, now) {
		t.Fatal("progress update rejected")
	}
	if p.InitializationStatus != InitializationStatusInProgress || p.InitializedAt != nil {
		t.Fatal("visiting the last step must not complete the project")
	}
	status = InitializationStatusCompleted
	if !applyInitializationUpdate(p, &UpdateProjectRequest{InitializationStatus: &status, InitializationStep: &step}, now) || p.InitializedAt == nil || !p.InitializedAt.Equal(now) {
		t.Fatal("explicit completion did not record its timestamp")
	}
	later := now.Add(time.Hour)
	if !applyInitializationUpdate(p, &UpdateProjectRequest{Name: "renamed"}, later) || !p.InitializedAt.Equal(now) {
		t.Fatal("metadata update must preserve completion")
	}
	if !applyInitializationUpdate(p, &UpdateProjectRequest{InitializationStatus: &status, InitializationStep: &step}, later) || !p.InitializedAt.Equal(now) {
		t.Fatal("repeated completion must be idempotent")
	}
	status, step = InitializationStatusInProgress, 2
	if applyInitializationUpdate(p, &UpdateProjectRequest{InitializationStatus: &status, InitializationStep: &step}, later) {
		t.Fatal("a stale tab reopened a completed project")
	}
	if p.InitializationStatus != InitializationStatusCompleted || p.InitializationStep != 4 {
		t.Fatal("rejected progress update changed completion")
	}
}

func TestInitializationRejectsInvalidStatusAndPrematureCompletion(t *testing.T) {
	for _, item := range []struct {
		status string
		step   int
	}{
		{"unknown", 2}, {InitializationStatusInProgress, -1}, {InitializationStatusInProgress, 5},
		{InitializationStatusCompleted, 0}, {InitializationStatusCompleted, 3},
	} {
		p := &Project{InitializationStatus: InitializationStatusPending}
		if applyInitializationUpdate(p, &UpdateProjectRequest{InitializationStatus: &item.status, InitializationStep: &item.step}, time.Now()) {
			t.Fatalf("accepted invalid initialization update: %+v", item)
		}
	}
}
