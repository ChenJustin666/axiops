package workflow

import (
	"context"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
)

// Reads, edits and execution must resolve the same environment context. A bound
// ID is server-owned; the caller cannot substitute another project's target.
func (l *Logic) authorizeTarget(ctx context.Context, userID, projectID, workflowID, environmentID int64, action string) int {
	request := authz.Request{Resource: "workflow", Action: action, ProjectID: projectID, WorkflowID: workflowID, EnvironmentID: environmentID}
	if environmentID > 0 {
		resolver, ok := l.repo.(interface {
			FindEnvironmentTarget(context.Context, int64) (*EnvironmentTarget, error)
		})
		if !ok {
			return errcode.CodeDBError
		}
		environment, err := resolver.FindEnvironmentTarget(ctx, environmentID)
		if err != nil {
			return errcode.CodeDBError
		}
		if environment != nil {
			if environment.ID != environmentID || environment.ProjectID != projectID {
				return errcode.CodeNoPermission
			}
			request.Environment = environment.Code
			request.ClusterID = environment.ClusterID
		}
	}
	allowed, err := l.authorizer.Authorize(ctx, userID, request)
	if err != nil {
		return errcode.CodeDBError
	}
	if !allowed {
		return errcode.CodeNoPermission
	}
	return errcode.CodeSuccess
}
