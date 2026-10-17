package workflow

import (
	"context"
	"fmt"
	"strconv"

	"helm.sh/helm/v3/pkg/release"
)

// A same-name release created outside this service must never be silently
// upgraded. Identity comes from the persisted task, not Chart values or labels.
func (p *OfficialHelmProvider) releaseOwnershipLabels(ctx context.Context, runID int64, nodeKey string) (map[string]string, error) {
	var pid, eid, sid int64
	err := p.db.QueryRowContext(ctx, `SELECT r.project_id,r.environment_id,
		CASE WHEN t.node_type='helm_deploy' THEN CAST(JSON_UNQUOTE(JSON_EXTRACT(t.node_snapshot,'$.config.service_id')) AS UNSIGNED) ELSE d.service_id END
		FROM workflow_run r JOIN workflow_task_run t ON t.workflow_run_id=r.id
		LEFT JOIN deployment_snapshot d ON t.node_type='helm_rollback'
		 AND d.id=CAST(JSON_UNQUOTE(JSON_EXTRACT(t.node_snapshot,'$.config.snapshot_id')) AS UNSIGNED)
		 AND d.project_id=r.project_id AND d.environment_id=r.environment_id AND d.provider='helm_deploy' AND d.status='applied'
		WHERE r.id=? AND t.node_key=? AND t.node_type IN ('helm_deploy','helm_rollback')`, runID, nodeKey).Scan(&pid, &eid, &sid)
	if err != nil || pid <= 0 || eid <= 0 || sid <= 0 {
		return nil, fmt.Errorf("Helm 服务归属无法确认，未提交发布")
	}
	return map[string]string{"axiops-project-id": strconv.FormatInt(pid, 10), "axiops-environment-id": strconv.FormatInt(eid, 10), "axiops-service-id": strconv.FormatInt(sid, 10)}, nil
}

func helmOwnerLabelsMatch(actual, expected map[string]string) bool {
	for _, key := range []string{"axiops-project-id", "axiops-environment-id", "axiops-service-id"} {
		if expected[key] == "" || actual[key] != expected[key] {
			return false
		}
	}
	return true
}

func (p *OfficialHelmProvider) validateReleaseOwnership(ctx context.Context, current *release.Release, clusterID int64, namespace string, labels map[string]string) error {
	if helmOwnerLabelsMatch(current.Labels, labels) {
		return nil
	}
	for key := range labels {
		if current.Labels[key] != "" {
			return fmt.Errorf("同名 Helm Release 属于其他项目、环境或服务，已停止发布")
		}
	}
	// Older AxiOps releases have no owner labels. Accept only an exact applied
	// revision registered to this service and target; stamp labels on upgrade.
	var count int
	err := p.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM deployment_snapshot WHERE project_id=? AND environment_id=? AND service_id=? AND cluster_id=? AND namespace=? AND provider='helm_deploy' AND helm_release_name=? AND helm_revision=? AND status='applied'`, labels["axiops-project-id"], labels["axiops-environment-id"], labels["axiops-service-id"], clusterID, namespace, current.Name, current.Version).Scan(&count)
	if err != nil {
		return err
	}
	if count != 1 {
		return fmt.Errorf("同名 Helm Release 未登记为当前服务的成功版本，已停止发布；请核对 Release 名称和资源归属")
	}
	return nil
}
