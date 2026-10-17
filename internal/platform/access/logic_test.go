package access

import "testing"

func TestValidSubject(t *testing.T) {
	if !validSubject("user") || !validSubject("department") {
		t.Fatal("supported subjects were rejected")
	}
	if validSubject("role") {
		t.Fatal("unsupported subject was accepted")
	}
}

func TestValidScope(t *testing.T) {
	tests := []struct {
		name      string
		scopeType string
		scopeID   int64
		want      bool
	}{
		{name: "platform", scopeType: "platform", scopeID: 0, want: true},
		{name: "platform with id", scopeType: "platform", scopeID: 1, want: false},
		{name: "project", scopeType: "project", scopeID: 10, want: true},
		{name: "workflow", scopeType: "workflow", scopeID: 42, want: true},
		{name: "project missing id", scopeType: "project", scopeID: 0, want: false},
		{name: "unknown", scopeType: "namespace", scopeID: 10, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validScope(tt.scopeType, tt.scopeID); got != tt.want {
				t.Fatalf("validScope(%q, %d)=%v want=%v", tt.scopeType, tt.scopeID, got, tt.want)
			}
		})
	}
}

func TestValidRoleBinding(t *testing.T) {
	if !validRoleBinding("scoped", "project", "developer") {
		t.Fatal("scoped role should be accepted for project binding")
	}
	if validRoleBinding("platform", "project", "platform_admin") {
		t.Fatal("platform role must not be accepted by role_binding")
	}
	if validRoleBinding("scoped", "platform", "developer") {
		t.Fatal("scoped role must not be promoted to platform scope")
	}
	if validRoleBinding("scoped", "project", "super_admin") {
		t.Fatal("super admin must not be assigned through role_binding")
	}
}
