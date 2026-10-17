package workflow

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"github.com/ChenJustin666/axiops/internal/shared/projectstate"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Internal operation only: public workflow definitions cannot manufacture it.
const NodeWorkloadRestart = "workload_restart"

type RestartRequest struct {
	Kind          string `json:"kind"`
	Name          string `json:"name"`
	ExpectedToken string `json:"expected_token"`
	RequestKey    string `json:"request_key"`
}

type RestartGuard struct {
	ServiceID       int64  `json:"service_id"`
	SnapshotID      int64  `json:"snapshot_id"`
	WorkflowID      int64  `json:"workflow_id"`
	WorkflowVersion int64  `json:"workflow_version"`
	Kind            string `json:"kind"`
	Name            string `json:"name"`
	UID             string `json:"uid"`
	Generation      int64  `json:"generation"`
	SpecHash        string `json:"spec_hash"`
}

type RestartPreview struct {
	Token           string             `json:"token"`
	EnvironmentID   int64              `json:"environment_id"`
	EnvironmentName string             `json:"environment_name"`
	ClusterID       int64              `json:"cluster_id"`
	Namespace       string             `json:"namespace"`
	ServiceID       int64              `json:"service_id"`
	Kind            string             `json:"kind"`
	Name            string             `json:"name"`
	IsProduction    bool               `json:"is_production"`
	Images          []string           `json:"images"`
	Guard           RestartGuard       `json:"-"`
	Target          *EnvironmentTarget `json:"-"`
}

type restartContext struct {
	Target  *EnvironmentTarget `json:"target"`
	Guard   RestartGuard       `json:"restart"`
	Request RestartRequest     `json:"restart_request"`
	Actor   *RunActor          `json:"trigger_actor"`
}

type restartOwner struct {
	Guard    RestartGuard
	Snapshot *DeploymentSnapshot
}

func restartKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "deployment":
		return "Deployment"
	case "statefulset":
		return "StatefulSet"
	case "daemonset":
		return "DaemonSet"
	}
	return ""
}

