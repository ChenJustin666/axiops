package build

import (
	"context"
	"testing"
)

type fakeECSDispatcher struct{}

func (fakeECSDispatcher) RecoverBuild(context.Context, int64, string) (ProviderExecution, bool, error) {
	return ProviderExecution{ExternalID: "ecs-1"}, true, nil
}
func (fakeECSDispatcher) FinalizeBuild(context.Context, ProviderExecution) error { return nil }

func (fakeECSDispatcher) SubmitBuild(context.Context, ProviderBuildRequest) (ProviderExecution, error) {
	return ProviderExecution{ExternalID: "ecs-1"}, nil
}
func (fakeECSDispatcher) CancelBuild(context.Context, ProviderExecution) error { return nil }
func (fakeECSDispatcher) StatusBuild(context.Context, ProviderExecution) (ProviderState, error) {
	return ProviderState{Status: StatusSuccess, ImageDigest: "sha256:" + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, nil
}

func TestECSAgentProviderDelegatesBuildLifecycle(t *testing.T) {
	provider := NewECSAgentProvider(fakeECSDispatcher{})
	execution, err := provider.Submit(context.Background(), ProviderBuildRequest{BuildRunID: 1, IdempotencyKey: "build:1:submit"})
	if err != nil || execution.ExternalID != "ecs-1" {
		t.Fatalf("submit = %#v, %v", execution, err)
	}
	if _, err := provider.Status(context.Background(), execution); err != nil {
		t.Fatal(err)
	}
	if recovered, found, err := provider.Recover(context.Background(), 1, "build:1:submit"); err != nil || !found || recovered.ExternalID != execution.ExternalID {
		t.Fatalf("host recovery=%+v found=%v err=%v", recovered, found, err)
	}
	if err := provider.Finalize(context.Background(), execution); err != nil {
		t.Fatal(err)
	}
}
