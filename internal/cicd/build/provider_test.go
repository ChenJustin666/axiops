package build

import (
	"context"
	"testing"
)

type fakeProvider struct{ name string }

func (p fakeProvider) Name() string { return p.name }
func (p fakeProvider) Submit(context.Context, ProviderBuildRequest) (ProviderExecution, error) {
	return ProviderExecution{}, nil
}
func (p fakeProvider) Cancel(context.Context, ProviderExecution) error { return nil }

func TestProviderRegistry(t *testing.T) {
	registry, err := NewProviderRegistry(fakeProvider{name: "kube_job"})
	if err != nil {
		t.Fatalf("NewProviderRegistry() error = %v", err)
	}
	if _, err := registry.Get("kube_job"); err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if err := registry.Register(fakeProvider{name: "kube_job"}); err == nil {
		t.Fatal("duplicate provider registration should fail")
	}
	if _, err := registry.Get("jenkins"); err == nil {
		t.Fatal("unknown provider lookup should fail")
	}
}
