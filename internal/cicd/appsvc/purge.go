package appsvc

import (
	"context"
	"database/sql"

	"github.com/ChenJustin666/axiops/internal/cicd/helmconfig"
	"github.com/ChenJustin666/axiops/internal/cicd/serviceassets"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
)

// Purge removes a trashed service's editable configuration and identity. Shared
// templates, environment resources and workflow execution evidence remain intact.
// Old execution IDs cannot attach themselves to a newly created same-code service.
func (l *Logic) Purge(ctx context.Context, userID, id int64, confirmCode string) int {
	if _, code := l.lifecycleService(ctx, userID, id); code != 0 {
		return code
	}
	tx, err := l.repo.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return errcode.CodeDBError
	}
	defer tx.Rollback()
	var projectID int64
	var status, code string
	err = tx.QueryRowContext(ctx, `SELECT project_id,status,code FROM app_service WHERE id=? FOR UPDATE`, id).Scan(&projectID, &status, &code)
	if err == sql.ErrNoRows {
		return errcode.CodeServiceNotFound
	}
	if err != nil {
		return errcode.CodeDBError
	}
	if status != ServiceStatusDeleted || confirmCode == "" || confirmCode != code {
		return errcode.CodeParamInvalid
	}
	check, err := deletionCheck(ctx, tx, id, status)
	if err != nil {
		return errcode.CodeDBError
	}
	if !check.CanPurge {
		return errcode.CodeServiceInUse
	}
	statements := []struct {
		query string
		args  []interface{}
	}{
		{`DELETE vi FROM variable_item vi JOIN variable_set vs ON vs.id=vi.variable_set_id WHERE vs.service_id=? AND vs.project_id=?`, []interface{}{id, projectID}},
		{`DELETE FROM variable_set WHERE service_id=? AND project_id=?`, []interface{}{id, projectID}},
		{`DELETE er FROM environment_resource er JOIN project_environment e ON e.id=er.environment_id WHERE e.project_id=? AND ((er.kind=? AND er.name=?) OR (er.kind=? AND er.name=?))`, []interface{}{projectID, helmconfig.Kind, helmconfig.RowName(id), serviceassets.ResourceKind, serviceassets.RowName(id)}},
		{`DELETE FROM environment_service_yaml_revision WHERE service_id=?`, []interface{}{id}},
		{`DELETE FROM service_revision WHERE service_id=?`, []interface{}{id}},
		{`DELETE FROM service_revision_sequence WHERE service_id=?`, []interface{}{id}},
		{`DELETE FROM build_config WHERE service_id=?`, []interface{}{id}},
		{`DELETE FROM build_number_sequence WHERE service_id=?`, []interface{}{id}},
		{`DELETE FROM role_binding WHERE scope_type='service' AND scope_id=?`, []interface{}{id}},
		{`DELETE FROM app_service WHERE id=? AND status='deleted'`, []interface{}{id}},
	}
	for _, statement := range statements {
		if _, err = tx.ExecContext(ctx, statement.query, statement.args...); err != nil {
			return errcode.CodeDBError
		}
	}
	if err = tx.Commit(); err != nil {
		return errcode.CodeDBError
	}
	return errcode.CodeSuccess
}
