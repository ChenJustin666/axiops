package workflow

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/projectstate"
)

type ProductionPolicy struct {
	EnvironmentID       int64 `json:"environment_id"`
	SourceEnvironmentID int64 `json:"source_environment_id"`
	Revision            int64 `json:"revision"`
}
type Verification struct {
	ID             int64     `json:"id"`
	SnapshotID     int64     `json:"snapshot_id"`
	EnvironmentID  int64     `json:"environment_id"`
	ServiceID      int64     `json:"service_id"`
	BuildID        int64     `json:"build_id"`
	ImageRef       string    `json:"image_ref"`
	ManifestDigest string    `json:"manifest_digest"`
	Status         string    `json:"status"`
	Method         string    `json:"method"`
	Note           string    `json:"note"`
	Username       string    `json:"username"`
	Revision       int64     `json:"revision"`
	VerifiedAt     time.Time `json:"verified_at"`
}
type VerificationInput struct {
	SnapshotID     int64  `json:"snapshot_id"`
	ExpectedDigest string `json:"expected_digest"`
	Revision       int64  `json:"revision"`
	Note           string `json:"note"`
	Action         string `json:"action"`
}
type VerificationOption struct {
	ServiceID     int64         `json:"service_id"`
	Name          string        `json:"name"`
	SnapshotID    int64         `json:"snapshot_id"`
	Digest        string        `json:"digest"`
	ImageRef      string        `json:"image_ref"`
	Verification  *Verification `json:"verification,omitempty"`
	BlockedReason string        `json:"blocked_reason"`
}

