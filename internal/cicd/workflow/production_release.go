package workflow

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
)

type ProductionCandidate struct {
	Verification
	Name             string `json:"name"`
	RunID            int64  `json:"run_id"`
	WorkflowID       int64  `json:"workflow_id"`
	RunNumber        int64  `json:"run_number"`
	BuildNumber      int64  `json:"build_number"`
	Commit           string `json:"commit"`
	Branch           string `json:"branch"`
	Recommended      bool   `json:"recommended"`
	BuiltAt          string `json:"built_at,omitempty"`
	CommitURL        string `json:"commit_url,omitempty"`
	CompareURL       string `json:"compare_url,omitempty"`
	SourceName       string `json:"source_name,omitempty"`
	DeploymentStatus string `json:"deployment_status,omitempty"`
}
type ProductionService struct {
	ServiceID  int64                 `json:"service_id"`
	Name       string                `json:"name"`
	Candidates []ProductionCandidate `json:"candidates"`
	Images     []ProductionCandidate `json:"images"`
}
type ProductionEnvironment struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Code      string `json:"code,omitempty"`
	ClusterID int64  `json:"-"`
}
type ProductionOptions struct {
	Policy          ProductionPolicy        `json:"policy"`
	SourceName      string                  `json:"source_name"`
	EnvironmentID   int64                   `json:"environment_id"`
	EnvironmentName string                  `json:"environment_name"`
	ClusterID       int64                   `json:"cluster_id"`
	Namespace       string                  `json:"namespace"`
	Sources         []ProductionEnvironment `json:"sources"`
	ImageSources    []ProductionEnvironment `json:"image_sources"`
	Services        []ProductionService     `json:"services"`
}
type ProductionPick struct {
	VerificationID int64  `json:"verification_id"`
	BuildID        int64  `json:"build_id,omitempty"`
	ExpectedToken  string `json:"expected_token"`
}
type ProductionSubmission struct {
	RequestKey string           `json:"request_key"`
	Picks      []ProductionPick `json:"picks"`
}
type ProductionPreview struct {
	Token            string              `json:"token"`
	Candidate        ProductionCandidate `json:"candidate"`
	EnvironmentID    int64               `json:"environment_id"`
	EnvironmentName  string              `json:"environment_name"`
	SourceName       string              `json:"source_name"`
	ClusterID        int64               `json:"cluster_id"`
	Namespace        string              `json:"namespace"`
	CurrentImage     string              `json:"current_image"`
	CurrentManifest  string              `json:"current_manifest"`
	TargetManifest   string              `json:"target_manifest"`
	BlockedReason    string              `json:"blocked_reason"`
	RequiresApproval bool                `json:"requires_approval"`
	// Execution metadata freezes the actual target input; the public preview does not expose it.
	source            *YAMLSource
	helmConfig        *HelmDeployConfig
	build             *PromotionBuild
	workflow          *Workflow
	environment       *EnvironmentTarget
	sourceEnvironment *EnvironmentTarget
	policy            ProductionPolicy
	currentID         int64
	configDigest      string
}
type ProductionGuard struct {
	BuildID              int64               `json:"build_id,omitempty"`
	ServiceID            int64               `json:"service_id"`
	VerificationID       int64               `json:"verification_id"`
	VerificationRevision int64               `json:"verification_revision"`
	SnapshotID           int64               `json:"snapshot_id"`
	CurrentID            int64               `json:"current_id"`
	Policy               ProductionPolicy    `json:"policy"`
	ImageRef             string              `json:"image_ref"`
	ConfigDigest         string              `json:"config_digest"`
	SourceTarget         EnvironmentTarget   `json:"source_target"`
	Version              ProductionCandidate `json:"version"`
}

func guardForProduction(p *ProductionPreview) ProductionGuard {
	if p.Candidate.Method == "build" {
		return ProductionGuard{ServiceID: p.Candidate.ServiceID, BuildID: p.build.ID, CurrentID: p.currentID, ImageRef: p.Candidate.ImageRef, ConfigDigest: p.configDigest, Version: p.Candidate}
	}
	return ProductionGuard{ServiceID: p.Candidate.ServiceID, VerificationID: p.Candidate.ID, VerificationRevision: p.Candidate.Revision, SnapshotID: p.Candidate.SnapshotID, CurrentID: p.currentID, Policy: p.policy, ImageRef: p.Candidate.ImageRef, ConfigDigest: p.configDigest, SourceTarget: *p.sourceEnvironment, Version: p.Candidate}
}

func checkProductionBatchResources(owners map[string]int64, p *ProductionPreview) error {
	objects, err := ValidateYAMLManifest([]byte(p.TargetManifest), p.Namespace)
	if p.helmConfig != nil {
		objects, err = ValidateHelmManifest([]byte(p.TargetManifest), p.Namespace)
	}
	if err != nil {
		return err
	}
	for _, object := range objects {
		identity := object.GroupVersionKind().Group + "/" + object.GetKind() + "/" + p.Namespace + "/" + object.GetName()
		if owner := owners[identity]; owner != 0 && owner != p.Candidate.ServiceID {
			return fmt.Errorf("服务 %d 与 %d 包含同名 %s/%s；请先明确资源归属，本次未提交", owner, p.Candidate.ServiceID, object.GetKind(), object.GetName())
		}
		owners[identity] = p.Candidate.ServiceID
	}
	return nil
}

