package workflow

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
)

// Successful build artifacts are selectable without claiming functional acceptance.
// Selection always uses the immutable digest and the target's own configuration.
func readProductionBuild(ctx context.Context, q sourceQuery, id int64) (*PromotionBuild, error) {
	var b PromotionBuild
	err := q.QueryRowContext(ctx, `SELECT id,project_id,service_id,build_number,status,COALESCE(image_url,''),COALESCE(image_digest,''),COALESCE(git_commit,''),COALESCE(git_branch,'') FROM build_run WHERE id=?`, id).Scan(&b.ID, &b.ProjectID, &b.ServiceID, &b.BuildNumber, &b.Status, &b.ImageURL, &b.ImageDigest, &b.Commit, &b.Branch)
	if err != nil || b.Status != "success" || !validPromotionImage(&b) {
		return nil, fmt.Errorf("所选构建未成功生成可追溯镜像，或记录已不可用")
	}
	return &b, nil
}

func productionBuildMetadata(ctx context.Context, q sourceQuery, b *PromotionBuild) (ProductionCandidate, error) {
	c := ProductionCandidate{Verification: Verification{BuildID: b.ID, ServiceID: b.ServiceID, ImageRef: b.ImageURL + "@" + b.ImageDigest, Method: "build"}, BuildNumber: b.BuildNumber, Commit: b.Commit, Branch: b.Branch}
	err := q.QueryRowContext(ctx, `SELECT COALESCE(DATE_FORMAT(COALESCE(finished_at,created_at),'%Y-%m-%dT%H:%i:%sZ'),''),COALESCE(git_commit_url,''),COALESCE(git_compare_url,'') FROM build_run WHERE id=?`, b.ID).Scan(&c.BuiltAt, &c.CommitURL, &c.CompareURL)
	if err != nil {
		return c, err
	}
	_ = q.QueryRowContext(ctx, "SELECT name FROM app_service WHERE id=? AND project_id=? AND status='active'", b.ServiceID, b.ProjectID).Scan(&c.Name)
	err = q.QueryRowContext(ctx, `SELECT wr.id,wr.workflow_id,COALESCE(wr.run_number,0),COALESCE(wr.environment_id,0),COALESCE(e.name,'') FROM workflow_task_run task JOIN workflow_run wr ON wr.id=task.workflow_run_id LEFT JOIN project_environment e ON e.id=wr.environment_id AND e.project_id=wr.project_id WHERE task.provider_run_id=CONCAT('build:',?) AND task.node_type='build' AND wr.project_id=? ORDER BY task.id DESC LIMIT 1`, b.ID, b.ProjectID).Scan(&c.RunID, &c.WorkflowID, &c.RunNumber, &c.EnvironmentID, &c.SourceName)
	if err != nil && err != sql.ErrNoRows {
		return c, err
	}
	return c, nil
}

