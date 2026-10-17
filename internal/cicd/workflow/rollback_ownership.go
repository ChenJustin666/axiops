package workflow

import (
	"context"
	"fmt"
)

// A hand-written service YAML must not replay another service's controller or
// shared resource. No resource is deleted as an implicit rollback side effect.
func checkRollbackResourceOwnership(ctx context.Context, q rollbackQuery, target *DeploymentSnapshot) error {
	if target.Provider == NodeHelmDeploy {
		var conflicts int
		if err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM deployment_snapshot ds WHERE ds.project_id=? AND ds.environment_id=? AND ds.service_id<>? AND ds.provider=? AND ds.helm_release_name=? AND ds.id=(SELECT MAX(other.id) FROM deployment_snapshot other WHERE other.project_id=ds.project_id AND other.environment_id=ds.environment_id AND other.service_id=ds.service_id)`, target.ProjectID, target.EnvironmentID, target.ServiceID, NodeHelmDeploy, target.HelmReleaseName).Scan(&conflicts); err != nil {
			return err
		}
		if conflicts > 0 {
			return fmt.Errorf("此 Helm Release 还被其他服务管理，已阻止回滚；请先明确 Release 归属")
		}
		return nil
	}
	objects, err := ValidateYAMLManifest(target.RenderedManifest, target.Namespace)
	if err != nil {
		return err
	}
	identities := map[string]bool{}
	for _, o := range objects {
		identities[o.GetAPIVersion()+"/"+o.GetKind()+"/"+o.GetName()] = true
	}
	rows, err := q.QueryContext(ctx, `SELECT ds.rendered_manifest FROM deployment_snapshot ds WHERE ds.project_id=? AND ds.environment_id=? AND ds.service_id<>? AND ds.provider=? AND ds.id=(SELECT MAX(other.id) FROM deployment_snapshot other WHERE other.project_id=ds.project_id AND other.environment_id=ds.environment_id AND other.service_id=ds.service_id)`, target.ProjectID, target.EnvironmentID, target.ServiceID, NodeYAMLDeploy)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return err
		}
		others, err := ValidateYAMLManifest(raw, target.Namespace)
		if err != nil {
			return fmt.Errorf("其他服务的资源归属无法确认，已阻止回滚")
		}
		for _, o := range others {
			if identities[o.GetAPIVersion()+"/"+o.GetKind()+"/"+o.GetName()] {
				return fmt.Errorf("目标版本包含其他服务也管理的 %s/%s；请先明确资源归属，避免影响其他服务", o.GetKind(), o.GetName())
			}
		}
	}
	return rows.Err()
}

func checkRollbackEnvironmentIsolation(ctx context.Context, q sourceQuery, env *EnvironmentTarget) error {
	var duplicates int
	if err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM project_environment e JOIN project p ON p.id=e.project_id AND p.status<>'deleted' WHERE e.cluster_id=? AND e.namespace=? AND e.status='active' AND e.id<>?`, env.ClusterID, env.Namespace, env.ID).Scan(&duplicates); err != nil {
		return err
	}
	if duplicates > 0 {
		return fmt.Errorf("此命名空间还被其他环境绑定，无法确保回滚环境隔离；请先核对环境绑定")
	}
	return nil
}
