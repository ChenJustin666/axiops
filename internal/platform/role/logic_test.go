package role

import "testing"

func TestValidPermission(t *testing.T) {
	valid := []PermissionItem{
		{Resource: "build", Action: "view", Effect: "allow"},
		{Resource: "release", Action: "deploy", Effect: "deny"},
		{Resource: "cluster", Action: "*", Effect: "allow"},
		{Resource: "cluster", Action: "test", Effect: "allow"},
		{Resource: "environment", Action: "manage_production", Effect: "allow"},
		{Resource: "deployment", Action: "restart", Effect: "deny"},
		{Resource: "monitoring", Action: "manage", Effect: "allow"},
		{Resource: "logging", Action: "manage", Effect: "deny"},
	}
	for _, permission := range valid {
		if !validPermission(permission) {
			t.Fatalf("valid permission rejected: %#v", permission)
		}
	}
	invalid := []PermissionItem{
		{Resource: "", Action: "view", Effect: "allow"},
		{Resource: "build", Action: "execute", Effect: "allow"},
		{Resource: "build", Action: "view", Effect: "maybe"},
	}
	for _, permission := range invalid {
		if validPermission(permission) {
			t.Fatalf("invalid permission accepted: %#v", permission)
		}
	}
}
