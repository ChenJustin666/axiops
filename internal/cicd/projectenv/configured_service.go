package projectenv

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/ChenJustin666/axiops/internal/cicd/appsvc"
	"github.com/ChenJustin666/axiops/internal/cicd/serviceassets"
	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"github.com/ChenJustin666/axiops/internal/shared/middleware"
	"github.com/ChenJustin666/axiops/internal/shared/response"
)

type ConfiguredServiceRequest struct {
	Name       string                  `json:"name"`
	Code       string                  `json:"code"`
	Definition string                  `json:"definition"`
	Variables  map[string]*string      `json:"variables"`
	Assets     *serviceassets.Settings `json:"assets,omitempty"`
}

// CreateConfiguredService commits the directory, first revision and target
// configuration together. Invalid manifests never leave an incomplete service.
func (h *Handler) CreateConfiguredService(w http.ResponseWriter, r *http.Request) {
	var input ConfiguredServiceRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&input) != nil || !appsvc.ValidServiceIdentity(input.Name, input.Code) || len(input.Variables) > 200 {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	if err := appsvc.ValidateYAMLDefinition(input.Definition); err != nil {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, err.Error())
		return
	}
	uid, eid := middleware.GetUserID(r.Context()), pathID(r, "id")
	env, code := h.logic.Get(r.Context(), uid, eid)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	if !IsEnvironmentCompatible(env.EnvironmentType, env.BindingStatus, DeliveryTypeK8sYAML) {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	for _, permission := range []authz.Request{
		{Resource: "service", Action: "create", ProjectID: env.ProjectID},
		{Resource: "environment", Action: "update", ProjectID: env.ProjectID, EnvironmentID: eid, Environment: env.Code},
	} {
		allowed, err := h.logic.authorizer.Authorize(r.Context(), uid, permission)
		if err != nil {
			response.Error(w, errcode.CodeDBError)
			return
		}
		if !allowed {
			response.Error(w, errcode.CodeNoPermission)
			return
		}
	}
	repo, ok := h.logic.repo.(*Repository)
	if !ok {
		response.Error(w, errcode.CodeDBError)
		return
	}
	tx, err := repo.db.BeginTx(r.Context(), &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		response.Error(w, errcode.CodeDBError)
		return
	}
	defer tx.Rollback()
	sid, req, err := repo.createConfiguredServiceTx(r.Context(), tx, uid, env.ProjectID, eid, &input, func(ctx context.Context, tx *sql.Tx, eid int64, old, next serviceassets.Settings) error {
		return h.validateDraftAssets(ctx, tx, repo, uid, env.ProjectID, eid, old, next)
	})
	preview := r.URL.Query().Get("preview") == "1"
	if err == nil && !preview {
		err = tx.Commit()
	}
	if err != nil {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, err.Error())
		return
	}
	response.Success(w, map[string]interface{}{"service_id": sid, "saved": !preview, "validated": true, "configs": req.Result})
}

func (repo *Repository) createConfiguredServiceTx(ctx context.Context, tx *sql.Tx, uid, pid, eid int64, input *ConfiguredServiceRequest, validate func(context.Context, *sql.Tx, int64, serviceassets.Settings, serviceassets.Settings) error) (int64, *SaveConfigRequest, error) {
	var status, delivery, binding, envType string
	err := tx.QueryRowContext(ctx, `SELECT p.delivery_status,p.delivery_type,e.binding_status,e.environment_type FROM project p JOIN project_environment e ON e.project_id=p.id WHERE p.id=? AND e.id=? AND e.status='active' FOR UPDATE`, pid, eid).Scan(&status, &delivery, &binding, &envType)
	if err != nil || status != "active" || delivery != DeliveryTypeK8sYAML || !IsEnvironmentCompatible(envType, binding, delivery) {
		return 0, nil, fmt.Errorf("目标环境或项目已变更，请重新加载")
	}
	var count int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM app_service WHERE project_id=? AND code=?`, pid, strings.TrimSpace(input.Code)).Scan(&count); err != nil {
		return 0, nil, err
	}
	if count > 0 {
		return 0, nil, fmt.Errorf("服务名称已存在，请配置已有服务的目标环境，无需重复创建")
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO app_service(name,code,project_id,source_type,delivery_type,deploy_type,status,owner_user_id,created_by,repo_branch,dockerfile_path,build_context) VALUES(?,?,?,'template','k8s_yaml','k8s','active',?,?,'main','Dockerfile','.')`, strings.TrimSpace(input.Name), strings.TrimSpace(input.Code), pid, uid, uid)
	if err != nil {
		return 0, nil, err
	}
	sid, err := result.LastInsertId()
	if err != nil {
		return 0, nil, err
	}
	config, err := readServiceConfig(ctx, tx, eid, sid, true)
	if err != nil {
		return 0, nil, err
	}
	req := &SaveConfigRequest{Targets: []ConfigTarget{{EnvironmentID: eid, Token: config.Token}}, Definition: &input.Definition, Variables: input.Variables, Assets: input.Assets, validateAssets: validate}
	if !validConfigRequest(req) {
		return 0, nil, fmt.Errorf("配置参数无效")
	}
	if err = repo.saveServiceConfigTx(ctx, tx, uid, pid, sid, req); err != nil {
		return 0, nil, err
	}
	return sid, req, nil
}