// Preserve shared policy nodes while selecting one service's deployment graph.
func productionServiceNodes(nodes []Node, sid int64) ([]Node, error) {
	owners := map[string]int64{}
	types := map[string]string{}
	for _, n := range nodes {
		if n.Key == "" {
			return nil, fmt.Errorf("工作流节点缺少名称")
		}
		if _, exists := types[n.Key]; exists {
			return nil, fmt.Errorf("工作流节点名称重复")
		}
		types[n.Key] = n.Type
		if n.Type == NodeApproval || n.Type == NodeNotify {
			continue
		}
		if n.Type != NodeBuild && n.Type != NodeYAMLDeploy && n.Type != NodeHelmDeploy {
			return nil, fmt.Errorf("此生产晋级编排包含尚不支持的测试或部署节点")
		}
		var c struct {
			ServiceID int64 `json:"service_id"`
		}
		if json.Unmarshal(n.Config, &c) != nil || c.ServiceID <= 0 {
			return nil, fmt.Errorf("工作流节点缺少有效服务绑定")
		}
		owners[n.Key] = c.ServiceID
	}
	selected := []Node{}
	deploys := 0
	for _, n := range nodes {
		if owners[n.Key] != 0 && owners[n.Key] != sid {
			continue
		}
		deps := []string{}
		for _, d := range n.DependsOn {
			if _, exists := types[d]; !exists {
				return nil, fmt.Errorf("工作流依赖节点不存在")
			}
			if owners[d] == 0 || owners[d] == sid {
				deps = append(deps, d)
				continue
			}
			if n.Type != NodeNotify && !((n.Type == NodeYAMLDeploy || n.Type == NodeHelmDeploy) && (types[d] == NodeYAMLDeploy || types[d] == NodeHelmDeploy)) {
				return nil, fmt.Errorf("工作流存在跨服务依赖，不能单独发布")
			}
		}
		n.DependsOn = deps
		if n.Type == NodeYAMLDeploy || n.Type == NodeHelmDeploy {
			deploys++
		}
		selected = append(selected, n)
	}
	if deploys != 1 {
		return nil, fmt.Errorf("所选服务必须有一个 YAML/Helm 部署节点")
	}
	return selected, nil
}

