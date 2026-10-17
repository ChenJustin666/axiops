package workflow

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type RollbackRequest struct {
	OriginRunID   int64  `json:"origin_run_id"`
	ServiceID     int64  `json:"service_id"`
	ExpectedToken string `json:"expected_token"`
}

type DeploymentVersion struct {
	ServiceName  string    `json:"service_name"`
	SnapshotID   int64     `json:"snapshot_id"`
	RunID        int64     `json:"run_id"`
	RunNumber    int64     `json:"run_number"`
	WorkflowID   int64     `json:"workflow_id"`
	RevisionID   int64     `json:"revision_id"`
	Digest       string    `json:"digest"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	Images       []string  `json:"images"`
	HelmRevision int       `json:"helm_revision,omitempty"`
}

type RollbackService struct {
	ServiceID int64               `json:"service_id"`
	Name      string              `json:"name"`
	Provider  string              `json:"provider"`
	Versions  []DeploymentVersion `json:"versions"`
}

type RollbackPreview struct {
	Token            string            `json:"token"`
	OriginRunID      int64             `json:"origin_run_id"`
	WorkflowID       int64             `json:"workflow_id"`
	EnvironmentID    int64             `json:"environment_id"`
	ClusterID        int64             `json:"cluster_id"`
	Namespace        string            `json:"namespace"`
	ServiceID        int64             `json:"service_id"`
	ServiceName      string            `json:"service_name"`
	RequiresApproval bool              `json:"requires_approval"`
	Current          DeploymentVersion `json:"current"`
	Target           DeploymentVersion `json:"target"`
	CurrentManifest  string            `json:"current_manifest"`
	TargetManifest   string            `json:"target_manifest"`
	BlockedReason    string            `json:"blocked_reason"`
	Warnings         []string          `json:"warnings"`
	// These fields never leave the API; submission and worker validation use them.
	environment     *EnvironmentTarget
	snapshot        *DeploymentSnapshot
	workflowVersion int64
}

type rollbackRepository interface {
	ListRollbackVersions(context.Context, int64, int64, int64, string) ([]DeploymentVersion, error)
	PreviewRollback(context.Context, int64, int64, int64) (*RollbackPreview, error)
	FindRollbackRequest(context.Context, string) (*WorkflowRun, error)
}

func rollbackServiceIDs(run *WorkflowRun) map[int64]string {
	result := map[int64]string{}
	for _, task := range run.Tasks {
		provider := task.NodeType
		if provider == NodeYAMLRollback {
			provider = NodeYAMLDeploy
		}
		if provider == NodeHelmRollback {
			provider = NodeHelmDeploy
		}
		if provider != NodeYAMLDeploy && provider != NodeHelmDeploy {
			continue
		}
		var node Node
		var c struct {
			ServiceID int64 `json:"service_id"`
		}
		if json.Unmarshal(task.NodeSnapshot, &node) == nil && json.Unmarshal(node.Config, &c) == nil && c.ServiceID > 0 {
			result[c.ServiceID] = provider
		}
	}
	return result
}

func (l *Logic) rollbackOrigin(ctx context.Context, userID, runID, serviceID int64) (*WorkflowRun, int) {
	run, code := l.GetRun(ctx, userID, runID)
	if code != 0 {
		return nil, code
	}
	if run.WorkflowID <= 0 || run.EnvironmentID <= 0 {
		return nil, errcode.CodeParamInvalid
	}
	if _, ok := rollbackServiceIDs(run)[serviceID]; !ok {
		return nil, errcode.CodeParamInvalid
	}
	runner, ok := l.repo.(runRepository)
	if !ok {
		return nil, errcode.CodeDBError
	}
	env, err := runner.FindEnvironmentTarget(ctx, run.EnvironmentID)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if env == nil || env.ID != run.EnvironmentID || env.ProjectID != run.ProjectID {
		return nil, errcode.CodeParamInvalid
	}
	allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "deployment", Action: "rollback", ProjectID: run.ProjectID, EnvironmentID: run.EnvironmentID, Environment: env.Code, ServiceID: serviceID, ClusterID: env.ClusterID})
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if !allowed {
		return nil, errcode.CodeNoPermission
	}
	return run, 0
}

func (l *Logic) RollbackOptions(ctx context.Context, userID, runID int64) ([]RollbackService, int) {
	run, code := l.GetRun(ctx, userID, runID)
	if code != 0 {
		return nil, code
	}
	repo, ok := l.repo.(rollbackRepository)
	if !ok {
		return nil, errcode.CodeDBError
	}
	result := []RollbackService{}
	ids := rollbackServiceIDs(run)
	ordered := make([]int64, 0, len(ids))
	for id := range ids {
		ordered = append(ordered, id)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	for _, sid := range ordered {
		if _, code = l.rollbackOrigin(ctx, userID, runID, sid); code == errcode.CodeNoPermission {
			continue
		} else if code != 0 {
			return nil, code
		}
		versions, err := repo.ListRollbackVersions(ctx, run.ProjectID, run.EnvironmentID, sid, ids[sid])
		if err != nil {
			return nil, errcode.CodeDBError
		}
		name := fmt.Sprintf("服务 #%d", sid)
		if len(versions) > 0 && versions[0].ServiceName != "" {
			name = versions[0].ServiceName
		}
		result = append(result, RollbackService{ServiceID: sid, Name: name, Provider: ids[sid], Versions: versions})
	}
	return result, 0
}

func (l *Logic) PreviewRollback(ctx context.Context, userID, originID, serviceID, snapshotID int64) (*RollbackPreview, int, string) {
	if originID <= 0 || serviceID <= 0 || snapshotID <= 0 {
		return nil, errcode.CodeParamInvalid, "请选择服务和成功版本"
	}
	if _, code := l.rollbackOrigin(ctx, userID, originID, serviceID); code != 0 {
		return nil, code, ""
	}
	repo, ok := l.repo.(rollbackRepository)
	if !ok {
		return nil, errcode.CodeDBError, ""
	}
	preview, err := repo.PreviewRollback(ctx, originID, serviceID, snapshotID)
	if err != nil {
		return nil, errcode.CodeParamInvalid, err.Error()
	}
	if preview == nil || preview.environment == nil || preview.snapshot == nil || preview.environment.ID != preview.EnvironmentID || preview.environment.ProjectID != preview.snapshot.ProjectID {
		return nil, errcode.CodeParamInvalid, "回滚环境归属不一致"
	}
	allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "deployment", Action: "rollback", ProjectID: preview.snapshot.ProjectID, EnvironmentID: preview.EnvironmentID, Environment: preview.environment.Code, ServiceID: serviceID, ClusterID: preview.ClusterID})
	if err != nil {
		return nil, errcode.CodeDBError, ""
	}
	if !allowed {
		return nil, errcode.CodeNoPermission, ""
	}
	return preview, 0, ""
}

// SubmitRollback redoes preview validation. The repository repeats it under the
// same project/service locks used by every normal submission.
func (l *Logic) SubmitRollback(ctx context.Context, userID, snapshotID int64, key string, req RollbackRequest) (int64, int, string) {
	if !rollbackRequestKeyPattern.MatchString(strings.TrimSpace(key)) || len(req.ExpectedToken) != 64 {
		return 0, errcode.CodeParamInvalid, "缺少回滚确认或请求标识，请重新预览"
	}
	if _, code := l.rollbackOrigin(ctx, userID, req.OriginRunID, req.ServiceID); code != 0 {
		return 0, code, ""
	}
	history, ok := l.repo.(rollbackRepository)
	if !ok {
		return 0, errcode.CodeDBError, ""
	}
	prior, err := history.FindRollbackRequest(ctx, strings.TrimSpace(key))
	if err != nil {
		return 0, errcode.CodeDBError, ""
	}
	if prior != nil {
		var frozen struct {
			Request RollbackRequest `json:"rollback_request"`
		}
		_ = json.Unmarshal(prior.ContextSnapshot, &frozen)
		if prior.TriggerUserID != userID || prior.SourceSnapshotID != snapshotID || frozen.Request != req {
			return 0, errcode.CodeParamInvalid, "回滚请求标识已被另一操作使用"
		}
		return prior.ID, 0, ""
	}
	preview, code, message := l.PreviewRollback(ctx, userID, req.OriginRunID, req.ServiceID, snapshotID)
	if code != 0 {
		return 0, code, message
	}
	if preview.Token != req.ExpectedToken {
		return 0, errcode.CodeParamInvalid, "当前版本、环境或工作流已变化；本次未回滚，请重新预览"
	}
	if preview.BlockedReason != "" {
		return 0, errcode.CodeParamInvalid, preview.BlockedReason
	}
	// Idempotent retries are handled by CreateRollbackRun before current-version
	// checks; do not reject a retry merely because the original has completed.
	snapshot := preview.snapshot
	config := map[string]interface{}{"snapshot_id": snapshot.ID, "service_id": snapshot.ServiceID, "service_revision_id": snapshot.ServiceRevisionID, "service_name": preview.ServiceName}
	kind := NodeYAMLRollback
	if snapshot.Provider == NodeHelmDeploy {
		kind = NodeHelmRollback
	}
	tasks := []TaskRun{}
	depends := []string(nil)
	status := RunPending
	if preview.RequiresApproval {
		status = RunPendingApproval
		depends = []string{"rollback-approval"}
		tasks = append(tasks, TaskRun{NodeKey: "rollback-approval", NodeType: NodeApproval, Status: RunPendingApproval, NodeSnapshot: mustJSON(Node{Key: "rollback-approval", Name: "回滚审批", Type: NodeApproval})})
	}
	node := Node{Key: "rollback-service", Name: "回滚 " + preview.ServiceName, Type: kind, DependsOn: depends, Config: mustJSON(config), TimeoutSeconds: 600}
	tasks = append(tasks, TaskRun{NodeKey: node.Key, NodeType: kind, Status: RunPending, DependsOn: depends, NodeSnapshot: mustJSON(node)})
	run := &WorkflowRun{WorkflowID: preview.WorkflowID, ProjectID: snapshot.ProjectID, EnvironmentID: preview.EnvironmentID, DefinitionVersion: preview.workflowVersion, TriggerType: "rollback", TriggerUserID: userID, Status: status, ContextSnapshot: mustJSON(map[string]interface{}{
		"operation": "rollback", "trigger_actor": authenticatedRunActor(ctx, userID), "target": preview.environment,
		"rollback_request": req, "deployment_snapshot_id": snapshot.ID, "service_id": snapshot.ServiceID, "service_revision_id": snapshot.ServiceRevisionID, "service_name": preview.ServiceName,
		"target_run_id": snapshot.WorkflowRunID, "target_run_number": preview.Target.RunNumber, "provider": snapshot.Provider, "expected_current_snapshot_id": preview.Current.SnapshotID, "expected_helm_revision": preview.Current.HelmRevision, "target_manifest_digest": snapshot.ManifestDigest,
	})}
	repo, ok := l.repo.(runRepository)
	if !ok {
		return 0, errcode.CodeDBError, ""
	}
	id, err := repo.CreateRollbackRun(ctx, run, tasks, snapshot.ID, strings.TrimSpace(key))
	if err != nil {
		return 0, errcode.CodeParamInvalid, err.Error()
	}
	return id, 0, ""
}

func (r *Repository) FindRollbackRequest(ctx context.Context, key string) (*WorkflowRun, error) {
	var run WorkflowRun
	err := r.db.QueryRowContext(ctx, `SELECT id,trigger_user_id,COALESCE(source_snapshot_id,0),context_snapshot FROM workflow_run WHERE request_key=? AND operation_type='rollback'`, key).Scan(&run.ID, &run.TriggerUserID, &run.SourceSnapshotID, &run.ContextSnapshot)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &run, err
}

type rollbackQuery interface {
	sourceQuery
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
}

func scanDeployment(q sourceQuery, ctx context.Context, where string, args ...interface{}) (*DeploymentSnapshot, error) {
	var s DeploymentSnapshot
	var values []byte
	err := q.QueryRowContext(ctx, `SELECT id,workflow_run_id,workflow_task_run_id,node_key,project_id,environment_id,service_id,service_revision_id,cluster_id,namespace,provider,manifest_digest,rendered_manifest,values_snapshot,helm_release_name,helm_revision,status,created_at FROM deployment_snapshot WHERE `+where, args...).Scan(&s.ID, &s.WorkflowRunID, &s.WorkflowTaskRunID, &s.NodeKey, &s.ProjectID, &s.EnvironmentID, &s.ServiceID, &s.ServiceRevisionID, &s.ClusterID, &s.Namespace, &s.Provider, &s.ManifestDigest, &s.RenderedManifest, &values, &s.HelmReleaseName, &s.HelmRevision, &s.Status, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	s.ValuesSnapshot = values
	if !s.VerifyManifestDigest() {
		return nil, fmt.Errorf("部署快照校验失败，禁止回滚")
	}
	return &s, nil
}

func deploymentVersion(ctx context.Context, q sourceQuery, s *DeploymentSnapshot) DeploymentVersion {
	v := describeDeployment(s)
	_ = q.QueryRowContext(ctx, "SELECT wr.workflow_id,COALESCE(wr.run_number,0),svc.name FROM workflow_run wr JOIN app_service svc ON svc.id=? WHERE wr.id=?", s.ServiceID, s.WorkflowRunID).Scan(&v.WorkflowID, &v.RunNumber, &v.ServiceName)
	return v
}

func describeDeployment(s *DeploymentSnapshot) DeploymentVersion {
	v := DeploymentVersion{SnapshotID: s.ID, RunID: s.WorkflowRunID, RevisionID: s.ServiceRevisionID, Digest: s.ManifestDigest, Status: s.Status, CreatedAt: s.CreatedAt, HelmRevision: s.HelmRevision, Images: []string{}}
	if s.Provider == NodeHelmDeploy {
		var values map[string]interface{}
		if json.Unmarshal(s.ValuesSnapshot, &values) == nil {
			if image, ok := values["_axiops_image"].(string); ok && image != "" {
				v.Images = append(v.Images, image)
			}
		}
	}
	if s.Provider == NodeYAMLDeploy {
		objects, err := ValidateYAMLManifest(s.RenderedManifest, s.Namespace)
		if err == nil {
			seen := map[string]bool{}
			for _, o := range objects {
				for _, group := range []string{"containers", "initContainers"} {
					items, _, _ := unstructured.NestedSlice(o.Object, append(workloadPodSpec(o), group)...)
					for _, raw := range items {
						c, _ := raw.(map[string]interface{})
						image, _ := c["image"].(string)
						if image != "" && !seen[image] {
							v.Images = append(v.Images, image)
							seen[image] = true
						}
					}
				}
			}
		}
		sort.Strings(v.Images)
	}
	return v
}

func (r *Repository) ListRollbackVersions(ctx context.Context, pid, eid, sid int64, provider string) ([]DeploymentVersion, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT ds.id,ds.workflow_run_id,ds.workflow_task_run_id,ds.node_key,ds.project_id,ds.environment_id,ds.service_id,ds.service_revision_id,ds.cluster_id,ds.namespace,ds.provider,ds.manifest_digest,ds.rendered_manifest,ds.values_snapshot,ds.helm_release_name,ds.helm_revision,ds.status,ds.created_at,wr.workflow_id,COALESCE(wr.run_number,0),svc.name FROM deployment_snapshot ds JOIN workflow_run wr ON wr.id=ds.workflow_run_id JOIN app_service svc ON svc.id=ds.service_id JOIN project_environment env ON env.id=ds.environment_id WHERE ds.project_id=? AND ds.environment_id=? AND ds.service_id=? AND ds.provider=? AND ds.status='applied' AND ds.cluster_id=env.cluster_id AND ds.namespace=env.namespace ORDER BY ds.id DESC LIMIT 100`, pid, eid, sid, provider)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []DeploymentVersion{}
	for rows.Next() {
		var s DeploymentSnapshot
		var values []byte
		var wid, number int64
		var name string
		if err := rows.Scan(&s.ID, &s.WorkflowRunID, &s.WorkflowTaskRunID, &s.NodeKey, &s.ProjectID, &s.EnvironmentID, &s.ServiceID, &s.ServiceRevisionID, &s.ClusterID, &s.Namespace, &s.Provider, &s.ManifestDigest, &s.RenderedManifest, &values, &s.HelmReleaseName, &s.HelmRevision, &s.Status, &s.CreatedAt, &wid, &number, &name); err != nil {
			return nil, err
		}
		if !s.VerifyManifestDigest() {
			return nil, fmt.Errorf("部署快照校验失败，禁止选择历史版本")
		}
		s.ValuesSnapshot = values
		v := describeDeployment(&s)
		v.WorkflowID = wid
		v.RunNumber = number
		v.ServiceName = name
		result = append(result, v)
	}
	return result, rows.Err()
}