func listProductionImages(ctx context.Context, q sourceQuery, pid, sid int64) ([]ProductionCandidate, error) {
	rows, err := q.QueryContext(ctx, `SELECT b.id,b.project_id,b.service_id,b.build_number,b.status,COALESCE(b.image_url,''),COALESCE(b.image_digest,''),COALESCE(b.git_commit,''),COALESCE(b.git_branch,''),
		COALESCE(DATE_FORMAT(COALESCE(b.finished_at,b.created_at),'%Y-%m-%dT%H:%i:%sZ'),''),COALESCE(b.git_commit_url,''),COALESCE(b.git_compare_url,''),svc.name,
		COALESCE(wr.id,0),COALESCE(wr.workflow_id,0),COALESCE(wr.run_number,0),COALESCE(wr.environment_id,0),COALESCE(e.name,'')
		FROM build_run b JOIN app_service svc ON svc.id=b.service_id AND svc.project_id=b.project_id AND svc.status='active'
		LEFT JOIN workflow_task_run task ON task.id=(SELECT MAX(t.id) FROM workflow_task_run t WHERE t.node_type='build' AND t.provider_run_id=CONCAT('build:',b.id))
		LEFT JOIN workflow_run wr ON wr.id=task.workflow_run_id AND wr.project_id=b.project_id
		LEFT JOIN project_environment e ON e.id=wr.environment_id AND e.project_id=wr.project_id
		WHERE b.project_id=? AND b.service_id=? AND b.status='success' ORDER BY b.id DESC LIMIT 100`, pid, sid)
	if err != nil {
		return nil, err
	}
	images := []ProductionCandidate{}
	for rows.Next() {
		var b PromotionBuild
		var c ProductionCandidate
		if err = rows.Scan(&b.ID, &b.ProjectID, &b.ServiceID, &b.BuildNumber, &b.Status, &b.ImageURL, &b.ImageDigest, &b.Commit, &b.Branch,
			&c.BuiltAt, &c.CommitURL, &c.CompareURL, &c.Name, &c.RunID, &c.WorkflowID, &c.RunNumber, &c.EnvironmentID, &c.SourceName); err != nil {
			rows.Close()
			return nil, err
		}
		if !validPromotionImage(&b) {
			continue
		}
		c.BuildID = b.ID
		c.ServiceID = b.ServiceID
		c.BuildNumber = b.BuildNumber
		c.ImageRef = b.ImageURL + "@" + b.ImageDigest
		c.Commit = b.Commit
		c.Branch = b.Branch
		c.Method = "build"
		images = append(images, c)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	// Inspect actual successful deployment manifests, not workflow status alone.
	rows, err = q.QueryContext(ctx, `SELECT ds.environment_id,ds.rendered_manifest,e.name,ds.namespace,ds.provider,ds.values_snapshot FROM deployment_snapshot ds JOIN project_environment e ON e.id=ds.environment_id AND e.project_id=ds.project_id AND e.status='active' AND e.cluster_id=ds.cluster_id AND e.namespace=ds.namespace WHERE ds.project_id=? AND ds.service_id=? AND ds.status='applied' AND ds.provider IN (?,?) ORDER BY ds.id DESC LIMIT 100`, pid, sid, NodeYAMLDeploy, NodeHelmDeploy)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var eid int64
		var manifest []byte
		var name string
		var namespace string
		var provider string
		var values []byte
		if err = rows.Scan(&eid, &manifest, &name, &namespace, &provider, &values); err != nil {
			return nil, err
		}
		refs := describeDeployment(&DeploymentSnapshot{Provider: provider, Namespace: namespace, RenderedManifest: manifest, ValuesSnapshot: values}).Images
		original := len(images)
		for i := 0; i < original; i++ {
			for _, ref := range refs {
				if images[i].ImageRef == ref {
					if images[i].EnvironmentID == eid {
						images[i].DeploymentStatus = "succeeded"
						continue
					}
					exists := false
					for _, other := range images {
						if other.BuildID == images[i].BuildID && other.EnvironmentID == eid {
							exists = true
							break
						}
					}
					if !exists {
						copy := images[i]
						copy.EnvironmentID = eid
						copy.SourceName = name
						copy.DeploymentStatus = "succeeded"
						images = append(images, copy)
					}
				}
			}
		}
	}
	return images, rows.Err()
}

func loadProductionPickPreview(ctx context.Context, q sourceQuery, wid int64, pick ProductionPick) (*ProductionPreview, error) {
	if (pick.BuildID > 0) == (pick.VerificationID > 0) || pick.BuildID < 0 || pick.VerificationID < 0 {
		return nil, fmt.Errorf("请选择一个有效镜像版本")
	}
	if pick.BuildID == 0 {
		return loadProductionPreview(ctx, q, wid, pick.VerificationID)
	}
	w, err := loadProductionWorkflow(ctx, q, wid)
	if err != nil {
		return nil, err
	}
	e, err := productionEnv(ctx, q, w.TargetEnvironmentID)
	if err != nil {
		return nil, err
	}
	if !e.IsProduction || e.ProjectID != w.ProjectID {
		return nil, fmt.Errorf("工作流未绑定此项目的生产环境")
	}
	b, err := readProductionBuild(ctx, q, pick.BuildID)
	if err != nil {
		return nil, err
	}
	if b.ProjectID != w.ProjectID {
		return nil, fmt.Errorf("镜像不属于当前项目")
	}
	c, err := productionBuildMetadata(ctx, q, b)
	if err != nil {
		return nil, err
	}
	return renderProductionPreview(ctx, q, w, e, c, b, nil, ProductionPolicy{})
}

func (l *Logic) ProductionImagePreview(ctx context.Context, uid, wid, bid int64) (*ProductionPreview, error) {
	return l.productionPickPreview(ctx, uid, wid, ProductionPick{BuildID: bid})
}

