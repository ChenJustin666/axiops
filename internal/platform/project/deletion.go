package project

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/ChenJustin666/axiops/internal/shared/errcode"
)

type DeleteProjectRequest struct {
	ConfirmName string `json:"confirm_name"`
	Permanent   bool   `json:"permanent"`
}

type DeletionItem struct {
	Kind  string `json:"kind"`
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

type DeletionCheck struct {
	ProjectID   int64          `json:"project_id"`
	ProjectName string         `json:"project_name"`
	CanDelete   bool           `json:"can_delete"`
	Items       []DeletionItem `json:"items"`
	Blockers    []DeletionItem `json:"blockers"`
}

// Unknown task states remain blocking until their executor confirms completion.
var deletionTerminalStates = map[string]string{
	"build_run":               "'success','failed','timeout','cancelled'",
	"workflow_run":            "'succeeded','failed','cancelled'",
	"build_record":            "'success','succeeded','failed','timeout','cancelled','canceled'",
	"pipeline_run":            "'success','succeeded','failed','timeout','cancelled','canceled'",
	"release_record":          "'success','succeeded','failed','timeout','cancelled','canceled','rolled_back'",
	"helm_execution":          "'succeeded','failed'",
	"host_deployment_command": "'success','failed','rolled_back','cancelled'",
	"host_deployment_rollout": "'success','failed','cancelled'",
}

var deletionLabels = map[string]string{
	"app_service": "服务", "project_environment": "项目环境", "environment": "旧版环境",
	"workflow": "工作流", "workflow_node": "工作流节点", "variable_set": "变量组",
	"variable_item": "变量", "environment_resource": "环境 YAML 配置",
	"environment_service_override": "环境服务配置", "environment_service_yaml_revision": "环境 YAML 版本",
	"service_revision": "服务版本", "service_revision_sequence": "服务版本编号",
	"build_config": "构建配置", "build_number_sequence": "构建编号", "pipeline": "流水线配置",
	"role_binding": "权限绑定", "build_concurrency_policy": "构建并发配置",
	"notification_subscription": "通知订阅", "project_member": "项目成员",
	"build_run": "构建执行记录", "workflow_run": "工作流执行记录",
	"deployment_snapshot": "部署记录", "build_record": "旧版构建记录",
	"pipeline_run": "旧版流水线执行记录", "release_record": "旧版发布记录",
	"credential": "项目专属凭据", "code_source": "项目专属代码源",
	"image_registry": "项目专属镜像仓库", "host_agent": "主机代理",
	"host_agent_registration_token": "主机代理注册凭据",
	"build_log":                     "构建日志", "build_command": "构建命令", "workflow_task_run": "工作流任务与日志",
	"helm_execution": "Helm 执行记录", "host_deployment_command": "主机部署任务",
	"host_deployment_rollout": "主机发布批次", "notification_outbox": "通知发送记录",
	"webhook_delivery": "Webhook 记录",
}

type deletionScope struct {
	table, predicate string
	args             []interface{}
	parents          []string
	activePredicate  string
}

func deletionBlockerMessage(check *DeletionCheck) string {
	var parts []string
	for _, item := range check.Blockers {
		parts = append(parts, fmt.Sprintf("%s %d 项", item.Name, item.Count))
	}
	return "项目仍有未结束任务：" + strings.Join(parts, "、") + "；请等待完成或取消后再删除"
}

// Resolve ownership, including descendants without project_id. Parents are kept
// until all their children have been deleted so ownership predicates stay valid.
func deletionScopes(ctx context.Context, tx *sql.Tx, id int64) ([]deletionScope, error) {
	rows, err := tx.QueryContext(ctx, `SELECT c.TABLE_NAME,c.COLUMN_NAME FROM information_schema.COLUMNS c
	 JOIN information_schema.TABLES t ON t.TABLE_SCHEMA=c.TABLE_SCHEMA AND t.TABLE_NAME=c.TABLE_NAME
	 WHERE c.TABLE_SCHEMA=DATABASE() AND t.TABLE_TYPE='BASE TABLE' ORDER BY c.TABLE_NAME,c.ORDINAL_POSITION`)
	if err != nil {
		return nil, err
	}
	columns := map[string]map[string]bool{}
	for rows.Next() {
		var table, column string
		if err = rows.Scan(&table, &column); err != nil {
			rows.Close()
			return nil, err
		}
		if columns[table] == nil {
			columns[table] = map[string]bool{}
		}
		columns[table][column] = true
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	parents := map[string]string{
		"service_id": "app_service", "environment_id": "project_environment",
		"project_environment_id": "project_environment", "workflow_id": "workflow",
		"variable_set_id": "variable_set", "pipeline_id": "pipeline",
		"build_run_id": "build_run", "workflow_run_id": "workflow_run",
		"workflow_task_run_id": "workflow_task_run", "subscription_id": "notification_subscription",
	}
	resolved := map[string]deletionScope{}
	visiting := map[string]bool{}
	var resolve func(string) (deletionScope, error)
	resolve = func(table string) (deletionScope, error) {
		if scope, ok := resolved[table]; ok {
			return scope, nil
		}
		if table == "project" || table == "audit_log" {
			return deletionScope{}, nil
		}
		if visiting[table] {
			return deletionScope{}, fmt.Errorf("cyclic deletion ownership: %s", table)
		}
		visiting[table] = true
		defer delete(visiting, table)
		cols := columns[table]
		var predicates []string
		var args []interface{}
		var dependencies []string
		if cols["project_id"] {
			predicates = append(predicates, "project_id=?")
			args = append(args, id)
		}
		links := make(map[string]string, len(parents))
		for column, parent := range parents {
			links[column] = parent
		}
		if table == "webhook_delivery" {
			links["build_id"], links["source_id"] = "build_run", "code_source"
		}
		for column, parent := range links {
			if !cols[column] || !columns[parent]["id"] {
				continue
			}
			owner, err := resolve(parent)
			if err != nil {
				return deletionScope{}, err
			}
			if owner.predicate != "" {
				predicates = append(predicates, fmt.Sprintf("`%s` IN (SELECT id FROM `%s` WHERE %s)", column, parent, owner.predicate))
				args = append(args, owner.args...)
				dependencies = append(dependencies, parent)
			}
		}
		if cols["scope_type"] && cols["scope_id"] {
			predicates = append(predicates, "(scope_type='project' AND scope_id=?)")
			args = append(args, id)
			if table == "role_binding" {
				for scope, parent := range map[string]string{"service": "app_service", "environment": "project_environment", "workflow": "workflow", "build": "build_run"} {
					if columns[parent]["project_id"] {
						predicates = append(predicates, fmt.Sprintf("(scope_type='%s' AND scope_id IN (SELECT id FROM `%s` WHERE project_id=?))", scope, parent))
						args = append(args, id)
						dependencies = append(dependencies, parent)
					}
				}
			}
		}
		scope := deletionScope{table: table, args: args, parents: dependencies}
		if table == "build_run" && cols["cleanup_pending"] {
			scope.activePredicate = "cleanup_pending=1"
		}
		if table == "build_command" && cols["status"] {
			scope.activePredicate = "status='processing'"
		}
		if len(predicates) > 0 {
			scope.predicate = "(" + strings.Join(predicates, " OR ") + ")"
		}
		resolved[table] = scope
		return scope, nil
	}
	// Reverse topological order: children before parents, project last.
	var tables []string
	for table := range columns {
		tables = append(tables, table)
	}
	sort.Strings(tables)
	var scopes []deletionScope
	added := map[string]bool{}
	var add func(string) error
	add = func(table string) error {
		if added[table] {
			return nil
		}
		scope, err := resolve(table)
		if err != nil {
			return err
		}
		added[table] = true
		for _, parent := range scope.parents {
			if err := add(parent); err != nil {
				return err
			}
		}
		if scope.predicate != "" {
			scopes = append(scopes, scope)
		}
		return nil
	}
	for _, table := range tables {
		if err := add(table); err != nil {
			return nil, err
		}
	}
	for i, j := 0, len(scopes)-1; i < j; i, j = i+1, j-1 {
		scopes[i], scopes[j] = scopes[j], scopes[i]
	}
	return scopes, nil
}

func checkDeletion(ctx context.Context, tx *sql.Tx, id int64, name string) (*DeletionCheck, []deletionScope, error) {
	scopes, err := deletionScopes(ctx, tx, id)
	if err != nil {
		return nil, nil, err
	}
	check := &DeletionCheck{ProjectID: id, ProjectName: name, CanDelete: true, Items: []DeletionItem{}, Blockers: []DeletionItem{}}
	for _, scope := range scopes {
		var count int64
		// Identifiers originate from the database catalog, never the request.
		query := "SELECT COUNT(*) FROM `" + strings.ReplaceAll(scope.table, "`", "``") + "` WHERE " + scope.predicate
		if err = tx.QueryRowContext(ctx, query, scope.args...).Scan(&count); err != nil {
			return nil, nil, err
		}
		if count == 0 {
			continue
		}
		label := deletionLabels[scope.table]
		if label == "" {
			label = "关联数据（" + scope.table + "）"
		}
		item := DeletionItem{scope.table, label, count}
		check.Items = append(check.Items, item)
		activePredicate := scope.activePredicate
		if terminal, ok := deletionTerminalStates[scope.table]; ok {
			statusPredicate := "(status IS NULL OR status NOT IN (" + terminal + "))"
			if activePredicate != "" {
				activePredicate += " OR " + statusPredicate
			} else {
				activePredicate = statusPredicate
			}
		}
		if activePredicate != "" {
			var active int64
			if err = tx.QueryRowContext(ctx, query+" AND ("+activePredicate+")", scope.args...).Scan(&active); err != nil {
				return nil, nil, err
			}
			if active > 0 {
				check.Blockers = append(check.Blockers, DeletionItem{scope.table, label + "（执行或回收未结束）", active})
			}
		}
	}
	check.CanDelete = len(check.Blockers) == 0
	return check, scopes, nil
}

func (l *Logic) CheckDeletion(ctx context.Context, userID, id int64) (*DeletionCheck, int) {
	check, code := l.deleteChecked(ctx, userID, id, nil)
	return check, code
}

func (l *Logic) deleteChecked(ctx context.Context, userID, id int64, req *DeleteProjectRequest) (*DeletionCheck, int) {
	if userID <= 0 || id <= 0 {
		return nil, errcode.CodeParamInvalid
	}
	if code := l.authorize(ctx, userID, "project", "delete", id); code != errcode.CodeSuccess {
		return nil, code
	}
	tx, err := l.repo.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, errcode.CodeDBError
	}
	defer tx.Rollback()
	var name, status string
	err = tx.QueryRowContext(ctx, "SELECT name,status FROM project WHERE id=? FOR UPDATE", id).Scan(&name, &status)
	if err == sql.ErrNoRows {
		return nil, errcode.CodeProjectNotFound
	}
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if status == "deleted" && req != nil && !req.Permanent {
		return nil, errcode.CodeProjectDeleted
	}
	if req != nil && (req.ConfirmName == "" || req.ConfirmName != name) {
		return nil, errcode.CodeParamInvalid
	}
	check, scopes, err := checkDeletion(ctx, tx, id, name)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if req == nil {
		return check, errcode.CodeSuccess
	}
	if !check.CanDelete {
		return check, errcode.CodeProjectInUse
	}
	if req.Permanent {
		for _, scope := range scopes {
			query := "DELETE FROM `" + strings.ReplaceAll(scope.table, "`", "``") + "` WHERE " + scope.predicate
			if _, err = tx.ExecContext(ctx, query, scope.args...); err != nil {
				return nil, errcode.CodeDBError
			}
		}
		if _, err = tx.ExecContext(ctx, "DELETE FROM project WHERE id=?", id); err != nil {
			return nil, errcode.CodeDBError
		}
	} else {
		if _, err = tx.ExecContext(ctx, "UPDATE project SET status='deleted',updated_at=CURRENT_TIMESTAMP WHERE id=?", id); err != nil {
			return nil, errcode.CodeDBError
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, errcode.CodeDBError
	}
	return check, errcode.CodeSuccess
}
