package appsvc

import "testing"

func TestDeliveryMappings(t *testing.T) {
	tests := []struct {
		deliveryType string
		deployType   string
		format       string
	}{
		{DeliveryTypeK8sYAML, "k8s", "k8s_yaml"},
		{DeliveryTypeHelm, "helm", "helm_chart"},
		{DeliveryTypeHost, "host", "host_spec"},
	}
	for _, tc := range tests {
		if !IsActiveDeliveryType(tc.deliveryType) {
			t.Fatalf("expected active delivery type %q", tc.deliveryType)
		}
		if got := LegacyDeployType(tc.deliveryType); got != tc.deployType {
			t.Fatalf("LegacyDeployType(%q) = %q, want %q", tc.deliveryType, got, tc.deployType)
		}
		if got := DefinitionFormat(tc.deliveryType); got != tc.format {
			t.Fatalf("DefinitionFormat(%q) = %q, want %q", tc.deliveryType, got, tc.format)
		}
	}
}

func TestUnclassifiedDeliveryIsNotRunnable(t *testing.T) {
	if IsActiveDeliveryType(DeliveryTypeUnclassified) {
		t.Fatal("unclassified service must not be runnable")
	}
	if LegacyDeployType(DeliveryTypeUnclassified) != "" || DefinitionFormat(DeliveryTypeUnclassified) != "" {
		t.Fatal("unclassified service must not have an execution mapping")
	}
}

func TestSourceTypes(t *testing.T) {
	for _, sourceType := range []string{"git", "manual", "template"} {
		if !IsValidSourceType(sourceType) {
			t.Fatalf("expected source type %q to be valid", sourceType)
		}
	}
	for _, sourceType := range []string{"", "upload", "ssh"} {
		if IsValidSourceType(sourceType) {
			t.Fatalf("expected source type %q to be invalid", sourceType)
		}
	}
}

func TestHelmCreationSourcesPreserveLegacyEditing(t *testing.T) {
	for _, delivery := range []string{DeliveryTypeHelm, DeliveryTypeK8sYAML, DeliveryTypeHost} {
		for _, source := range []string{"template", "git", "manual", "upload", ""} {
			want := (source == "git" || source == "template") || (source == "manual" && delivery != DeliveryTypeHelm)
			if got := IsValidCreationSource(delivery, source); got != want {
				t.Errorf("creation %s/%s = %v, want %v", delivery, source, got, want)
			}
		}
	}
	if !IsValidSourceType("manual") {
		t.Fatal("legacy manual services must remain readable and editable")
	}
}

func TestServiceStatuses(t *testing.T) {
	for _, status := range []string{ServiceStatusActive, ServiceStatusInactive} {
		if !IsValidServiceStatus(status) {
			t.Fatalf("expected service status %q to be valid", status)
		}
	}
	for _, status := range []string{"", "disabled", "deleted", "pending"} {
		if IsValidServiceStatus(status) {
			t.Fatalf("expected service status %q to be invalid", status)
		}
	}
}
