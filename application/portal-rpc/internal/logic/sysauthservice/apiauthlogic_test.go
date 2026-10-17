package sysauthservicelogic

import "testing"

func TestAllowDemoModeRequest(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		method string
		want   bool
	}{
		{name: "allows get", path: "/manager/v1/cluster/1", method: "GET", want: true},
		{name: "normalizes method", path: "/manager/v1/cluster/1", method: " get ", want: true},
		{name: "allows logout post", path: "/portal/v1/auth/logout", method: "POST", want: true},
		{name: "denies logout delete", path: "/portal/v1/auth/logout", method: "DELETE", want: false},
		{name: "denies write", path: "/manager/v1/cluster/1", method: "POST", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := allowDemoModeRequest(tt.path, tt.method)
			if got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}