// Only the latest snapshot per service is authoritative, including failed or
// prepared snapshots. An older successful version cannot grant ownership.
func restartOwners(ctx context.Context, q rollbackQuery, env *EnvironmentTarget) (map[string]restartOwner, error) {
	rows, err := q.QueryContext(ctx, `SELECT ds.id,ds.service_id,ds.status,ds.provider,ds.rendered_manifest,ds.manifest_digest,wr.workflow_id,w.definition_version,w.status,w.target_environment_id
	 FROM deployment_snapshot ds JOIN workflow_run wr ON wr.id=ds.workflow_run_id AND wr.project_id=ds.project_id
	 JOIN workflow w ON w.id=wr.workflow_id AND w.project_id=ds.project_id
	 JOIN app_service s ON s.id=ds.service_id AND s.project_id=ds.project_id AND s.status='active'
	 WHERE ds.project_id=? AND ds.environment_id=? AND ds.cluster_id=? AND ds.namespace=?
	 AND ds.id=(SELECT MAX(latest.id) FROM deployment_snapshot latest WHERE latest.project_id=ds.project_id AND latest.environment_id=ds.environment_id AND latest.service_id=ds.service_id)`, env.ProjectID, env.ID, env.ClusterID, env.Namespace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	owners := map[string]restartOwner{}
	for rows.Next() {
		s := &DeploymentSnapshot{ProjectID: env.ProjectID, EnvironmentID: env.ID, ClusterID: env.ClusterID, Namespace: env.Namespace}
		var wid, version, bound int64
		var status string
		if err := rows.Scan(&s.ID, &s.ServiceID, &s.Status, &s.Provider, &s.RenderedManifest, &s.ManifestDigest, &wid, &version, &status, &bound); err != nil {
			return nil, err
		}
		if s.Status != "applied" || s.Provider != NodeYAMLDeploy || status != "active" || bound != env.ID {
			continue
		}
		if !s.VerifyManifestDigest() {
			return nil, fmt.Errorf("部署快照校验失败")
		}
		objects, err := ValidateYAMLManifest(s.RenderedManifest, env.Namespace)
		if err != nil {
			return nil, err
		}
		for _, o := range objects {
			if restartKind(o.GetKind()) == "" {
				continue
			}
			key := o.GetKind() + "/" + o.GetName()
			if _, exists := owners[key]; exists {
				return nil, fmt.Errorf("工作负载归属冲突，禁止重启")
			}
			owners[key] = restartOwner{Guard: RestartGuard{ServiceID: s.ServiceID, SnapshotID: s.ID, WorkflowID: wid, WorkflowVersion: version, Kind: o.GetKind(), Name: o.GetName()}, Snapshot: s}
		}
	}
	return owners, rows.Err()
}

func (l *Logic) WithRestartFactory(f HelmRESTConfigFactory) *Logic { l.restartFactory = f; return l }

func (l *Logic) authorizeRestart(ctx context.Context, uid int64, env *EnvironmentTarget, g RestartGuard) int {
	if env == nil || env.ID <= 0 || env.ProjectID <= 0 || env.ClusterID <= 0 || env.Namespace == "" || env.Status != "active" || env.BindingStatus != "validated" || env.EnvironmentType != "k8s_namespace" {
		return errcode.CodeParamInvalid
	}
	for _, p := range [][2]string{{"environment", "view"}, {"service", "view"}, {"workflow", "view"}, {"deployment", "restart"}} {
		ok, err := l.authorizer.Authorize(ctx, uid, authz.Request{Resource: p[0], Action: p[1], ProjectID: env.ProjectID, EnvironmentID: env.ID, Environment: env.Code, ClusterID: env.ClusterID, ServiceID: g.ServiceID, WorkflowID: g.WorkflowID})
		if err != nil {
			return errcode.CodeDBError
		}
		if !ok {
			return errcode.CodeNoPermission
		}
	}
	return 0
}

func restartToken(env *EnvironmentTarget, g RestartGuard) string {
	b, _ := json.Marshal([]interface{}{env, g})
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func restartSpecHash(o *unstructured.Unstructured) string {
	copy := o.DeepCopy()
	annotations, _, _ := unstructured.NestedStringMap(copy.Object, "spec", "template", "metadata", "annotations")
	delete(annotations, restartAnnotation)
	delete(annotations, "kubectl.kubernetes.io/restartedAt")
	if len(annotations) == 0 {
		unstructured.RemoveNestedField(copy.Object, "spec", "template", "metadata", "annotations")
	} else {
		_ = unstructured.SetNestedStringMap(copy.Object, annotations, "spec", "template", "metadata", "annotations")
	}
	spec, _, _ := unstructured.NestedMap(copy.Object, "spec")
	b, _ := json.Marshal(spec)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func validateRestartWorkload(o *unstructured.Unstructured) error {
	if o.GetUID() == "" || o.GetResourceVersion() == "" || o.GetGeneration() <= 0 || o.GetDeletionTimestamp() != nil {
		return fmt.Errorf("工作负载已变化或正在删除，请刷新")
	}
	n := func(path ...string) int64 { v, _, _ := unstructured.NestedInt64(o.Object, path...); return v }
	strategy, _, _ := unstructured.NestedString(o.Object, "spec", "updateStrategy", "type")
	if o.GetKind() == "Deployment" {
		paused, _, _ := unstructured.NestedBool(o.Object, "spec", "paused")
		if paused {
			return fmt.Errorf("工作负载已暂停，请先恢复发布")
		}
		strategy, _, _ = unstructured.NestedString(o.Object, "spec", "strategy", "type")
	}
	if strategy != "" && strategy != "RollingUpdate" {
		return fmt.Errorf("当前更新策略不支持滚动重启")
	}
	if o.GetKind() == "StatefulSet" && n("spec", "updateStrategy", "rollingUpdate", "partition") != 0 {
		return fmt.Errorf("分区更新中的 StatefulSet 不支持整体滚动重启")
	}
	if o.GetKind() == "DaemonSet" {
		if n("status", "desiredNumberScheduled") <= 0 {
			return fmt.Errorf("没有可重启的目标节点")
		}
	} else {
		replicas, found, _ := unstructured.NestedInt64(o.Object, "spec", "replicas")
		if found && replicas <= 0 {
			return fmt.Errorf("副本数为 0，无需重启")
		}
	}
	return nil
}

func (l *Logic) PreviewRestart(ctx context.Context, uid, eid int64, kind, name string) (*RestartPreview, int, string) {
	r, ok := l.repo.(*Repository)
	if !ok || l.restartFactory == nil {
		return nil, errcode.CodeDBError, "重启服务未配置"
	}
	kind = restartKind(kind)
	name = strings.TrimSpace(name)
	if eid <= 0 || kind == "" || name == "" {
		return nil, errcode.CodeParamInvalid, "请选择工作负载"
	}
	env, err := r.FindEnvironmentTarget(ctx, eid)
	if err != nil {
		return nil, errcode.CodeDBError, "读取目标环境失败"
	}
	// Gate environment discovery before querying snapshots or Kubernetes.
	if env == nil {
		return nil, errcode.CodeNotFound, "环境不存在"
	}
	allowed, err := l.authorizer.Authorize(ctx, uid, authz.Request{Resource: "environment", Action: "view", ProjectID: env.ProjectID, EnvironmentID: env.ID, Environment: env.Code, ClusterID: env.ClusterID})
	if err != nil {
		return nil, errcode.CodeDBError, "权限校验失败"
	}
	if !allowed {
		return nil, errcode.CodeNoPermission, "无权查看此环境"
	}
	owners, err := restartOwners(ctx, r.db, env)
	if err != nil {
		return nil, errcode.CodeParamInvalid, "无法确认工作负载归属"
	}
	owner, ok := owners[kind+"/"+name]
	if !ok {
		return nil, errcode.CodeParamInvalid, "仅支持当前环境中平台已成功部署的 YAML 工作负载"
	}
	if code := l.authorizeRestart(ctx, uid, env, owner.Guard); code != 0 {
		return nil, code, "无权重启此服务"
	}
	if err := checkRollbackEnvironmentIsolation(ctx, r.db, env); err != nil {
		return nil, errcode.CodeParamInvalid, err.Error()
	}
	if err := checkRollbackResourceOwnership(ctx, r.db, owner.Snapshot); err != nil {
		return nil, errcode.CodeParamInvalid, err.Error()
	}
	if err := checkRollbackConflicts(ctx, r.db, env.ProjectID, eid, []int64{owner.Guard.ServiceID}, true, 0); err != nil {
		return nil, errcode.CodeParamInvalid, err.Error()
	}
	client, err := restartResourceClient(ctx, l.restartFactory, env.ClusterID, env.Namespace, kind)
	if err != nil {
		return nil, errcode.CodeClusterConnectFail, "连接集群失败"
	}
	o, err := client.Get(ctx, name, getRestartOptions)
	if err != nil {
		return nil, errcode.CodeClusterConnectFail, "读取工作负载失败，请刷新后重试"
	}
	if err := validateRestartWorkload(o); err != nil {
		return nil, errcode.CodeParamInvalid, err.Error()
	}
	g := owner.Guard
	g.UID = string(o.GetUID())
	g.Generation = o.GetGeneration()
	g.SpecHash = restartSpecHash(o)
	images := []string{}
	containers, _, _ := unstructured.NestedSlice(o.Object, "spec", "template", "spec", "containers")
	for _, v := range containers {
		c, _ := v.(map[string]interface{})
		if image, ok := c["image"].(string); ok {
			images = append(images, image)
		}
	}
	return &RestartPreview{Token: restartToken(env, g), EnvironmentID: eid, EnvironmentName: env.Code, ClusterID: env.ClusterID, Namespace: env.Namespace, ServiceID: g.ServiceID, Kind: kind, Name: name, IsProduction: env.IsProduction, Images: images, Guard: g, Target: env}, 0, ""
}

func (l *Logic) SubmitRestart(ctx context.Context, uid, eid int64, req RestartRequest) (int64, int, string) {
	r, ok := l.repo.(*Repository)
	if !ok {
		return 0, errcode.CodeDBError, "重启服务未配置"
	}
	req.Kind = restartKind(req.Kind)
	req.Name = strings.TrimSpace(req.Name)
	req.RequestKey = strings.TrimSpace(req.RequestKey)
	actor := authenticatedRunActor(ctx, uid)
	if actor == nil || eid <= 0 || req.Kind == "" || req.Name == "" || len(req.ExpectedToken) != 64 || !rollbackRequestKeyPattern.MatchString(req.RequestKey) {
		return 0, errcode.CodeParamInvalid, "重启请求不完整，请重新预览"
	}
	// Resolve a retry before obtaining a new preview: a successful patch changes generation.
	prior, err := findRestartRequest(ctx, r.db, req.RequestKey, uid, eid, req)
	if err != nil {
		return 0, errcode.CodeParamInvalid, "请求标识已被使用或读取失败"
	}
	if prior != nil {
		var frozen restartContext
		_ = json.Unmarshal(prior.ContextSnapshot, &frozen)
		env, err := r.FindEnvironmentTarget(ctx, eid)
		if err != nil {
			return 0, errcode.CodeDBError, "读取环境失败"
		}
		if code := l.authorizeRestart(ctx, uid, env, frozen.Guard); code != 0 {
			return 0, code, "无权读取此重启任务"
		}
		return prior.ID, 0, ""
	}
	p, code, message := l.PreviewRestart(ctx, uid, eid, req.Kind, req.Name)
	if code != 0 {
		return 0, code, message
	}
	if p.Token != req.ExpectedToken {
		return 0, errcode.CodeParamInvalid, "目标版本或环境已变化，请重新预览"
	}
	frozen := restartContext{Target: p.Target, Guard: p.Guard, Request: req, Actor: actor}
	run := WorkflowRun{WorkflowID: p.Guard.WorkflowID, ProjectID: p.Target.ProjectID, EnvironmentID: eid, DefinitionVersion: p.Guard.WorkflowVersion, TriggerType: "manual", OperationType: "restart", SourceSnapshotID: p.Guard.SnapshotID, TriggerUserID: uid, Status: RunPending, ContextSnapshot: mustJSON(frozen)}
	id, err := r.createRestartRun(ctx, &run, frozen)
	if err != nil {
		return 0, errcode.CodeParamInvalid, safeDiagnostic(err.Error())
	}
	return id, 0, ""
}

func findRestartRequest(ctx context.Context, q sourceQuery, key string, uid, eid int64, req RestartRequest) (*WorkflowRun, error) {
	var run WorkflowRun
	var raw []byte
	err := q.QueryRowContext(ctx, "SELECT id,environment_id,trigger_user_id,operation_type,context_snapshot FROM workflow_run WHERE request_key=?", key).Scan(&run.ID, &run.EnvironmentID, &run.TriggerUserID, &run.OperationType, &raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var before restartContext
	if json.Unmarshal(raw, &before) != nil || run.EnvironmentID != eid || run.TriggerUserID != uid || run.OperationType != "restart" || before.Request != req {
		return nil, fmt.Errorf("request key already used")
	}
	run.ContextSnapshot = raw
	return &run, nil
}

func (r *Repository) createRestartRun(ctx context.Context, run *WorkflowRun, frozen restartContext) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if err := projectstate.LockWritable(ctx, tx, run.ProjectID); err != nil {
		return 0, err
	}
	node := Node{Key: "restart", Name: "滚动重启 " + frozen.Guard.Name, Type: NodeWorkloadRestart, TimeoutSeconds: 600, Config: mustJSON(frozen.Guard)}
	task := TaskRun{NodeKey: node.Key, NodeType: node.Type, NodeSnapshot: mustJSON(node), Status: RunPending}
	if err := lockTaskServices(ctx, tx, run.ProjectID, []TaskRun{task}); err != nil {
		return 0, err
	}
	// Project lock serializes duplicate submissions with normal deployments.
	if prior, err := findRestartRequest(ctx, tx, frozen.Request.RequestKey, run.TriggerUserID, run.EnvironmentID, frozen.Request); err != nil || prior != nil {
		if prior != nil {
			return prior.ID, nil
		}
		return 0, err
	}
	if err := validateRestartDatabase(ctx, tx, *run, frozen); err != nil {
		return 0, err
	}
	number, err := allocateRunNumber(ctx, tx, run.WorkflowID)
	if err != nil {
		return 0, err
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO workflow_run(workflow_id,project_id,environment_id,definition_version,trigger_type,operation_type,source_snapshot_id,request_key,trigger_user_id,status,context_snapshot,run_number) VALUES(?,?,?,?,'manual','restart',?,?,?,'pending',?,?)`, run.WorkflowID, run.ProjectID, run.EnvironmentID, run.DefinitionVersion, run.SourceSnapshotID, frozen.Request.RequestKey, run.TriggerUserID, run.ContextSnapshot, number)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO workflow_task_run(workflow_run_id,node_key,node_type,depends_on,node_snapshot,status) VALUES(?,?,?,JSON_ARRAY(),?,'pending')`, id, task.NodeKey, task.NodeType, task.NodeSnapshot)
	if err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

func validateRestartDatabase(ctx context.Context, q rollbackQuery, run WorkflowRun, frozen restartContext) error {
	if frozen.Target == nil || frozen.Guard.ServiceID <= 0 || frozen.Guard.SnapshotID <= 0 || frozen.Target.ID != run.EnvironmentID || frozen.Target.ProjectID != run.ProjectID || frozen.Guard.WorkflowID != run.WorkflowID {
		return fmt.Errorf("重启目标快照无效")
	}
	var current EnvironmentTarget
	err := q.QueryRowContext(ctx, `SELECT id,project_id,environment_type,binding_status,is_production,deploy_approval,status,code,cluster_id,namespace,host_group_ref FROM project_environment WHERE id=?`, run.EnvironmentID).Scan(&current.ID, &current.ProjectID, &current.EnvironmentType, &current.BindingStatus, &current.IsProduction, &current.DeployApproval, &current.Status, &current.Code, &current.ClusterID, &current.Namespace, &current.HostGroupRef)
	if err != nil {
		return err
	}
	if current != *frozen.Target {
		return fmt.Errorf("目标环境已变化，请重新操作")
	}
	owners, err := restartOwners(ctx, q, &current)
	if err != nil {
		return err
	}
	owner, ok := owners[frozen.Guard.Kind+"/"+frozen.Guard.Name]
	if !ok || owner.Guard.ServiceID != frozen.Guard.ServiceID || owner.Guard.SnapshotID != frozen.Guard.SnapshotID || owner.Guard.WorkflowID != frozen.Guard.WorkflowID || owner.Guard.WorkflowVersion != frozen.Guard.WorkflowVersion {
		return fmt.Errorf("部署版本或工作流已变化，请重新预览")
	}
	if err := checkRollbackEnvironmentIsolation(ctx, q, &current); err != nil {
		return err
	}
	if err := checkRollbackResourceOwnership(ctx, q, owner.Snapshot); err != nil {
		return err
	}
	return checkRollbackConflicts(ctx, q, run.ProjectID, run.EnvironmentID, []int64{frozen.Guard.ServiceID}, true, run.ID)
}
