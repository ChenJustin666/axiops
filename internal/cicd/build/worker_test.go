package build

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeCommandStore struct {
	command       *BuildCommand
	completed     bool
	failed        bool
	failureReason string
}

func (s *fakeCommandStore) ClaimCommand(context.Context, string) (*BuildCommand, error) {
	command := s.command
	s.command = nil
	return command, nil
}
func (s *fakeCommandStore) CompleteCommand(context.Context, int64, string) error {
	s.completed = true
	return nil
}
func (s *fakeCommandStore) FailCommand(_ context.Context, _ int64, _ string, reason string, _ int, _ time.Duration) error {
	s.failed, s.failureReason = true, reason
	return nil
}

type fakeResolver struct {
	submitted, cancelled bool
	emptyExecution       bool
}

func (r *fakeResolver) BuildRequest(context.Context, int64) (ProviderBuildRequest, error) {
	return ProviderBuildRequest{BuildRunID: 1}, nil
}
func (r *fakeResolver) Execution(context.Context, int64) (ProviderExecution, error) {
	if r.emptyExecution {
		return ProviderExecution{}, nil
	}
	return ProviderExecution{JobName: "axiops-build-1"}, nil
}
func (r *fakeResolver) MarkSubmitted(context.Context, int64, ProviderExecution) error {
	r.submitted = true
	return nil
}
func (r *fakeResolver) MarkCancelled(context.Context, int64) error { r.cancelled = true; return nil }

type fakeWorkerProvider struct {
	name      string
	submitErr error
}

func (p fakeWorkerProvider) Name() string { return p.name }
func (p fakeWorkerProvider) Submit(context.Context, ProviderBuildRequest) (ProviderExecution, error) {
	if p.submitErr != nil {
		return ProviderExecution{}, p.submitErr
	}
	return ProviderExecution{JobName: "axiops-build-1"}, nil
}
func (p fakeWorkerProvider) Cancel(context.Context, ProviderExecution) error { return nil }

func TestWorkerProcessesSubmit(t *testing.T) {
	store := &fakeCommandStore{command: &BuildCommand{ID: 11, BuildRunID: 1, CommandType: CommandSubmit, Provider: "fake", Attempts: 1}}
	resolver := &fakeResolver{}
	providers, err := NewProviderRegistry(fakeWorkerProvider{name: "fake"})
	if err != nil {
		t.Fatal(err)
	}
	worker := NewWorker(store, providers, resolver, 3, time.Second)
	processed, err := worker.ProcessOne(context.Background(), "worker-1")
	if err != nil || !processed || !store.completed || !resolver.submitted {
		t.Fatalf("ProcessOne() processed=%v error=%v completed=%v submitted=%v", processed, err, store.completed, resolver.submitted)
	}
}

func TestWorkerDeadLettersAfterMaxAttempts(t *testing.T) {
	store := &fakeCommandStore{command: &BuildCommand{ID: 12, BuildRunID: 1, CommandType: CommandSubmit, Provider: "fake", Attempts: 3}}
	resolver := &fakeResolver{}
	providers, _ := NewProviderRegistry(fakeWorkerProvider{name: "fake", submitErr: errors.New("provider down")})
	worker := NewWorker(store, providers, resolver, 3, time.Second)
	if _, err := worker.ProcessOne(context.Background(), "worker-1"); err != nil || !store.failed {
		t.Fatalf("dead letter handling error=%v failed=%v", err, store.failed)
	}
}

func TestWorkerCancelsBeforeProviderSubmission(t *testing.T) {
	store := &fakeCommandStore{command: &BuildCommand{ID: 13, BuildRunID: 1, CommandType: CommandCancel, Provider: "fake", Attempts: 1}}
	resolver := &fakeResolver{emptyExecution: true}
	providers, _ := NewProviderRegistry(fakeWorkerProvider{name: "fake"})
	processed, err := NewWorker(store, providers, resolver, 3, time.Second).ProcessOne(context.Background(), "worker-1")
	if err != nil || !processed || !resolver.cancelled || !store.completed {
		t.Fatalf("processed=%v cancelled=%v completed=%v err=%v", processed, resolver.cancelled, store.completed, err)
	}
}