func (r *Repository) PreviewRollback(ctx context.Context, originID, sid, snapshotID int64) (*RollbackPreview, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	p, err := loadRollbackPreview(ctx, tx, originID, sid, snapshotID)
	if err != nil {
		return nil, err
	}
	return p, tx.Commit()
}

func loadRollbackPreview(ctx context.Context, q rollbackQuery, originID, sid, snapshotID int64) (*RollbackPreview, error) {
	p := &RollbackPreview{OriginRunID: originID, ServiceID: sid, Warnings: []string{"回滚恢复所选版本的部署配置与镜像；不恢复数据库、PVC 数据或外部密码。"}}
	var pid, bound int64
	var workflowStatus string
	if err := q.QueryRowContext(ctx, `SELECT wr.project_id,wr.environment_id,wr.workflow_id,w.definition_version,w.target_environment_id,w.status FROM workflow_run wr JOIN workflow w ON w.id=wr.workflow_id AND w.project_id=wr.project_id WHERE wr.id=?`, originID).Scan(&pid, &p.EnvironmentID, &p.WorkflowID, &p.workflowVersion, &bound, &workflowStatus); err != nil {
		return nil, fmt.Errorf("原工作流不存在，无法创建回滚记录")
	}
	if workflowStatus != "active" || bound != p.EnvironmentID {
		return nil, fmt.Errorf("工作流目标环境已变化，请从当前环境工作流操作")
	}
	env := &EnvironmentTarget{}
	p.environment = env
	if err := q.QueryRowContext(ctx, `SELECT id,project_id,environment_type,binding_status,is_production,deploy_approval,status,code,cluster_id,namespace,host_group_ref FROM project_environment WHERE id=?`, p.EnvironmentID).Scan(&env.ID, &env.ProjectID, &env.EnvironmentType, &env.BindingStatus, &env.IsProduction, &env.DeployApproval, &env.Status, &env.Code, &env.ClusterID, &env.Namespace, &env.HostGroupRef); err != nil {
		return nil, fmt.Errorf("目标环境不存在")
	}
	if env.ProjectID != pid || env.Status != "active" || env.BindingStatus != "validated" || env.ClusterID <= 0 || env.Namespace == "" {
		return nil, fmt.Errorf("目标环境未就绪，禁止回滚")
	}
	if err := checkRollbackEnvironmentIsolation(ctx, q, env); err != nil {
		return nil, err
	}
	if err := q.QueryRowContext(ctx, "SELECT name FROM app_service WHERE id=? AND project_id=? AND status='active'", sid, pid).Scan(&p.ServiceName); err != nil {
		return nil, fmt.Errorf("服务已删除或不属于此项目")
	}
	target, err := scanDeployment(q, ctx, "id=? AND project_id=? AND environment_id=? AND service_id=? AND status='applied'", snapshotID, pid, p.EnvironmentID, sid)
	if err != nil {
		return nil, fmt.Errorf("目标成功部署快照不可用或不属于此服务和环境")
	}
	if target.Provider != NodeYAMLDeploy && target.Provider != NodeHelmDeploy {
		return nil, fmt.Errorf("此部署类型暂不支持快照回滚")
	}
	if target.ClusterID != env.ClusterID || target.Namespace != env.Namespace {
		return nil, fmt.Errorf("环境的集群或命名空间已变化，禁止恢复到旧目标")
	}
	current, err := scanDeployment(q, ctx, "project_id=? AND environment_id=? AND service_id=? ORDER BY id DESC LIMIT 1", pid, p.EnvironmentID, sid)
	if err != nil {
		return nil, fmt.Errorf("当前部署状态无法确认，请重新检查发布历史")
	}
	if current.ClusterID != env.ClusterID || current.Namespace != env.Namespace || current.Provider != target.Provider {
		return nil, fmt.Errorf("当前部署目标或类型与历史版本不一致，禁止回滚")
	}
	p.ClusterID, p.Namespace = env.ClusterID, env.Namespace
	p.RequiresApproval = env.IsProduction || env.DeployApproval
	p.snapshot = target
	p.Current = deploymentVersion(ctx, q, current)
	p.Target = deploymentVersion(ctx, q, target)
	p.CurrentManifest, p.TargetManifest = string(current.RenderedManifest), string(target.RenderedManifest)
	if current.Status == "applied" && current.ManifestDigest == target.ManifestDigest && (target.Provider != NodeHelmDeploy || current.HelmRevision == target.HelmRevision) {
		p.BlockedReason = "当前已是此版本，无需重复回滚"
	}
	if target.Provider == NodeYAMLDeploy {
		if len(p.Target.Images) == 0 {
			p.BlockedReason = "目标版本缺少容器镜像，无法作为应用回滚版本"
		}
		for _, image := range p.Target.Images {
			parts := strings.Split(image, "@")
			if len(parts) != 2 || !promotionDigest.MatchString(parts[1]) {
				p.BlockedReason = "目标版本没有完整的不可变镜像摘要，禁止使用可变标签回滚"
			}
		}
		oldObjects, err := ValidateYAMLManifest(current.RenderedManifest, current.Namespace)
		if err != nil {
			return nil, fmt.Errorf("当前资源清单无法校验，禁止回滚: %w", err)
		}
		newObjects, err := ValidateYAMLManifest(target.RenderedManifest, target.Namespace)
		if err != nil {
			return nil, err
		}
		present := map[string]bool{}
		for _, o := range newObjects {
			present[o.GetKind()+"/"+o.GetName()] = true
		}
		for _, o := range oldObjects {
			key := o.GetKind() + "/" + o.GetName()
			if !present[key] {
				if o.GetKind() == "PersistentVolumeClaim" {
					p.Warnings = append(p.Warnings, "保留现有 PVC "+o.GetName()+" 及其数据，不自动删除。")
					continue
				}
				p.BlockedReason = "目标版本不包含当前资源 " + key + "；为避免残留控制器或误删数据，请先明确处理资源差异"
				break
			}
		}
	} else {
		if target.HelmRevision <= 0 || target.HelmReleaseName == "" || current.HelmReleaseName != target.HelmReleaseName || (current.HelmRevision > 0 && current.HelmRevision <= target.HelmRevision) {
			p.BlockedReason = "Helm Release 或历史 revision 不满足回滚条件"
		}
		for _, inventory := range [][]byte{current.RenderedManifest, target.RenderedManifest} {
			if err := rejectHelmJobRollbackInventory(inventory); err != nil {
				p.BlockedReason = err.Error()
			}
		}
		currentView, _ := json.MarshalIndent(map[string]interface{}{"resources": json.RawMessage(current.RenderedManifest), "values": current.ValuesSnapshot}, "", "  ")
		targetView, _ := json.MarshalIndent(map[string]interface{}{"resources": json.RawMessage(target.RenderedManifest), "values": target.ValuesSnapshot}, "", "  ")
		p.CurrentManifest, p.TargetManifest = string(currentView), string(targetView)
		p.Warnings = append(p.Warnings, "Helm 对照为资源清单与脱敏参数引用；实际 values/Secret 以集群历史 Release 为准。执行前再次校验 revision、清单和镜像摘要。")
	}
	if err := checkRollbackResourceOwnership(ctx, q, target); err != nil {
		p.BlockedReason = err.Error()
	}
	if err := checkRollbackConflicts(ctx, q, pid, p.EnvironmentID, []int64{sid}, true, 0); err != nil {
		p.BlockedReason = err.Error()
	}
	tokenRaw, _ := json.Marshal([]interface{}{originID, p.WorkflowID, p.workflowVersion, env, sid, current.ID, current.Status, current.ManifestDigest, current.HelmRevision, target.ID, target.ManifestDigest, target.HelmRevision})
	sum := sha256.Sum256(tokenRaw)
	p.Token = hex.EncodeToString(sum[:])
	return p, nil
}