// Explicit build-and-deploy is supported even for a project with one environment.
// The transactional check requires frozen target configuration and a build for
// every deployed service; it cannot be used as an arbitrary-image shortcut.
func validateProductionBuildTasks(ctx context.Context, q sourceQuery, run *WorkflowRun, tasks []TaskRun) error {
	builds := map[int64]bool{}
	for _, task := range tasks {
		if task.NodeType == NodeBuild {
			var n Node
			var c struct {
				ServiceID int64 `json:"service_id"`
			}
			if json.Unmarshal(task.NodeSnapshot, &n) != nil || json.Unmarshal(n.Config, &c) != nil || c.ServiceID <= 0 {
				return fmt.Errorf("生产构建服务配置无效")
			}
			builds[c.ServiceID] = true
		}
	}
	count := 0
	for _, task := range tasks {
		if task.NodeType == NodeHelmDeploy {
			cfg, err := helmDeployConfig(task.NodeSnapshot)
			if err != nil || !builds[cfg.ServiceID] || !hasServiceBuildDependency(tasks, task, cfg.ServiceID) || cfg.FrozenHelm == nil || cfg.ImageRef != "" {
				return fmt.Errorf("生产 Helm 构建必须绑定服务构建与目标环境配置")
			}
			current, err := loadHelmEnvironmentConfig(ctx, q, run.EnvironmentID, cfg.ServiceID, cfg.ReleaseName)
			if err != nil || helmConfigurationDigest(current) != helmConfigurationDigest(cfg) {
				return fmt.Errorf("生产 Helm 环境配置已变化，请重新执行")
			}
			count++
			continue
		}
		if task.NodeType != NodeYAMLDeploy {
			continue
		}
		cfg, err := yamlDeployConfig(task.NodeSnapshot)
		if err != nil || !builds[cfg.ServiceID] || !hasServiceBuildDependency(tasks, task, cfg.ServiceID) || cfg.FrozenYAML == nil || cfg.FrozenYAML.ImageRef != "" {
			return fmt.Errorf("生产构建部署必须绑定服务构建与目标环境配置")
		}
		var revision int64
		if err = q.QueryRowContext(ctx, `SELECT COALESCE(ov.service_revision_id,svc.current_revision_id) FROM app_service svc JOIN environment_service_override ov ON ov.service_id=svc.id AND ov.project_environment_id=? WHERE svc.id=? AND svc.project_id=? AND svc.status='active'`, run.EnvironmentID, cfg.ServiceID, run.ProjectID).Scan(&revision); err != nil || revision != cfg.ServiceRevisionID {
			return fmt.Errorf("请先为服务 %d 保存目标生产环境配置", cfg.ServiceID)
		}
		current, err := (&SQLYAMLSourceStore{db: q}).LoadYAMLSource(ctx, run.EnvironmentID, cfg.ServiceID, cfg.ServiceRevisionID)
		if err != nil || current == nil || productionConfigDigest(current) != productionConfigDigest(cfg.FrozenYAML) {
			return fmt.Errorf("生产环境配置已变化，请重新执行")
		}
		count++
	}
	if count == 0 {
		return fmt.Errorf("生产构建部署缺少 YAML/Helm 部署节点")
	}
	return nil
}

func validateProductionBuildExecution(ctx context.Context, q sourceQuery, run WorkflowRun, task TaskRun, env *EnvironmentTarget) error {
	cfg, err := yamlDeployConfig(task.NodeSnapshot)
	if err != nil || cfg.FrozenYAML == nil || cfg.FrozenYAML.ImageRef != "" {
		return fmt.Errorf("缺少生产构建部署配置快照")
	}
	buildID, err := matchingBuildID(run.Tasks, cfg.ServiceID)
	if err != nil || buildID <= 0 {
		return fmt.Errorf("生产部署尚未获得当前运行的成功构建镜像")
	}
	b, err := readProductionBuild(ctx, q, buildID)
	if err != nil || b.ProjectID != run.ProjectID || b.ServiceID != cfg.ServiceID {
		return fmt.Errorf("当前构建镜像与生产服务不匹配")
	}
	var exists int
	if err = q.QueryRowContext(ctx, "SELECT COUNT(*) FROM app_service WHERE id=? AND project_id=? AND status='active'", cfg.ServiceID, run.ProjectID).Scan(&exists); err != nil || exists != 1 {
		return fmt.Errorf("生产服务不可用")
	}
	var latestRun int64
	err = q.QueryRowContext(ctx, "SELECT workflow_run_id FROM deployment_snapshot WHERE project_id=? AND environment_id=? AND service_id=? ORDER BY id DESC LIMIT 1", run.ProjectID, env.ID, cfg.ServiceID).Scan(&latestRun)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if latestRun > run.ID {
		return fmt.Errorf("已有新的生产部署，旧任务已停止")
	}
	if err = checkRollbackConflicts(ctx, q, run.ProjectID, env.ID, []int64{cfg.ServiceID}, true, run.ID); err != nil {
		return err
	}
	if err = checkRollbackEnvironmentIsolation(ctx, q, env); err != nil {
		return err
	}
	target, err := NewDefaultYAMLResolver(&SQLYAMLSourceStore{db: q}).RenderYAML(ctx, run, cfg)
	if err != nil {
		return err
	}
	return checkRollbackResourceOwnership(ctx, q, &DeploymentSnapshot{ProjectID: run.ProjectID, EnvironmentID: env.ID, ServiceID: cfg.ServiceID, Provider: NodeYAMLDeploy, Namespace: env.Namespace, RenderedManifest: target.Manifest})
}

func (l *Logic) authorizeProductionBuild(ctx context.Context, uid int64, env *EnvironmentTarget, wid int64) error {
	if err := l.productionWorkflowPermission(ctx, uid, env, wid); err != nil {
		return err
	}
	allowed, err := l.authorizer.Authorize(ctx, uid, authz.Request{Resource: "environment", Action: "view", ProjectID: env.ProjectID, EnvironmentID: env.ID, Environment: env.Code, ClusterID: env.ClusterID})
	if err != nil || !allowed {
		return fmt.Errorf("无权读取生产环境配置")
	}
	return nil
}
