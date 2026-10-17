package build

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestBuildResourceInheritanceAndOmission(t *testing.T) {
	for _, tc := range []struct {
		name                                                   string
		platformCPU, platformMemory, serviceCPU, serviceMemory string
		wantCPU, wantMemory                                    string
	}{
		{name: "all empty"},
		{name: "only cpu", serviceCPU: "500m", wantCPU: "500m"},
		{name: "only memory", serviceMemory: "2Gi", wantMemory: "2Gi"},
		{name: "inherit platform", platformCPU: "1", platformMemory: "4Gi", wantCPU: "1", wantMemory: "4Gi"},
		{name: "override cpu", platformCPU: "1", platformMemory: "4Gi", serviceCPU: "250m", wantCPU: "250m", wantMemory: "4Gi"},
		{name: "override memory", platformCPU: "1", platformMemory: "4Gi", serviceMemory: "1Gi", wantCPU: "1", wantMemory: "1Gi"},
		{name: "whitespace is empty", platformCPU: " ", platformMemory: " ", serviceCPU: " ", serviceMemory: " "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := &fakeJobClient{}
			provider := NewKubeJobProvider("ci", client, client).WithDefaults(KubeJobDefaults{
				BuilderImage: "registry/ci:v1", BuildCommand: "build-and-report",
				ResourceCPU: tc.platformCPU, ResourceMemory: tc.platformMemory,
			})
			_, err := provider.Submit(context.Background(), ProviderBuildRequest{
				BuildRunID: 99, RepositoryURL: "https://git.example/app.git", IdempotencyKey: "build:99",
				ResourceCPU: tc.serviceCPU, ResourceMemory: tc.serviceMemory,
			})
			if err != nil {
				t.Fatal(err)
			}
			resources := client.created.Spec.Template.Spec.Containers[0].Resources
			for _, values := range []corev1.ResourceList{resources.Requests, resources.Limits} {
				for name, want := range map[corev1.ResourceName]string{corev1.ResourceCPU: tc.wantCPU, corev1.ResourceMemory: tc.wantMemory} {
					got, exists := values[name]
					if want == "" {
						if exists {
							t.Fatalf("empty %s unexpectedly received %s", name, got.String())
						}
					} else if !exists || got.Cmp(resource.MustParse(want)) != 0 {
						t.Fatalf("%s=%s, want %s", name, got.String(), want)
					}
				}
			}
		})
	}
}

func TestInvalidBuildResourcesFailBeforeJobCreation(t *testing.T) {
	for _, values := range [][2]string{{"wrong", ""}, {"0", ""}, {"-1", ""}, {"", "wrong"}, {"", "2"}, {"", "0"}} {
		client := &fakeJobClient{}
		provider := NewKubeJobProvider("ci", client, client).WithDefaults(KubeJobDefaults{BuilderImage: "registry/ci:v1", BuildCommand: "build-and-report"})
		_, err := provider.Submit(context.Background(), ProviderBuildRequest{
			BuildRunID: 99, RepositoryURL: "https://git.example/app.git", IdempotencyKey: "build:99",
			ResourceCPU: values[0], ResourceMemory: values[1],
		})
		if err == nil || client.created != nil {
			t.Fatalf("invalid resources %v were silently accepted: %v", values, err)
		}
	}
}
