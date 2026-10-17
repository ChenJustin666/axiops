package appsvc

import (
	"strings"
	"testing"
)

func TestValidServiceIdentity(t *testing.T) {
	tests := []struct {
		name string
		code string
		want bool
	}{
		{"risk_server", "risk-server", true},
		{"order.api_v2", "order-api", true},
		{"Risk_Server", "risk-server", false},
		{"risk--server", "risk-server", true},
		{"risk_server", "risk_server", false},
		{"order-api", "Order-API", false},
		{"", "order-api", false},
	}
	for _, tt := range tests {
		if got := validServiceIdentity(tt.name, tt.code); got != tt.want {
			t.Fatalf("validServiceIdentity(%q,%q)=%v, want %v", tt.name, tt.code, got, tt.want)
		}
	}
}

func TestValidServiceName(t *testing.T) {
	for _, name := range []string{"risk_server", "order-api", "api.v2", "api__v2", "api--v2", "1worker", strings.Repeat("a", 128)} {
		if !validServiceName(name) {
			t.Errorf("valid service name rejected: %q", name)
		}
	}
	for _, name := range []string{"", "Risk_Server", "-api", "api-", "api..v2", "api.-v2", "中文服务", "api/name", strings.Repeat("a", 129)} {
		if validServiceName(name) {
			t.Errorf("invalid service name accepted: %q", name)
		}
	}
}