func productionEnv(ctx context.Context, q sourceQuery, id int64) (*EnvironmentTarget, error) {
	e := &EnvironmentTarget{}
	err := q.QueryRowContext(ctx, `SELECT id,project_id,environment_type,binding_status,is_production,deploy_approval,status,code,cluster_id,namespace,host_group_ref FROM project_environment WHERE id=?`, id).Scan(&e.ID, &e.ProjectID, &e.EnvironmentType, &e.BindingStatus, &e.IsProduction, &e.DeployApproval, &e.Status, &e.Code, &e.ClusterID, &e.Namespace, &e.HostGroupRef)
	if err != nil {
		return nil, fmt.Errorf("环境不存在")
	}
	if e.Status != "active" || e.BindingStatus != "validated" || e.EnvironmentType != "k8s_namespace" || e.ClusterID <= 0 || e.Namespace == "" {
		return nil, fmt.Errorf("环境未就绪或类型不支持")
	}
	var status string
	if err = q.QueryRowContext(ctx, "SELECT status FROM project WHERE id=?", e.ProjectID).Scan(&status); err != nil || status == "deleted" {
		return nil, fmt.Errorf("项目不可用")
	}
	return e, nil
}
func (l *Logic) productionPermission(ctx context.Context, uid int64, e *EnvironmentTarget, resource, action string) error {
	ok, err := l.authorizer.Authorize(ctx, uid, authz.Request{Resource: resource, Action: action, ProjectID: e.ProjectID, EnvironmentID: e.ID, Environment: e.Code, ClusterID: e.ClusterID})
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("没有%s权限", action)
	}
	return nil
}
func (l *Logic) productionWorkflowPermission(ctx context.Context, uid int64, e *EnvironmentTarget, wid int64) error {
	ok, err := l.authorizer.Authorize(ctx, uid, authz.Request{Resource: "workflow", Action: "run", ProjectID: e.ProjectID, WorkflowID: wid, EnvironmentID: e.ID, Environment: e.Code, ClusterID: e.ClusterID})
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("没有执行此工作流的权限")
	}
	return nil
}
func readProductionPolicy(ctx context.Context, q sourceQuery, eid int64) (ProductionPolicy, error) {
	p := ProductionPolicy{EnvironmentID: eid}
	err := q.QueryRowContext(ctx, "SELECT source_environment_id,revision FROM production_release_policy WHERE environment_id=?", eid).Scan(&p.SourceEnvironmentID, &p.Revision)
	if err == sql.ErrNoRows {
		return p, nil
	}
	return p, err
}
func (l *Logic) ProductionPolicy(ctx context.Context, uid, eid int64, save *ProductionPolicy) (ProductionPolicy, error) {
	r, ok := l.repo.(*Repository)
	if !ok {
		return ProductionPolicy{}, fmt.Errorf("发布策略不可用")
	}
	e, err := productionEnv(ctx, r.db, eid)
	if err != nil {
		return ProductionPolicy{}, err
	}
	if !e.IsProduction {
		return ProductionPolicy{}, fmt.Errorf("仅生产环境需要配置版本来源")
	}
	action := "view"
	if save != nil {
		action = "manage_production"
	}
	if err = l.productionPermission(ctx, uid, e, "environment", action); err != nil {
		return ProductionPolicy{}, err
	}
	if save == nil {
		return readProductionPolicy(ctx, r.db, eid)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ProductionPolicy{}, err
	}
	defer tx.Rollback()
	if err = projectstate.LockWritable(ctx, tx, e.ProjectID); err != nil {
		return ProductionPolicy{}, err
	}
	current, err := productionEnv(ctx, tx, eid)
	if err != nil || *current != *e {
		return ProductionPolicy{}, fmt.Errorf("环境已变化，请重新打开")
	}
	source, err := productionEnv(ctx, tx, save.SourceEnvironmentID)
	if err != nil {
		return ProductionPolicy{}, err
	}
	if source.ProjectID != e.ProjectID || source.IsProduction || source.ID == e.ID || (source.ClusterID == e.ClusterID && source.Namespace == e.Namespace) {
		return ProductionPolicy{}, fmt.Errorf("来源应为本项目独立的非生产环境")
	}
	if err = l.productionPermission(ctx, uid, source, "environment", "view"); err != nil {
		return ProductionPolicy{}, err
	}
	p, err := readProductionPolicy(ctx, tx, eid)
	if err != nil {
		return p, err
	}
	if p.Revision != save.Revision {
		return p, fmt.Errorf("发布来源配置已变化，请重新加载")
	}
	if p.Revision == 0 {
		_, err = tx.ExecContext(ctx, "INSERT INTO production_release_policy(environment_id,project_id,source_environment_id,updated_by) VALUES(?,?,?,?)", eid, e.ProjectID, source.ID, uid)
	} else {
		_, err = tx.ExecContext(ctx, "UPDATE production_release_policy SET source_environment_id=?,revision=revision+1,updated_by=? WHERE environment_id=?", source.ID, uid, eid)
	}
	if err != nil {
		return p, err
	}
	if err = tx.Commit(); err != nil {
		return p, err
	}
	return readProductionPolicy(ctx, r.db, eid)
}
func readVerification(ctx context.Context, q sourceQuery, snapshotID int64) (*Verification, error) {
	v := &Verification{}
	err := q.QueryRowContext(ctx, `SELECT id,snapshot_id,environment_id,service_id,build_run_id,image_ref,manifest_digest,status,method,note,verified_username,revision,verified_at FROM deployment_verification WHERE snapshot_id=?`, snapshotID).Scan(&v.ID, &v.SnapshotID, &v.EnvironmentID, &v.ServiceID, &v.BuildID, &v.ImageRef, &v.ManifestDigest, &v.Status, &v.Method, &v.Note, &v.Username, &v.Revision, &v.VerifiedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return v, err
}
func snapshotBuild(ctx context.Context, q sourceQuery, s *DeploymentSnapshot) (*PromotionBuild, error) {
	images := describeDeployment(s).Images
	if len(images) != 1 {
		return nil, fmt.Errorf("需能准确匹配单一应用镜像；当前服务的多镜像版本暂不能晋级")
	}
	parts := strings.Split(images[0], "@")
	if len(parts) != 2 || !promotionDigest.MatchString(parts[1]) {
		return nil, fmt.Errorf("部署镜像缺少不可变摘要")
	}
	b := &PromotionBuild{}
	err := q.QueryRowContext(ctx, `SELECT id,project_id,service_id,build_number,status,image_url,image_digest,COALESCE(git_commit,''),COALESCE(git_branch,'') FROM build_run WHERE project_id=? AND service_id=? AND status='success' AND image_url=? AND image_digest=? ORDER BY id DESC LIMIT 1`, s.ProjectID, s.ServiceID, parts[0], parts[1]).Scan(&b.ID, &b.ProjectID, &b.ServiceID, &b.BuildNumber, &b.Status, &b.ImageURL, &b.ImageDigest, &b.Commit, &b.Branch)
	if err != nil || !validPromotionImage(b) {
		return nil, fmt.Errorf("找不到此部署镜像对应的成功构建")
	}
	return b, nil
}

func verifiedSnapshotBuild(ctx context.Context, q sourceQuery, s *DeploymentSnapshot, v *Verification) (*PromotionBuild, error) {
	if v == nil {
		return nil, fmt.Errorf("没有有效验收记录")
	}
	b := &PromotionBuild{}
	err := q.QueryRowContext(ctx, `SELECT id,project_id,service_id,build_number,status,image_url,image_digest,COALESCE(git_commit,''),COALESCE(git_branch,'') FROM build_run WHERE id=? AND project_id=? AND service_id=? AND status='success'`, v.BuildID, s.ProjectID, s.ServiceID).Scan(&b.ID, &b.ProjectID, &b.ServiceID, &b.BuildNumber, &b.Status, &b.ImageURL, &b.ImageDigest, &b.Commit, &b.Branch)
	images := describeDeployment(s).Images
	if err != nil || len(images) != 1 || !validPromotionImage(b) || images[0] != v.ImageRef || v.ImageRef != b.ImageURL+"@"+b.ImageDigest {
		return nil, fmt.Errorf("验收镜像与原构建记录不匹配")
	}
	return b, nil
}
func (l *Logic) VerificationOptions(ctx context.Context, uid, runID int64) ([]VerificationOption, error) {
	run, code := l.GetRun(ctx, uid, runID)
	if code != 0 {
		return nil, fmt.Errorf("无权查看运行记录")
	}
	r, ok := l.repo.(*Repository)
	if !ok {
		return nil, fmt.Errorf("验收记录不可用")
	}
	e, err := productionEnv(ctx, r.db, run.EnvironmentID)
	if err != nil {
		return nil, err
	}
	if e.IsProduction {
		return []VerificationOption{}, nil
	}
	if err = l.productionWorkflowPermission(ctx, uid, e, run.WorkflowID); err != nil {
		return nil, err
	}
	out := []VerificationOption{}
	for sid, provider := range rollbackServiceIDs(run) {
		if provider != NodeYAMLDeploy {
			continue
		}
		item := VerificationOption{ServiceID: sid, Name: fmt.Sprintf("服务 #%d", sid)}
		_ = r.db.QueryRowContext(ctx, "SELECT name FROM app_service WHERE id=?", sid).Scan(&item.Name)
		s, err := scanDeployment(r.db, ctx, "workflow_run_id=? AND service_id=? AND provider=? ORDER BY id DESC LIMIT 1", runID, sid, NodeYAMLDeploy)
		if err != nil {
			item.BlockedReason = "尚无可用部署快照"
		} else {
			item.SnapshotID, item.Digest = s.ID, s.ManifestDigest
			b, buildErr := snapshotBuild(ctx, r.db, s)
			if buildErr == nil {
				item.ImageRef = b.ImageURL + "@" + b.ImageDigest
			} else {
				item.BlockedReason = buildErr.Error()
			}
			var latest int64
			_ = r.db.QueryRowContext(ctx, "SELECT MAX(id) FROM deployment_snapshot WHERE project_id=? AND environment_id=? AND service_id=?", s.ProjectID, s.EnvironmentID, s.ServiceID).Scan(&latest)
			if s.Status != "applied" || latest != s.ID || s.ClusterID != e.ClusterID || s.Namespace != e.Namespace {
				item.BlockedReason = "此版本未完整成功或已不是当前环境的运行版本"
			}
			if err = checkRollbackConflicts(ctx, r.db, run.ProjectID, run.EnvironmentID, []int64{sid}, true, 0); err != nil {
				item.BlockedReason = err.Error()
			}
			item.Verification, err = readVerification(ctx, r.db, s.ID)
			if err != nil {
				return nil, err
			}
		}
		out = append(out, item)
	}
	return out, nil
}
func (l *Logic) SaveVerification(ctx context.Context, uid, runID int64, input VerificationInput) (*Verification, error) {
	run, code := l.GetRun(ctx, uid, runID)
	if code != 0 {
		return nil, fmt.Errorf("无权查看运行记录")
	}
	r, ok := l.repo.(*Repository)
	if !ok {
		return nil, fmt.Errorf("验收不可用")
	}
	e, err := productionEnv(ctx, r.db, run.EnvironmentID)
	if err != nil {
		return nil, err
	}
	if e.IsProduction {
		return nil, fmt.Errorf("生产运行不能作为测试验收来源")
	}
	if err = l.productionWorkflowPermission(ctx, uid, e, run.WorkflowID); err != nil {
		return nil, err
	}
	input.Note = strings.TrimSpace(input.Note)
	if len(input.Note) == 0 || len(input.Note) > 1024 || (input.Action != "pass" && input.Action != "revoke") {
		return nil, fmt.Errorf("请填写测试结论，长度不超过1024字节")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err = projectstate.LockWritable(ctx, tx, run.ProjectID); err != nil {
		return nil, err
	}
	currentEnv, err := productionEnv(ctx, tx, e.ID)
	if err != nil || *currentEnv != *e {
		return nil, fmt.Errorf("环境绑定已变化")
	}
	s, err := scanDeployment(tx, ctx, "id=? AND project_id=? AND workflow_run_id=? AND environment_id=? AND provider=? AND status='applied'", input.SnapshotID, run.ProjectID, runID, e.ID, NodeYAMLDeploy)
	if err != nil {
		return nil, fmt.Errorf("请选择此运行已成功的部署版本")
	}
	if s.ManifestDigest != input.ExpectedDigest || s.Namespace != e.Namespace || s.ClusterID != e.ClusterID {
		return nil, fmt.Errorf("部署版本或环境已变化，请重新加载")
	}
	before, err := readVerification(ctx, tx, s.ID)
	if err != nil {
		return nil, err
	}
	if before != nil && before.Revision != input.Revision || before == nil && input.Revision != 0 {
		return nil, fmt.Errorf("验收记录已变化，请重新加载")
	}
	username := fmt.Sprintf("账号 #%d", uid)
	if actor := authenticatedRunActor(ctx, uid); actor != nil {
		username = actor.Username
	}
	if input.Action == "revoke" {
		if before == nil || before.Status != "passed" {
			return nil, fmt.Errorf("此版本没有有效验收记录")
		}
		_, err = tx.ExecContext(ctx, "UPDATE deployment_verification SET status='revoked',revision=revision+1 WHERE id=?", before.ID)
	} else {
		var latest int64
		if err = tx.QueryRowContext(ctx, "SELECT MAX(id) FROM deployment_snapshot WHERE project_id=? AND environment_id=? AND service_id=?", s.ProjectID, s.EnvironmentID, s.ServiceID).Scan(&latest); err != nil {
			return nil, err
		}
		if latest != s.ID {
			return nil, fmt.Errorf("只能验收当前测试环境运行的版本，请切换到最新部署记录")
		}
		if err = checkRollbackConflicts(ctx, tx, s.ProjectID, s.EnvironmentID, []int64{s.ServiceID}, true, 0); err != nil {
			return nil, err
		}
		if before != nil && before.Status == "passed" {
			return nil, fmt.Errorf("此版本已验收，无需重复操作")
		}
		b, err := snapshotBuild(ctx, tx, s)
		if err != nil {
			return nil, err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO deployment_verification(project_id,environment_id,service_id,snapshot_id,build_run_id,image_ref,manifest_digest,note,verified_by,verified_username) VALUES(?,?,?,?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE status='passed',method='manual',note=VALUES(note),verified_by=VALUES(verified_by),verified_username=VALUES(verified_username),verified_at=CURRENT_TIMESTAMP,revision=revision+1`, s.ProjectID, s.EnvironmentID, s.ServiceID, s.ID, b.ID, b.ImageURL+"@"+b.ImageDigest, s.ManifestDigest, input.Note, uid, username)
		if err != nil {
			return nil, err
		}
	}
	if err != nil {
		return nil, err
	}
	after, err := readVerification(ctx, tx, s.ID)
	if err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO deployment_verification_event(project_id,verification_id,action,actor_id,actor_username,note) VALUES(?,?,?,?,?,?)", run.ProjectID, after.ID, input.Action, uid, username, input.Note); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return after, nil
}