// Submission is serialized by the project lock. Existing active task rows are
// inspected without acquiring run/task locks in the opposite worker order.
func checkRollbackConflicts(ctx context.Context, q rollbackQuery, pid, eid int64, ids []int64, isRollback bool, excludeRun int64) error {
	if eid <= 0 || len(ids) == 0 {
		return nil
	}
	rows, err := q.QueryContext(ctx, `SELECT wr.operation_type,wt.node_type,wt.node_snapshot,wr.context_snapshot FROM workflow_run wr JOIN workflow_task_run wt ON wt.workflow_run_id=wr.id WHERE wr.project_id=? AND wr.environment_id=? AND wr.id<>? AND wr.status IN ('pending','running','pending_approval','cancelling')`, pid, eid, excludeRun)
	if err != nil {
		return err
	}
	defer rows.Close()
	wanted := map[int64]bool{}
	for _, id := range ids {
		wanted[id] = true
	}
	for rows.Next() {
		var operation, kind string
		var raw, contextRaw []byte
		if err = rows.Scan(&operation, &kind, &raw, &contextRaw); err != nil {
			return err
		}
		if !isRollback && operation != "rollback" && operation != "restart" {
			continue
		}
		if kind != NodeYAMLDeploy && kind != NodeHelmDeploy && kind != NodeHostDeploy && kind != NodeYAMLRollback && kind != NodeHelmRollback && kind != NodeWorkloadRestart {
			continue
		}
		var node Node
		var c struct {
			ServiceID int64 `json:"service_id"`
		}
		if json.Unmarshal(raw, &node) != nil || json.Unmarshal(node.Config, &c) != nil {
			return fmt.Errorf("已有发布任务配置无法确认，请等待其结束")
		}
		if c.ServiceID <= 0 {
			_ = json.Unmarshal(contextRaw, &c)
		}
		if c.ServiceID <= 0 || wanted[c.ServiceID] {
			return fmt.Errorf("该服务在此环境已有未结束的发布、回滚或重启，请结束后重新预览")
		}
	}
	return rows.Err()
}

