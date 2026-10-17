package observabilityaccess

import (
	"context"
	"github.com/ChenJustin666/axiops/application/manager-rpc/client/managerservice"
)

func Workspaces(manager managerservice.ManagerService) WorkspaceResolver {
	return func(ctx context.Context, id uint64) (Target, error) {
		info, err := manager.ProjectWorkspaceGetById(ctx, &managerservice.GetOnecProjectWorkspaceByIdReq{Id: id})
		if err != nil || info == nil || info.Data == nil {
			return Target{}, ErrDenied
		}
		return Target{ClusterUUID: info.Data.ClusterUuid, Namespace: info.Data.Namespace}, nil
	}
}
