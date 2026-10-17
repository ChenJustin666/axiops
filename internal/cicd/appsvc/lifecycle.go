package appsvc

import (
	"context"
	"database/sql"
	"strings"
	"unicode"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
)

type ServiceReference struct {
	Kind     string `json:"kind"`
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Revision int64  `json:"revision"`
}
type DeletionCheck struct {
	CanDelete         bool               `json:"can_delete"`
	CanPurge          bool               `json:"can_purge"`
	Status            string             `json:"status"`
	Blockers          []ServiceReference `json:"blockers"`
	DeploymentHistory int64              `json:"deployment_history"`
}
type lifecycleQuery interface {
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

func deletionCheck(ctx context.Context, q lifecycleQuery, id int64, status string) (*DeletionCheck, error) {
	result := &DeletionCheck{Status: status, Blockers: []ServiceReference{}}
	// Retained history is deliberately not a live relationship. No cluster API is called.
	checks := []struct{ kind, query string }{
		{"environment", `SELECT e.id,e.name,o.revision FROM environment_service_override o JOIN project_environment e ON e.id=o.project_environment_id WHERE o.service_id=? ORDER BY e.id`},
		{"workflow", `SELECT DISTINCT w.id,w.name,0 FROM workflow_node n JOIN workflow w ON w.id=n.workflow_id WHERE CAST(JSON_UNQUOTE(JSON_EXTRACT(n.config_json,'$.service_id')) AS UNSIGNED)=? ORDER BY w.id`},
		{"build", `SELECT id,CONCAT('构建 #',build_number,' · ',status),0 FROM build_run WHERE service_id=? AND status NOT IN ('success','failed','timeout','cancelled') ORDER BY id`},
		{"build", `SELECT id,CONCAT('构建 #',build_number,' · 资源回收未结束'),0 FROM build_run WHERE service_id=? AND cleanup_pending=1 ORDER BY id`},
		{"build", `SELECT c.id,CONCAT('构建命令 #',c.id,' · ',c.status),0 FROM build_command c JOIN build_run b ON b.id=c.build_run_id WHERE b.service_id=? AND c.status NOT IN ('succeeded','failed','dead')`},
		{"run", `SELECT id,CONCAT('主机部署 #',id,' · ',status),0 FROM host_deployment_command WHERE service_id=? AND status NOT IN ('success','failed','rolled_back','cancelled')`},
		{"run", `SELECT id,CONCAT('主机发布 #',id,' · ',status),0 FROM host_deployment_rollout WHERE service_id=? AND status NOT IN ('success','failed','cancelled')`},
		{"run", `SELECT DISTINCT h.id,CONCAT('Helm 执行 #',h.id,' · ',h.status),0 FROM helm_execution h JOIN workflow_task_run t ON t.workflow_run_id=h.workflow_run_id AND t.node_key=h.node_key WHERE CAST(JSON_UNQUOTE(JSON_EXTRACT(t.node_snapshot,'$.config.service_id')) AS UNSIGNED)=? AND h.status NOT IN ('succeeded','failed')`},
		{"run", `SELECT DISTINCT r.id,CONCAT('工作流执行 #',r.id,' · ',r.status),0 FROM workflow_run r JOIN workflow_task_run t ON t.workflow_run_id=r.id WHERE CAST(JSON_UNQUOTE(JSON_EXTRACT(t.node_snapshot,'$.config.service_id')) AS UNSIGNED)=? AND (r.status NOT IN ('succeeded','failed','cancelled') OR t.status IN ('running','dispatching','cancelling')) ORDER BY r.id`},
	}
	for _, check := range checks {
		rows, err := q.QueryContext(ctx, check.query, id)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var item ServiceReference
			item.Kind = check.kind
			if err = rows.Scan(&item.ID, &item.Name, &item.Revision); err != nil {
				rows.Close()
				return nil, err
			}
			result.Blockers = append(result.Blockers, item)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
	}
	if err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM deployment_snapshot WHERE service_id=?`, id).Scan(&result.DeploymentHistory); err != nil {
		return nil, err
	}
	result.CanDelete = status != ServiceStatusDeleted && len(result.Blockers) == 0
	result.CanPurge = status == ServiceStatusDeleted && len(result.Blockers) == 0
	return result, nil
}

func (l *Logic) lifecycleService(ctx context.Context, userID, id int64) (*AppService, int) {
	if userID <= 0 || id <= 0 {
		return nil, errcode.CodeParamInvalid
	}
	s, err := l.repo.FindByID(ctx, id)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if s == nil {
		return nil, errcode.CodeServiceNotFound
	}
	allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "service", Action: "delete", ProjectID: s.ProjectID, ServiceID: s.ID})
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if !allowed {
		return nil, errcode.CodeNoPermission
	}
	return s, errcode.CodeSuccess
}

func (l *Logic) CheckDeletion(ctx context.Context, userID, id int64) (*DeletionCheck, int) {
	s, code := l.lifecycleService(ctx, userID, id)
	if code != errcode.CodeSuccess {
		return nil, code
	}
	check, err := deletionCheck(ctx, l.repo.db, id, s.Status)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	return check, errcode.CodeSuccess
}

func (l *Logic) DeleteChecked(ctx context.Context, userID, id int64) (*DeletionCheck, int) {
	if _, code := l.lifecycleService(ctx, userID, id); code != errcode.CodeSuccess {
		return nil, code
	}
	tx, err := l.repo.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, errcode.CodeDBError
	}
	defer tx.Rollback()
	var status string
	if err = tx.QueryRowContext(ctx, `SELECT status FROM app_service WHERE id=? FOR UPDATE`, id).Scan(&status); err != nil {
		return nil, errcode.CodeDBError
	}
	if status == ServiceStatusDeleted {
		return &DeletionCheck{Status: status, Blockers: []ServiceReference{}}, errcode.CodeSuccess
	}
	check, err := deletionCheck(ctx, tx, id, status)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if !check.CanDelete {
		return check, errcode.CodeServiceInUse
	}
	if _, err = tx.ExecContext(ctx, `UPDATE app_service SET status='deleted',updated_at=CURRENT_TIMESTAMP WHERE id=?`, id); err != nil {
		return nil, errcode.CodeDBError
	}
	if err = tx.Commit(); err != nil {
		return nil, errcode.CodeDBError
	}
	return check, errcode.CodeSuccess
}

func (l *Logic) Restore(ctx context.Context, userID, id int64) int {
	if _, code := l.lifecycleService(ctx, userID, id); code != errcode.CodeSuccess {
		return code
	}
	// Same id/code, revisions and history; no external resources or tasks are created.
	result, err := l.repo.db.ExecContext(ctx, `UPDATE app_service SET status='active',updated_at=CURRENT_TIMESTAMP WHERE id=? AND status='deleted'`, id)
	if err != nil {
		return errcode.CodeDBError
	}
	if n, err := result.RowsAffected(); err != nil {
		return errcode.CodeDBError
	} else if n == 0 {
		s, err := l.repo.FindByID(ctx, id)
		if err != nil {
			return errcode.CodeDBError
		}
		if s == nil {
			return errcode.CodeServiceNotFound
		}
	}
	return errcode.CodeSuccess
}

func (l *Logic) Rename(ctx context.Context, userID, id int64, name string) int {
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 64 || strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return errcode.CodeParamInvalid
	}
	s, code := l.GetByID(ctx, userID, id)
	if code != errcode.CodeSuccess {
		return code
	}
	allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "service", Action: "update", ProjectID: s.ProjectID, ServiceID: s.ID})
	if err != nil {
		return errcode.CodeDBError
	}
	if !allowed {
		return errcode.CodeNoPermission
	}
	tx, err := l.repo.db.BeginTx(ctx, nil)
	if err != nil {
		return errcode.CodeDBError
	}
	defer tx.Rollback()
	var oldName, serviceCode, status string
	if err = tx.QueryRowContext(ctx, `SELECT name,code,status FROM app_service WHERE id=? FOR UPDATE`, id).Scan(&oldName, &serviceCode, &status); err != nil {
		return errcode.CodeDBError
	}
	if status == ServiceStatusDeleted {
		return errcode.CodeParamInvalid
	}
	// Freeze the previous default image leaf before allowing human-readable labels.
	imageLeaf := oldName
	if !validServiceName(imageLeaf) {
		imageLeaf = serviceCode
	}
	if _, err = tx.ExecContext(ctx, `UPDATE app_service SET name=?,image_name=CASE WHEN image_name='' THEN ? ELSE image_name END,updated_at=CURRENT_TIMESTAMP WHERE id=?`, name, imageLeaf, id); err != nil {
		return errcode.CodeDBError
	}
	if err = tx.Commit(); err != nil {
		return errcode.CodeDBError
	}
	return errcode.CodeSuccess
}

// DetachEnvironment removes desired overrides only, never running resources or history.
func (l *Logic) DetachEnvironment(ctx context.Context, userID, id, environmentID, revision int64) int {
	if environmentID <= 0 || revision <= 0 {
		return errcode.CodeParamInvalid
	}
	s, code := l.lifecycleService(ctx, userID, id)
	if code != errcode.CodeSuccess {
		return code
	}
	var projectID, clusterID int64
	var envCode string
	if err := l.repo.db.QueryRowContext(ctx, `SELECT project_id,cluster_id,code FROM project_environment WHERE id=?`, environmentID).Scan(&projectID, &clusterID, &envCode); err != nil {
		return errcode.CodeEnvironmentNotFound
	}
	if projectID != s.ProjectID {
		return errcode.CodeNoPermission
	}
	allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "environment", Action: "update", ProjectID: projectID, EnvironmentID: environmentID, ClusterID: clusterID, Environment: envCode})
	if err != nil {
		return errcode.CodeDBError
	}
	if !allowed {
		return errcode.CodeNoPermission
	}
	tx, err := l.repo.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return errcode.CodeDBError
	}
	defer tx.Rollback()
	var status string
	if err = tx.QueryRowContext(ctx, `SELECT status FROM app_service WHERE id=? FOR UPDATE`, id).Scan(&status); err != nil {
		return errcode.CodeDBError
	}
	check, err := deletionCheck(ctx, tx, id, status)
	if err != nil {
		return errcode.CodeDBError
	}
	for _, ref := range check.Blockers {
		if ref.Kind == "build" || ref.Kind == "run" {
			return errcode.CodeServiceInUse
		}
	}
	result, err := tx.ExecContext(ctx, `DELETE FROM environment_service_override WHERE project_environment_id=? AND service_id=? AND revision=?`, environmentID, id, revision)
	if err != nil {
		return errcode.CodeDBError
	}
	n, err := result.RowsAffected()
	if err != nil {
		return errcode.CodeDBError
	}
	if n != 1 {
		return errcode.CodeParamInvalid
	}
	if err = tx.Commit(); err != nil {
		return errcode.CodeDBError
	}
	return errcode.CodeSuccess
}
