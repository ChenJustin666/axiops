package argocd

import (
	"github.com/ChenJustin666/axiops/common/devops/internal/httpcheck"
	devopstypes "github.com/ChenJustin666/axiops/common/devops/types"
)

func New() devopstypes.Provider {
	return httpcheck.ServiceProvider{
		ProductName: "Argo CD",
		Probes: []httpcheck.Probe{
			{Path: "/api/version", Capability: "versionApi"},
			{Path: "/api/v1/session/userinfo", Capability: "sessionApi"},
		},
	}
}
