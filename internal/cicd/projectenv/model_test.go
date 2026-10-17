package projectenv

import "testing"

func TestResolveEnvironmentTarget(t *testing.T) {
	k8sReq := CreateEnvironmentRequest{ClusterID: 2, Namespace: "payment-test"}
	for _, deliveryType := range []string{DeliveryTypeK8sYAML, DeliveryTypeHelm} {
		target, status, valid := ResolveEnvironmentTarget(deliveryType, k8sReq)
		if !valid || target != EnvironmentTypeK8sNamespace || status != BindingStatusValidated {
			t.Fatalf("unexpected K8s target for %s: %s %s %v", deliveryType, target, status, valid)
		}
	}
	target, status, valid := ResolveEnvironmentTarget(DeliveryTypeHost, CreateEnvironmentRequest{HostGroupRef: "binding-1"})
	if !valid || target != EnvironmentTypeHostGroup || status != BindingStatusPendingValidation {
		t.Fatalf("unexpected host target: %s %s %v", target, status, valid)
	}
}

func TestResolveEnvironmentTargetRejectsMixedResources(t *testing.T) {
	cases := []struct {
		deliveryType string
		req          CreateEnvironmentRequest
	}{
		{DeliveryTypeK8sYAML, CreateEnvironmentRequest{ClusterID: 1, Namespace: "Bad_Name"}},
		{DeliveryTypeHelm, CreateEnvironmentRequest{ClusterID: 1, Namespace: "test", HostGroupRef: "group"}},
		{DeliveryTypeHost, CreateEnvironmentRequest{ClusterID: 1, HostGroupRef: "group"}},
		{"unclassified", CreateEnvironmentRequest{ClusterID: 1, Namespace: "test"}},
	}
	for _, tc := range cases {
		if _, _, valid := ResolveEnvironmentTarget(tc.deliveryType, tc.req); valid {
			t.Fatalf("unexpected valid target: %+v", tc)
		}
	}
}

func TestValidateVariableItems(t *testing.T) {
	valid := []VariableItemInput{
		{VariableKey: "IMAGE_TAG", Value: "latest"},
		{VariableKey: "db.password", IsSecret: true, CredentialID: 7},
	}
	if !ValidateVariableItems(valid) {
		t.Fatal("expected valid variables")
	}
	invalid := [][]VariableItemInput{
		{{VariableKey: "bad key", Value: "x"}},
		{{VariableKey: "TOKEN", IsSecret: true, CredentialID: 7, Value: "plaintext"}},
		{{VariableKey: "TOKEN", IsSecret: true}},
		{{VariableKey: "A"}, {VariableKey: "A"}},
		{{VariableKey: "A", CredentialID: 1}},
	}
	for _, items := range invalid {
		if ValidateVariableItems(items) {
			t.Fatalf("unexpected valid variables: %+v", items)
		}
	}
}

func TestValidateEnvironmentIdentity(t *testing.T) {
	for _, code := range []string{"dev", "test-1", "prod"} {
		if !ValidateEnvironmentIdentity("Payment "+code, code) {
			t.Fatalf("expected valid environment code %q", code)
		}
	}
	for _, code := range []string{"", "Prod", "1prod", "bad_name", "a23456789012345678901234567890123"} {
		if ValidateEnvironmentIdentity("Payment", code) {
			t.Fatalf("expected invalid environment code %q", code)
		}
	}
}

func TestEnvironmentServiceCompatibility(t *testing.T) {
	if !IsEnvironmentCompatible(EnvironmentTypeK8sNamespace, BindingStatusValidated, DeliveryTypeK8sYAML) {
		t.Fatal("K8s YAML service should be compatible with a validated K8s environment")
	}
	if IsEnvironmentCompatible(EnvironmentTypeK8sNamespace, BindingStatusPendingValidation, DeliveryTypeK8sYAML) {
		t.Fatal("pending environment must not be deployable")
	}
	if IsEnvironmentCompatible(EnvironmentTypeK8sNamespace, BindingStatusValidated, DeliveryTypeHost) {
		t.Fatal("host service must not target a K8s environment")
	}
}

func TestValidateServiceOverrideRequest(t *testing.T) {
	replicas := 3
	if !ValidateServiceOverrideRequest(SaveServiceOverrideRequest{ServiceRevisionID: 2, Replicas: &replicas}) {
		t.Fatal("expected valid override")
	}
	negative := -1
	tooMany := 10001
	for _, req := range []SaveServiceOverrideRequest{
		{ExpectedRevision: -1, ServiceRevisionID: 2},
		{ExpectedRevision: 0, ServiceRevisionID: 0},
		{ExpectedRevision: 0, ServiceRevisionID: 2, Replicas: &negative},
		{ExpectedRevision: 0, ServiceRevisionID: 2, Replicas: &tooMany},
	} {
		if ValidateServiceOverrideRequest(req) {
			t.Fatalf("unexpected valid override: %+v", req)
		}
	}
}