func rollbackTaskServices(tasks []TaskRun) []int64 {
	ids := []int64{}
	for _, task := range tasks {
		if task.NodeType != NodeYAMLDeploy && task.NodeType != NodeHelmDeploy && task.NodeType != NodeHostDeploy && task.NodeType != NodeYAMLRollback && task.NodeType != NodeHelmRollback && task.NodeType != NodeWorkloadRestart {
			continue
		}
		var n Node
		var c struct {
			ServiceID int64 `json:"service_id"`
		}
		if json.Unmarshal(task.NodeSnapshot, &n) == nil && json.Unmarshal(n.Config, &c) == nil && c.ServiceID > 0 {
			ids = append(ids, c.ServiceID)
		}
	}
	return ids
}

// Only requests carrying a frozen rollback guard can enter the new page flow.
func validateRollbackSubmission(ctx context.Context, tx *sql.Tx, run *WorkflowRun, snapshotID int64) error {
	var frozen struct {
		Request *RollbackRequest `json:"rollback_request"`
	}
	if json.Unmarshal(run.ContextSnapshot, &frozen) != nil || frozen.Request == nil {
		return errors.New("缺少回滚预览，请刷新后重新操作")
	}
	p, err := loadRollbackPreview(ctx, tx, frozen.Request.OriginRunID, frozen.Request.ServiceID, snapshotID)
	if err != nil {
		return err
	}
	if p.Token != frozen.Request.ExpectedToken {
		return fmt.Errorf("当前版本、环境或工作流已变化；本次未回滚，请重新预览")
	}
	if p.BlockedReason != "" {
		return errors.New(p.BlockedReason)
	}
	if p.WorkflowID != run.WorkflowID || p.snapshot.ProjectID != run.ProjectID || p.EnvironmentID != run.EnvironmentID || p.workflowVersion != run.DefinitionVersion {
		return fmt.Errorf("回滚身份与预览不一致")
	}
	return nil
}
