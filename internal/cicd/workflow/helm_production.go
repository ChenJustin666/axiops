package workflow

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// Previews use credential references as placeholders. Actual credentials are
// resolved only when validating saved configuration or executing the release.
type helmPreviewDecryptor struct{}

func (helmPreviewDecryptor) Decrypt(string) (string, error) {
	return `{"value":"axiops-credential-redacted"}`, nil
}

func helmConfigurationDigest(config HelmDeployConfig) string {
	config.ImageRef = ""
	config.StartOrder = 0
	config.FrozenHelmVariables = append([]HelmVariable(nil), config.FrozenHelmVariables...)
	for i := range config.FrozenHelmVariables {
		config.FrozenHelmVariables[i].CredentialData = ""
	}
	data, _ := json.Marshal(config)
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func loadHelmEnvironmentConfig(ctx context.Context, q sourceQuery, eid, sid int64, release string) (HelmDeployConfig, error) {
	var rid int64
	if err := q.QueryRowContext(ctx, `SELECT ov.service_revision_id FROM environment_service_override ov JOIN app_service s ON s.id=ov.service_id JOIN project_environment e ON e.id=ov.project_environment_id AND e.project_id=s.project_id WHERE ov.project_environment_id=? AND ov.service_id=? AND s.status='active'`, eid, sid).Scan(&rid); err != nil || rid <= 0 {
		return HelmDeployConfig{}, fmt.Errorf("请先为该服务保存目标环境的 Helm 配置")
	}
	store := &SQLHelmSourceStore{db: q}
	source, err := store.LoadHelmSource(ctx, eid, sid, rid)
	if err != nil || source == nil {
		return HelmDeployConfig{}, fmt.Errorf("Helm 环境配置不可用")
	}
	vars, err := store.LoadHelmVariables(ctx, source.ProjectID, eid, sid)
	if err != nil {
		return HelmDeployConfig{}, err
	}
	for i := range vars {
		vars[i].CredentialData = ""
	}
	return HelmDeployConfig{ServiceID: sid, ServiceRevisionID: rid, ReleaseName: release, FrozenHelm: source, FrozenHelmVariables: vars}, nil
}

func renderHelmProductionPreview(ctx context.Context, q sourceQuery, w *Workflow, e *EnvironmentTarget, candidate ProductionCandidate, b *PromotionBuild) (*ProductionPreview, error) {
	if candidate.Method != "build" {
		return nil, fmt.Errorf("Helm 请选择成功构建的镜像")
	}
	nodes, err := productionServiceNodes(w.Nodes, candidate.ServiceID)
	if err != nil {
		return nil, err
	}
	var release string
	for _, n := range nodes {
		if n.Type == NodeHelmDeploy {
			var cfg HelmDeployConfig
			if json.Unmarshal(n.Config, &cfg) != nil {
				return nil, fmt.Errorf("Helm 发布配置无效")
			}
			release = cfg.ReleaseName
		}
	}
	cfg, err := loadHelmEnvironmentConfig(ctx, q, e.ID, candidate.ServiceID, release)
	if err != nil {
		return nil, err
	}
	cfg.ImageRef = candidate.ImageRef
	// Use frozen credential references with harmless placeholders in public output.
	previewStore := helmPreviewSourceStore{SQLHelmSourceStore: &SQLHelmSourceStore{db: q}}
	target, err := NewDefaultHelmResolver(previewStore, helmPreviewDecryptor{}).RenderHelm(ctx, WorkflowRun{ProjectID: e.ProjectID, EnvironmentID: e.ID}, cfg)
	if err != nil {
		return nil, fmt.Errorf("Helm 生产配置校验失败，请检查 Chart、values 和镜像映射")
	}
	manifest, err := renderHelmManifest(ctx, HelmRequest{Namespace: e.Namespace, ReleaseName: release, Chart: target.Chart, Values: target.Values}, false)
	if err != nil {
		return nil, fmt.Errorf("Helm 生产 Chart 渲染失败，请检查环境 values")
	}
	if err = validateHelmBuiltImage(manifest, e.Namespace, candidate.ImageRef); err != nil {
		return nil, err
	}
	public, err := redactedHelmPreview(manifest, e.Namespace, target.Values, target.SnapshotValues)
	if err != nil {
		return nil, err
	}
	p := &ProductionPreview{Candidate: candidate, EnvironmentID: e.ID, ClusterID: e.ClusterID, Namespace: e.Namespace, RequiresApproval: e.IsProduction || e.DeployApproval, build: b, workflow: w, environment: e, helmConfig: &cfg, TargetManifest: string(public), configDigest: helmConfigurationDigest(cfg)}
	_ = q.QueryRowContext(ctx, "SELECT name FROM project_environment WHERE id=?", e.ID).Scan(&p.EnvironmentName)
	current, err := scanDeployment(q, ctx, "project_id=? AND environment_id=? AND service_id=? ORDER BY id DESC LIMIT 1", e.ProjectID, e.ID, candidate.ServiceID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if current != nil {
		p.currentID = current.ID
		p.CurrentManifest = "# Helm 历史资源以 Release revision 保存；此处仅展示资源清单\n" + string(current.RenderedManifest)
		var raw []byte
		var values map[string]interface{}
		if json.Unmarshal(current.ValuesSnapshot, &values) == nil {
			if image, ok := values["_axiops_image"].(string); ok {
				p.CurrentImage = image
			}
		}
		if current.ClusterID != e.ClusterID || current.Namespace != e.Namespace {
			p.BlockedReason = "生产环境已迁移，请先核对原运行资源"
		}
		if q.QueryRowContext(ctx, "SELECT context_snapshot FROM workflow_run WHERE id=?", current.WorkflowRunID).Scan(&raw) == nil {
			var old struct {
				Guards []ProductionGuard `json:"production_guards"`
			}
			_ = json.Unmarshal(raw, &old)
			for _, g := range old.Guards {
				if current.Status == "applied" && g.ServiceID == candidate.ServiceID && g.ImageRef == candidate.ImageRef && g.ConfigDigest == p.configDigest {
					p.BlockedReason = "生产已发布此镜像及配置，无需重复发布"
				}
			}
		}
	}
	if err = checkRollbackEnvironmentIsolation(ctx, q, e); err != nil {
		p.BlockedReason = err.Error()
	}
	if err = checkRollbackResourceOwnership(ctx, q, &DeploymentSnapshot{ProjectID: e.ProjectID, EnvironmentID: e.ID, ServiceID: candidate.ServiceID, Provider: NodeHelmDeploy, Namespace: e.Namespace, HelmReleaseName: release}); err != nil {
		p.BlockedReason = err.Error()
	}
	if err = checkRollbackConflicts(ctx, q, e.ProjectID, e.ID, []int64{candidate.ServiceID}, true, 0); err != nil {
		p.BlockedReason = err.Error()
	}
	raw, _ := json.Marshal([]interface{}{w.ID, w.DefinitionVersion, e, candidate, p.configDigest, current})
	digest := sha256.Sum256(raw)
	p.Token = hex.EncodeToString(digest[:])
	return p, nil
}

// Frozen credential previews request only reference metadata and render
// placeholders; they never decrypt credentials for the public manifest.
type helmPreviewSourceStore struct{ *SQLHelmSourceStore }

func (s helmPreviewSourceStore) LoadFrozenCredential(ctx context.Context, v *HelmVariable) error {
	// Validate reference ownership/type; the plaintext remains a placeholder.
	v.CredentialData = ""
	return s.db.QueryRowContext(ctx, `SELECT type,project_id FROM credential WHERE id=?`, v.CredentialID).Scan(&v.CredentialType, &v.CredentialProjectID)
}

func (s *SQLDeploymentSnapshotStore) ValidateHelmProductionExecution(ctx context.Context, run WorkflowRun, task TaskRun, cfg HelmDeployConfig, target HelmTarget) error {
	env, err := productionEnv(ctx, s.db, run.EnvironmentID)
	if err != nil {
		return err
	}
	if !env.IsProduction {
		return nil
	}
	var frozen struct {
		Target *EnvironmentTarget `json:"target"`
		Mode   string             `json:"execution_mode"`
		Guards []ProductionGuard  `json:"production_guards"`
	}
	if json.Unmarshal(run.ContextSnapshot, &frozen) != nil || frozen.Target == nil || *env != *frozen.Target {
		return fmt.Errorf("生产环境绑定已变化")
	}
	if cfg.FrozenHelm == nil || target.ImageRef == "" {
		return fmt.Errorf("生产 Helm 缺少固定镜像或配置快照")
	}
	switch frozen.Mode {
	case "build":
		bid, err := matchingBuildID(run.Tasks, cfg.ServiceID)
		if err != nil || bid <= 0 {
			return fmt.Errorf("当前运行没有成功构建镜像")
		}
		b, err := readProductionBuild(ctx, s.db, bid)
		if err != nil || b.ProjectID != run.ProjectID || b.ServiceID != cfg.ServiceID || b.ImageURL+"@"+b.ImageDigest != target.ImageRef {
			return fmt.Errorf("构建镜像与生产服务不匹配")
		}
	case "selected_image":
		matched := false
		for _, g := range frozen.Guards {
			if g.ServiceID == cfg.ServiceID {
				b, err := readProductionBuild(ctx, s.db, g.BuildID)
				if err != nil || b.ProjectID != run.ProjectID || b.ServiceID != cfg.ServiceID || b.ImageURL+"@"+b.ImageDigest != target.ImageRef || cfg.ImageRef != g.ImageRef || helmConfigurationDigest(cfg) != g.ConfigDigest {
					return fmt.Errorf("生产 Helm 镜像或配置已变化")
				}
				matched = true
				var latest, latestRun int64
				var node string
				err = s.db.QueryRowContext(ctx, "SELECT id,workflow_run_id,node_key FROM deployment_snapshot WHERE project_id=? AND environment_id=? AND service_id=? ORDER BY id DESC LIMIT 1", run.ProjectID, env.ID, cfg.ServiceID).Scan(&latest, &latestRun, &node)
				if err != nil && err != sql.ErrNoRows {
					return err
				}
				if latest != g.CurrentID && (latestRun != run.ID || node != task.NodeKey) {
					return fmt.Errorf("已有新的生产部署，旧任务已停止")
				}
			}
		}
		if !matched {
			return fmt.Errorf("缺少此服务的生产镜像选择记录")
		}
	default:
		return fmt.Errorf("生产 Helm 请通过已有镜像或明确构建入口发布")
	}
	if err = checkRollbackConflicts(ctx, s.db, run.ProjectID, env.ID, []int64{cfg.ServiceID}, true, run.ID); err != nil {
		return err
	}
	var latestRun int64
	err = s.db.QueryRowContext(ctx, "SELECT workflow_run_id FROM deployment_snapshot WHERE project_id=? AND environment_id=? AND service_id=? ORDER BY id DESC LIMIT 1", run.ProjectID, env.ID, cfg.ServiceID).Scan(&latestRun)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if latestRun > run.ID {
		return fmt.Errorf("已有新的生产部署，旧任务已停止")
	}
	if err = checkRollbackResourceOwnership(ctx, s.db, &DeploymentSnapshot{ProjectID: run.ProjectID, EnvironmentID: env.ID, ServiceID: cfg.ServiceID, Provider: NodeHelmDeploy, HelmReleaseName: target.ReleaseName, Namespace: env.Namespace}); err != nil {
		return err
	}
	return checkRollbackEnvironmentIsolation(ctx, s.db, env)
}
