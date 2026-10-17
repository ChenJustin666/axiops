package authz

import (
	"context"
	"testing"

	"github.com/casbin/casbin/v2"
)

func TestCheckPermissionDefaultDenyAndReadOnly(t *testing.T) {
	enforcer, err := casbin.NewSyncedEnforcer(newRBACModel())
	if err != nil {
		t.Fatalf("create enforcer: %v", err)
	}
	if _, err := enforcer.AddPolicy("viewer", "/manager/v1/cluster/*", "GET"); err != nil {
		t.Fatalf("add policy: %v", err)
	}

	manager := &CasbinRBACManager{enforcer: enforcer}
	tests := []struct {
		name   string
		roles  []string
		path   string
		method string
		want   bool
	}{
		{
			name:   "no role is denied",
			path:   "/manager/v1/cluster/1",
			method: "GET",
			want:   false,
		},
		{
			name:   "viewer can read",
			roles:  []string{"viewer"},
			path:   "/manager/v1/cluster/1",
			method: "GET",
			want:   true,
		},
		{
			name:   "viewer cannot write",
			roles:  []string{"viewer"},
			path:   "/manager/v1/cluster/1",
			method: "POST",
			want:   false,
		},
		{
			name:   "super admin can write",
			roles:  []string{"super_admin"},
			path:   "/manager/v1/cluster/1",
			method: "DELETE",
			want:   true,
		},
		{
			name:   "whitelist allows self service",
			path:   "/portal/v1/user/info",
			method: "GET",
			want:   true,
		},
		{
			name:   "whitelist does not allow self service update",
			path:   "/portal/v1/user/info",
			method: "PUT",
			want:   false,
		},
		{
			name:   "whitelist allows logout",
			path:   "/portal/v1/auth/logout",
			method: "POST",
			want:   true,
		},
		{
			name:   "whitelist does not allow logout with another method",
			path:   "/portal/v1/auth/logout",
			method: "GET",
			want:   false,
		},
		{
			name:   "whitelist allows websocket connection",
			path:   "/ws/v1/site-messages/connect",
			method: "GET",
			want:   true,
		},
		{
			name:   "whitelist does not allow lookalike path",
			path:   "/portal/v1/user/information",
			method: "GET",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := manager.CheckPermission(context.Background(), tt.roles, tt.path, tt.method)
			if err != nil {
				t.Fatalf("check permission: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}