func loadProductionWorkflow(ctx context.Context, q sourceQuery, id int64) (*Workflow, error) {
	w := &Workflow{ID: id}
	err := q.QueryRowContext(ctx, "SELECT project_id,target_environment_id,definition_version,status,delivery_type,name FROM workflow WHERE id=?", id).Scan(&w.ProjectID, &w.TargetEnvironmentID, &w.DefinitionVersion, &w.Status, &w.DeliveryType, &w.Name)
	if err != nil || w.Status != "active" || (w.DeliveryType != DeliveryTypeK8sYAML && w.DeliveryType != DeliveryTypeHelm) {
		return nil, fmt.Errorf("生产晋级需要有效的 YAML 工作流")
	}
	rows, err := q.QueryContext(ctx, "SELECT node_key,name,node_type,config_json,depends_on,timeout_seconds,retry_limit FROM workflow_node WHERE workflow_id=? ORDER BY ordinal,id", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var n Node
		var deps []byte
		if err = rows.Scan(&n.Key, &n.Name, &n.Type, &n.Config, &deps, &n.TimeoutSeconds, &n.RetryLimit); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(deps, &n.DependsOn); err != nil {
			return nil, err
		}
		w.Nodes = append(w.Nodes, n)
	}
	return w, rows.Err()
}
func (l *Logic) productionWorkflow(ctx context.Context, uid, wid int64) (*Repository, *Workflow, *EnvironmentTarget, error) {
	w, code := l.Get(ctx, uid, wid)
	if code != 0 {
		return nil, nil, nil, fmt.Errorf("无权访问工作流")
	}
	r, ok := l.repo.(*Repository)
	if !ok {
		return nil, nil, nil, fmt.Errorf("生产发布不可用")
	}
	e, err := productionEnv(ctx, r.db, w.TargetEnvironmentID)
	if err != nil {
		return nil, nil, nil, err
	}
	if !e.IsProduction || e.ProjectID != w.ProjectID || (w.DeliveryType != DeliveryTypeK8sYAML && w.DeliveryType != DeliveryTypeHelm) {
		return nil, nil, nil, fmt.Errorf("请选择绑定生产环境的 YAML 工作流")
	}
	if err = l.productionWorkflowPermission(ctx, uid, e, wid); err != nil {
		return nil, nil, nil, err
	}
	return r, w, e, nil
}
func (l *Logic) ProductionOptions(ctx context.Context, uid, wid int64) (*ProductionOptions, error) {
	r, w, e, err := l.productionWorkflow(ctx, uid, wid)
	if err != nil {
		return nil, err
	}
	p, err := readProductionPolicy(ctx, r.db, e.ID)
	if err != nil {
		return nil, err
	}
	out := &ProductionOptions{Policy: p, EnvironmentID: e.ID, ClusterID: e.ClusterID, Namespace: e.Namespace, Sources: []ProductionEnvironment{}, Services: []ProductionService{}}
	_ = r.db.QueryRowContext(ctx, "SELECT name FROM project_environment WHERE id=?", e.ID).Scan(&out.EnvironmentName)
	rows, err := r.db.QueryContext(ctx, `SELECT id,name FROM project_environment WHERE project_id=? AND is_production=0 AND status='active' AND binding_status='validated' AND environment_type='k8s_namespace' ORDER BY id`, e.ProjectID)
	if err != nil {
		return nil, err
	}
	sources := []ProductionEnvironment{}
	for rows.Next() {
		var s ProductionEnvironment
		if err = rows.Scan(&s.ID, &s.Name); err != nil {
			rows.Close()
			return nil, err
		}
		sources = append(sources, s)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	for _, s := range sources {
		source, err := productionEnv(ctx, r.db, s.ID)
		if err == nil && l.productionPermission(ctx, uid, source, "environment", "view") == nil {
			out.Sources = append(out.Sources, s)
			if s.ID == p.SourceEnvironmentID {
				out.SourceName = s.Name
			}
		}
	}
	all, err := r.db.QueryContext(ctx, "SELECT id,name,code,COALESCE(cluster_id,0) FROM project_environment WHERE project_id=? AND status='active' ORDER BY id", e.ProjectID)
	if err != nil {
		return nil, err
	}
	imageSources := []ProductionEnvironment{}
	for all.Next() {
		var source ProductionEnvironment
		if err = all.Scan(&source.ID, &source.Name, &source.Code, &source.ClusterID); err != nil {
			all.Close()
			return nil, err
		}
		imageSources = append(imageSources, source)
	}
	err = all.Err()
	all.Close()
	if err != nil {
		return nil, err
	}
	out.ImageSources = []ProductionEnvironment{}
	for _, source := range imageSources {
		allowed, err := l.authorizer.Authorize(ctx, uid, authz.Request{Resource: "environment", Action: "view", ProjectID: e.ProjectID, EnvironmentID: source.ID, Environment: source.Code, ClusterID: source.ClusterID})
		if err != nil {
			return nil, err
		}
		if allowed {
			out.ImageSources = append(out.ImageSources, source)
		}
	}
	ids := map[int64]bool{}
	for _, n := range w.Nodes {
		if n.Type == NodeYAMLDeploy || n.Type == NodeHelmDeploy {
			var c YAMLDeployConfig
			if json.Unmarshal(n.Config, &c) == nil && c.ServiceID > 0 {
				ids[c.ServiceID] = true
			}
		}
	}
	ordered := []int64{}
	for sid := range ids {
		ordered = append(ordered, sid)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	for _, sid := range ordered {
		allowed, err := l.authorizer.Authorize(ctx, uid, authz.Request{Resource: "service", Action: "view", ProjectID: e.ProjectID, ServiceID: sid})
		if err != nil {
			return nil, err
		}
		if !allowed {
			continue
		}
		s := ProductionService{ServiceID: sid, Name: fmt.Sprintf("服务 #%d", sid), Candidates: []ProductionCandidate{}}
		_ = r.db.QueryRowContext(ctx, "SELECT name FROM app_service WHERE id=? AND status='active'", sid).Scan(&s.Name)
		s.Images, err = listProductionImages(ctx, r.db, e.ProjectID, sid)
		if err != nil {
			return nil, err
		}
		visible := []ProductionCandidate{}
		for _, image := range s.Images {
			allowed, err := l.authorizer.Authorize(ctx, uid, authz.Request{Resource: "build", Action: "view", ProjectID: e.ProjectID, ServiceID: sid, BuildID: image.BuildID})
			if err != nil {
				return nil, err
			}
			if allowed {
				visible = append(visible, image)
			}
		}
		s.Images = visible
		allowedSources := map[int64]bool{}
		for _, source := range out.ImageSources {
			allowedSources[source.ID] = true
		}
		for i := range s.Images {
			if !allowedSources[s.Images[i].EnvironmentID] {
				s.Images[i].EnvironmentID = 0
				s.Images[i].SourceName = ""
				s.Images[i].RunID = 0
				s.Images[i].WorkflowID = 0
				s.Images[i].RunNumber = 0
				s.Images[i].DeploymentStatus = ""
			}
		}
		if out.SourceName != "" {
			s.Candidates, err = listProductionCandidates(ctx, r.db, e.ProjectID, p.SourceEnvironmentID, sid)
			if err != nil {
				return nil, err
			}
		}
		out.Services = append(out.Services, s)
	}
	return out, nil
}
func listProductionCandidates(ctx context.Context, q sourceQuery, pid, eid, sid int64) ([]ProductionCandidate, error) {
	rows, err := q.QueryContext(ctx, `SELECT v.snapshot_id FROM deployment_verification v JOIN deployment_snapshot ds ON ds.id=v.snapshot_id JOIN project_environment e ON e.id=v.environment_id WHERE v.project_id=? AND v.environment_id=? AND v.service_id=? AND v.status='passed' AND ds.status='applied' AND ds.cluster_id=e.cluster_id AND ds.namespace=e.namespace ORDER BY v.id DESC LIMIT 100`, pid, eid, sid)
	if err != nil {
		return nil, err
	}
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	result := []ProductionCandidate{}
	for _, id := range ids {
		v, err := readVerification(ctx, q, id)
		if err != nil {
			return nil, err
		}
		s, err := scanDeployment(q, ctx, "id=?", id)
		if err != nil {
			return nil, err
		}
		b, err := verifiedSnapshotBuild(ctx, q, s, v)
		if err != nil {
			continue
		}
		if v.ManifestDigest != s.ManifestDigest || v.ImageRef != b.ImageURL+"@"+b.ImageDigest {
			continue
		}
		c := candidateMetadata(ctx, q, v, s, b)
		result = append(result, c)
	}
	return result, nil
}
func candidateMetadata(ctx context.Context, q sourceQuery, v *Verification, s *DeploymentSnapshot, b *PromotionBuild) ProductionCandidate {
	c := ProductionCandidate{Verification: *v, RunID: s.WorkflowRunID, BuildNumber: b.BuildNumber, Commit: b.Commit, Branch: b.Branch}
	_ = q.QueryRowContext(ctx, "SELECT COALESCE(run_number,0),workflow_id FROM workflow_run WHERE id=?", s.WorkflowRunID).Scan(&c.RunNumber, &c.WorkflowID)
	_ = q.QueryRowContext(ctx, "SELECT name FROM app_service WHERE id=?", s.ServiceID).Scan(&c.Name)
	var latest int64
	_ = q.QueryRowContext(ctx, "SELECT MAX(id) FROM deployment_snapshot WHERE project_id=? AND environment_id=? AND service_id=?", s.ProjectID, s.EnvironmentID, s.ServiceID).Scan(&latest)
	c.Recommended = latest == s.ID
	if checkRollbackConflicts(ctx, q, s.ProjectID, s.EnvironmentID, []int64{s.ServiceID}, true, 0) != nil {
		c.Recommended = false
	}
	return c
}
func productionConfigDigest(source *YAMLSource) string {
	copy := *source
	copy.ImageRef = ""
	return DeploymentConfigDigest(&copy)
}
func loadProductionPreview(ctx context.Context, q sourceQuery, wid, verificationID int64) (*ProductionPreview, error) {
	w, err := loadProductionWorkflow(ctx, q, wid)
	if err != nil {
		return nil, err
	}
	e, err := productionEnv(ctx, q, w.TargetEnvironmentID)
	if err != nil {
		return nil, err
	}
	if !e.IsProduction || w.ProjectID != e.ProjectID {
		return nil, fmt.Errorf("工作流未绑定此项目的生产环境")
	}
	p, err := readProductionPolicy(ctx, q, e.ID)
	if err != nil {
		return nil, err
	}
	if p.SourceEnvironmentID <= 0 {
		return nil, fmt.Errorf("请先设置生产环境的版本来源")
	}
	sourceEnv, err := productionEnv(ctx, q, p.SourceEnvironmentID)
	if err != nil {
		return nil, err
	}
	if sourceEnv.ProjectID != e.ProjectID || sourceEnv.IsProduction {
		return nil, fmt.Errorf("版本来源环境不可用")
	}
	var snapshotID int64
	if err = q.QueryRowContext(ctx, "SELECT snapshot_id FROM deployment_verification WHERE id=? AND project_id=? AND environment_id=? AND status='passed'", verificationID, e.ProjectID, p.SourceEnvironmentID).Scan(&snapshotID); err != nil {
		return nil, fmt.Errorf("所选版本没有有效的测试通过记录")
	}
	v, err := readVerification(ctx, q, snapshotID)
	if err != nil {
		return nil, err
	}
	s, err := scanDeployment(q, ctx, "id=? AND project_id=? AND environment_id=? AND status='applied' AND provider=?", snapshotID, e.ProjectID, p.SourceEnvironmentID, NodeYAMLDeploy)
	if err != nil {
		return nil, fmt.Errorf("来源成功部署快照不可用")
	}
	if s.ClusterID != sourceEnv.ClusterID || s.Namespace != sourceEnv.Namespace || s.ManifestDigest != v.ManifestDigest {
		return nil, fmt.Errorf("来源环境或快照已变化")
	}
	b, err := verifiedSnapshotBuild(ctx, q, s, v)
	if err != nil {
		return nil, err
	}
	if b.ID != v.BuildID || v.ImageRef != b.ImageURL+"@"+b.ImageDigest {
		return nil, fmt.Errorf("测试镜像与构建记录不匹配")
	}
	return renderProductionPreview(ctx, q, w, e, candidateMetadata(ctx, q, v, s, b), b, sourceEnv, p)
}

func renderProductionPreview(ctx context.Context, q sourceQuery, w *Workflow, e *EnvironmentTarget, candidate ProductionCandidate, b *PromotionBuild, sourceEnv *EnvironmentTarget, p ProductionPolicy) (*ProductionPreview, error) {
	if w.DeliveryType == DeliveryTypeHelm {
		return renderHelmProductionPreview(ctx, q, w, e, candidate, b)
	}
	sid, imageRef := candidate.ServiceID, candidate.ImageRef
	if _, err := productionServiceNodes(w.Nodes, sid); err != nil {
		return nil, err
	}
	var rid int64
	if err := q.QueryRowContext(ctx, `SELECT COALESCE(ov.service_revision_id,svc.current_revision_id) FROM app_service svc JOIN environment_service_override ov ON ov.service_id=svc.id AND ov.project_environment_id=? WHERE svc.id=? AND svc.project_id=? AND svc.status='active'`, e.ID, sid, e.ProjectID).Scan(&rid); err != nil || rid <= 0 {
		return nil, fmt.Errorf("请先为该服务保存生产环境配置")
	}
	source, err := (&SQLYAMLSourceStore{db: q}).LoadYAMLSource(ctx, e.ID, sid, rid)
	if err != nil || source == nil {
		return nil, fmt.Errorf("生产配置无法读取")
	}
	preview := &ProductionPreview{Candidate: candidate, EnvironmentID: e.ID, ClusterID: e.ClusterID, Namespace: e.Namespace, RequiresApproval: e.IsProduction || e.DeployApproval, source: source, build: b, workflow: w, environment: e, sourceEnvironment: sourceEnv, policy: p, configDigest: productionConfigDigest(source)}
	source.ImageRef = imageRef
	target, err := NewDefaultYAMLResolver(&SQLYAMLSourceStore{db: q}).RenderYAML(ctx, WorkflowRun{ProjectID: e.ProjectID, EnvironmentID: e.ID}, YAMLDeployConfig{ServiceID: sid, ServiceRevisionID: rid, FrozenYAML: source})
	if err != nil {
		return nil, fmt.Errorf("生产配置校验失败: %w", err)
	}
	imageSnapshot := &DeploymentSnapshot{Provider: NodeYAMLDeploy, RenderedManifest: target.Manifest, Namespace: e.Namespace}
	images := describeDeployment(imageSnapshot).Images
	if len(images) != 1 || images[0] != imageRef {
		return nil, fmt.Errorf("生产配置不能准确映射到所选应用镜像，请核对容器与镜像配置")
	}
	preview.TargetManifest = string(target.Manifest)
	_ = q.QueryRowContext(ctx, "SELECT name FROM project_environment WHERE id=?", e.ID).Scan(&preview.EnvironmentName)
	if sourceEnv != nil {
		_ = q.QueryRowContext(ctx, "SELECT name FROM project_environment WHERE id=?", sourceEnv.ID).Scan(&preview.SourceName)
	}
	current, err := scanDeployment(q, ctx, "project_id=? AND environment_id=? AND service_id=? ORDER BY id DESC LIMIT 1", e.ProjectID, e.ID, sid)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if current != nil {
		preview.currentID = current.ID
		preview.CurrentManifest = string(current.RenderedManifest)
		preview.CurrentImage = strings.Join(describeDeployment(current).Images, ", ")
		if current.ClusterID != e.ClusterID || current.Namespace != e.Namespace {
			preview.BlockedReason = "生产环境已迁移，请先核对原运行资源"
		}
		var raw []byte
		if q.QueryRowContext(ctx, "SELECT context_snapshot FROM workflow_run WHERE id=?", current.WorkflowRunID).Scan(&raw) == nil {
			var old struct {
				Guards []ProductionGuard `json:"production_guards"`
			}
			_ = json.Unmarshal(raw, &old)
			for _, g := range old.Guards {
				if current.Status == "applied" && g.ServiceID == sid && g.ImageRef == imageRef && g.ConfigDigest == preview.configDigest {
					preview.BlockedReason = "生产已经发布此镜像及配置，无需重复发布"
				}
			}
		}
	}
	if err = checkRollbackEnvironmentIsolation(ctx, q, e); err != nil {
		preview.BlockedReason = err.Error()
	}
	if err = checkRollbackResourceOwnership(ctx, q, &DeploymentSnapshot{ProjectID: e.ProjectID, EnvironmentID: e.ID, ServiceID: sid, Provider: NodeYAMLDeploy, Namespace: e.Namespace, RenderedManifest: target.Manifest}); err != nil {
		preview.BlockedReason = err.Error()
	}
	if err = checkRollbackConflicts(ctx, q, e.ProjectID, e.ID, []int64{sid}, true, 0); err != nil {
		preview.BlockedReason = err.Error()
	}
	raw, _ := json.Marshal([]interface{}{w.ID, w.DefinitionVersion, e, sourceEnv, p, candidate, preview.configDigest, current})
	hash := sha256.Sum256(raw)
	preview.Token = hex.EncodeToString(hash[:])
	return preview, nil
}
func (l *Logic) ProductionPreview(ctx context.Context, uid, wid, vid int64) (*ProductionPreview, error) {
	return l.productionPickPreview(ctx, uid, wid, ProductionPick{VerificationID: vid})
}
func (l *Logic) productionPickPreview(ctx context.Context, uid, wid int64, pick ProductionPick) (*ProductionPreview, error) {
	r, _, _, err := l.productionWorkflow(ctx, uid, wid)
	if err != nil {
		return nil, err
	}
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	p, err := loadProductionPickPreview(ctx, tx, wid, pick)
	if err != nil {
		return nil, err
	}
	requests := []authz.Request{{Resource: "service", Action: "view", ProjectID: p.environment.ProjectID, ServiceID: p.Candidate.ServiceID}, {Resource: "build", Action: "view", ProjectID: p.environment.ProjectID, ServiceID: p.Candidate.ServiceID, BuildID: p.build.ID}}
	if p.sourceEnvironment != nil {
		requests = append(requests, authz.Request{Resource: "environment", Action: "view", ProjectID: p.environment.ProjectID, EnvironmentID: p.sourceEnvironment.ID, Environment: p.sourceEnvironment.Code, ClusterID: p.sourceEnvironment.ClusterID})
	}
	for _, a := range requests {
		allowed, err := l.authorizer.Authorize(ctx, uid, a)
		if err != nil || !allowed {
			return nil, fmt.Errorf("无权读取所选版本")
		}
	}
	return p, tx.Commit()
}
func (l *Logic) SubmitProduction(ctx context.Context, uid, wid int64, input ProductionSubmission) (int64, error) {
	r, w, e, err := l.productionWorkflow(ctx, uid, wid)
	if err != nil {
		return 0, err
	}
	if !rollbackRequestKeyPattern.MatchString(input.RequestKey) || len(input.Picks) == 0 || len(input.Picks) > 50 {
		return 0, fmt.Errorf("请选择版本并重新预览")
	}
	if id, err := r.findProductionRequest(ctx, r.db, uid, wid, input); err != nil || id > 0 {
		return id, err
	}
	guards := []ProductionGuard{}
	nodes := []Node{}
	seen := map[int64]bool{}
	resourceOwners := map[string]int64{}
	mode := "verified_image"
	if input.Picks[0].BuildID > 0 {
		mode = "selected_image"
	}
	for i, pick := range input.Picks {
		if (pick.BuildID > 0) != (mode == "selected_image") {
			return 0, fmt.Errorf("同一次发布不能混用两种版本来源")
		}
		p, err := l.productionPickPreview(ctx, uid, wid, pick)
		if err != nil {
			return 0, err
		}
		if len(pick.ExpectedToken) != 64 || p.Token != pick.ExpectedToken {
			return 0, fmt.Errorf("版本、配置或环境已变化，请重新预览")
		}
		if p.BlockedReason != "" {
			return 0, fmt.Errorf("%s", p.BlockedReason)
		}
		if err = checkProductionBatchResources(resourceOwners, p); err != nil {
			return 0, err
		}
		sid := p.Candidate.ServiceID
		if seen[sid] {
			return 0, fmt.Errorf("同一个服务只能选择一个版本")
		}
		seen[sid] = true
		selected, err := productionServiceNodes(w.Nodes, sid)
		if err != nil {
			return 0, err
		}
		for j := range selected {
			if selected[j].Type == NodeHelmDeploy && p.helmConfig != nil {
				cfg := *p.helmConfig
				cfg.StartOrder = i + 1
				selected[j].Config = mustJSON(cfg)
			}
			if selected[j].Type == NodeYAMLDeploy {
				selected[j].Config = mustJSON(YAMLDeployConfig{ServiceID: sid, ServiceRevisionID: p.source.ServiceRevisionID, FrozenYAML: p.source, StartOrder: i + 1})
			}
		}
		selected, err = promoteYAMLNodes(selected, sid, p.build)
		if err != nil {
			return 0, err
		}
		nodes = append(nodes, selected...)
		guards = append(guards, guardForProduction(p))
	}
	// Shared approval/notification ancestors are emitted once for a multi-service run.
	unique := []Node{}
	keys := map[string]bool{}
	for _, n := range nodes {
		if !keys[n.Key] {
			unique = append(unique, n)
			keys[n.Key] = true
		} else {
			for i := range unique {
				if unique[i].Key != n.Key {
					continue
				}
				for _, dep := range n.DependsOn {
					found := false
					for _, existing := range unique[i].DependsOn {
						if existing == dep {
							found = true
						}
					}
					if !found {
						unique[i].DependsOn = append(unique[i].DependsOn, dep)
					}
				}
			}
		}
	}
	status := RunPending
	if e.IsProduction || e.DeployApproval {
		if !hasApprovalNode(unique) {
			return 0, fmt.Errorf("生产环境现有策略要求审批节点，请检查工作流编排")
		}
		status = RunPendingApproval
	}
	run := &WorkflowRun{WorkflowID: wid, ProjectID: e.ProjectID, EnvironmentID: e.ID, DefinitionVersion: w.DefinitionVersion, TriggerType: "manual", Status: status, ContextSnapshot: mustJSON(map[string]interface{}{"target": e, "trigger_actor": authenticatedRunActor(ctx, uid), "execution_mode": mode, "production_submission": input, "production_guards": guards}), TriggerUserID: uid}
	tasks := []TaskRun{}
	for _, n := range unique {
		taskStatus := RunPending
		if n.Type == NodeApproval {
			taskStatus = RunPendingApproval
		}
		tasks = append(tasks, TaskRun{NodeKey: n.Key, NodeType: n.Type, DependsOn: n.DependsOn, NodeSnapshot: mustJSON(n), Status: taskStatus})
	}
	return r.CreateRun(ctx, run, tasks)
}
func (r *Repository) findProductionRequest(ctx context.Context, q sourceQuery, uid, wid int64, input ProductionSubmission) (int64, error) {
	var id, owner, workflow int64
	var raw []byte
	err := q.QueryRowContext(ctx, "SELECT id,trigger_user_id,workflow_id,context_snapshot FROM workflow_run WHERE request_key=?", input.RequestKey).Scan(&id, &owner, &workflow, &raw)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	var old struct {
		Submission ProductionSubmission `json:"production_submission"`
	}
	_ = json.Unmarshal(raw, &old)
	a, _ := json.Marshal(old.Submission)
	b, _ := json.Marshal(input)
	if owner != uid || workflow != wid || string(a) != string(b) {
		return 0, fmt.Errorf("请求标识已被其他操作使用")
	}
	return id, nil
}

func validateProductionSubmission(ctx context.Context, q sourceQuery, run *WorkflowRun, tasks []TaskRun) error {
	var frozen struct {
		Target     *EnvironmentTarget    `json:"target"`
		Mode       string                `json:"execution_mode"`
		Submission *ProductionSubmission `json:"production_submission"`
		Guards     []ProductionGuard     `json:"production_guards"`
	}
	if json.Unmarshal(run.ContextSnapshot, &frozen) != nil {
		return fmt.Errorf("发布快照无效")
	}
	if frozen.Target == nil || !frozen.Target.IsProduction {
		return nil
	}
	if frozen.Mode == "config_only" {
		return nil
	} // Existing guarded path preserves every deployed image.
	if frozen.Mode == "build" {
		return validateProductionBuildTasks(ctx, q, run, tasks)
	}
	if (frozen.Mode != "verified_image" && frozen.Mode != "selected_image") || frozen.Submission == nil || len(frozen.Guards) == 0 || len(frozen.Guards) != len(frozen.Submission.Picks) {
		return fmt.Errorf("生产发布必须选择测试验收通过的版本，请使用生产发布入口")
	}
	seen := map[int64]bool{}
	resourceOwners := map[string]int64{}
	for i, pick := range frozen.Submission.Picks {
		if (pick.BuildID > 0) != (frozen.Mode == "selected_image") {
			return fmt.Errorf("发布方式与所选镜像不一致")
		}
		p, err := loadProductionPickPreview(ctx, q, run.WorkflowID, pick)
		if err != nil {
			return err
		}
		if p.Token != pick.ExpectedToken || p.BlockedReason != "" || p.workflow.DefinitionVersion != run.DefinitionVersion {
			return fmt.Errorf("生产发布预览已失效，请重新检查版本与配置")
		}
		if err = checkProductionBatchResources(resourceOwners, p); err != nil {
			return err
		}
		g := frozen.Guards[i]
		if seen[g.ServiceID] || g != guardForProduction(p) {
			return fmt.Errorf("发布来源与预览不一致")
		}
		seen[g.ServiceID] = true
	}
	deploys := 0
	for _, task := range tasks {
		if task.NodeType == NodeBuild || task.NodeType == NodeTest || task.NodeType == NodeHostDeploy {
			return fmt.Errorf("此生产晋级编排包含尚不支持的构建或测试节点")
		}
		if task.NodeType == NodeHelmDeploy {
			c, err := helmDeployConfig(task.NodeSnapshot)
			if err != nil || !seen[c.ServiceID] || c.FrozenHelm == nil {
				return fmt.Errorf("生产 Helm 节点不属于所选服务")
			}
			matched := false
			for _, g := range frozen.Guards {
				if g.ServiceID == c.ServiceID && c.ImageRef == g.ImageRef && helmConfigurationDigest(c) == g.ConfigDigest {
					matched = true
				}
			}
			if !matched {
				return fmt.Errorf("生产 Helm 配置与镜像快照不一致")
			}
			deploys++
			continue
		}
		if task.NodeType != NodeYAMLDeploy {
			continue
		}
		c, err := yamlDeployConfig(task.NodeSnapshot)
		if err != nil || !seen[c.ServiceID] || c.FrozenYAML == nil {
			return fmt.Errorf("生产节点不属于所选服务")
		}
		deploys++
		matched := false
		for _, g := range frozen.Guards {
			if g.ServiceID == c.ServiceID && c.FrozenYAML.ImageRef == g.ImageRef && productionConfigDigest(c.FrozenYAML) == g.ConfigDigest {
				matched = true
			}
		}
		if !matched {
			return fmt.Errorf("生产配置与镜像快照不一致")
		}
	}
	if deploys != len(seen) {
		return fmt.Errorf("生产节点数量与服务选择不一致")
	}
	return nil
}

func (s *SQLDeploymentSnapshotStore) ValidateProductionExecution(ctx context.Context, run WorkflowRun, task TaskRun) error {
	env, err := productionEnv(ctx, s.db, run.EnvironmentID)
	if err != nil {
		return err
	}
	if !env.IsProduction {
		return nil
	}
	var f struct {
		Target *EnvironmentTarget `json:"target"`
		Mode   string             `json:"execution_mode"`
		Guards []ProductionGuard  `json:"production_guards"`
	}
	if json.Unmarshal(run.ContextSnapshot, &f) != nil || f.Target == nil || *env != *f.Target {
		return fmt.Errorf("生产环境绑定已变化")
	}
	if f.Mode == "config_only" {
		return nil
	}
	if f.Mode == "build" {
		return validateProductionBuildExecution(ctx, s.db, run, task, env)
	}
	if f.Mode != "verified_image" && f.Mode != "selected_image" {
		return fmt.Errorf("此生产任务没有已验证镜像来源，请重新发起生产发布")
	}
	cfg, err := yamlDeployConfig(task.NodeSnapshot)
	if err != nil {
		return err
	}
	for _, g := range f.Guards {
		if g.ServiceID == cfg.ServiceID {
			if f.Mode == "selected_image" {
				b, err := readProductionBuild(ctx, s.db, g.BuildID)
				if err != nil || b.ProjectID != run.ProjectID || b.ServiceID != g.ServiceID || b.ImageURL+"@"+b.ImageDigest != g.ImageRef || b.Commit != g.Version.Commit || b.Branch != g.Version.Branch {
					return fmt.Errorf("所选构建镜像已变化或不可用")
				}
			} else {
				v, err := readVerification(ctx, s.db, g.SnapshotID)
				if err != nil || v == nil || v.ID != g.VerificationID || v.Status != "passed" || v.Revision != g.VerificationRevision || v.ImageRef != g.ImageRef {
					return fmt.Errorf("测试验收记录已撤销或变化，已停止发布")
				}
				policy, err := readProductionPolicy(ctx, s.db, env.ID)
				if err != nil || policy != g.Policy {
					return fmt.Errorf("生产版本来源已变化")
				}
				source, err := productionEnv(ctx, s.db, policy.SourceEnvironmentID)
				if err != nil || *source != g.SourceTarget {
					return fmt.Errorf("测试来源环境绑定已变化")
				}
				snap, err := scanDeployment(s.db, ctx, "id=? AND status='applied'", g.SnapshotID)
				if err != nil || snap.ManifestDigest != v.ManifestDigest {
					return fmt.Errorf("测试部署快照无法验证")
				}
			}
			if cfg.FrozenYAML == nil || cfg.FrozenYAML.ImageRef != g.ImageRef || productionConfigDigest(cfg.FrozenYAML) != g.ConfigDigest {
				return fmt.Errorf("生产镜像或配置快照不一致")
			}
			var latest, latestRun int64
			var latestNode string
			err = s.db.QueryRowContext(ctx, "SELECT id,workflow_run_id,node_key FROM deployment_snapshot WHERE project_id=? AND environment_id=? AND service_id=? ORDER BY id DESC LIMIT 1", run.ProjectID, env.ID, g.ServiceID).Scan(&latest, &latestRun, &latestNode)
			if err != nil && err != sql.ErrNoRows {
				return err
			}
			if latest != g.CurrentID && (latestRun != run.ID || latestNode != task.NodeKey) {
				return fmt.Errorf("已有新的生产部署，旧任务已停止")
			}
			if err = checkRollbackConflicts(ctx, s.db, run.ProjectID, env.ID, []int64{g.ServiceID}, true, run.ID); err != nil {
				return err
			}
			if err = checkRollbackEnvironmentIsolation(ctx, s.db, env); err != nil {
				return err
			}
			target, err := NewDefaultYAMLResolver(&SQLYAMLSourceStore{db: s.db}).RenderYAML(ctx, run, cfg)
			if err != nil {
				return err
			}
			return checkRollbackResourceOwnership(ctx, s.db, &DeploymentSnapshot{ProjectID: run.ProjectID, EnvironmentID: env.ID, ServiceID: g.ServiceID, Provider: NodeYAMLDeploy, Namespace: env.Namespace, RenderedManifest: target.Manifest})
		}
	}
	return fmt.Errorf("缺少此服务的生产版本验证信息")
}
