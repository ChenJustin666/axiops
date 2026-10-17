package jenkins

import (
	"github.com/ChenJustin666/axiops/common/devops/internal/httpcheck"
	devopstypes "github.com/ChenJustin666/axiops/common/devops/types"
)

func New() devopstypes.Provider {
	return httpcheck.ServiceProvider{
		ProductName: "Jenkins",
		Probes: []httpcheck.Probe{
			{Path: "/api/json", Capability: "remoteApi"},
			{Path: "/", Capability: "web"},
		},
	}
}
