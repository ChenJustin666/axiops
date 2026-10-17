package projectenv

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ChenJustin666/axiops/internal/cicd/serviceassets"
	"github.com/ChenJustin666/axiops/internal/cicd/servicestate"
	"github.com/ChenJustin666/axiops/internal/cicd/workflow"
	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"github.com/ChenJustin666/axiops/internal/shared/middleware"
	"github.com/ChenJustin666/axiops/internal/shared/response"
)

func (h *Handler) SaveServiceAssets(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token  string                 `json:"token"`
		Assets serviceassets.Settings `json:"assets"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20)).Decode(&req) != nil || len(req.Token) != 64 {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	if err := serviceassets.Validate(req.Assets); err != nil {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, err.Error())
		return
	}
	uid, eid, sid := middleware.GetUserID(r.Context()), pathID(r, "id"), pathID(r, "service_id")
	env, svc, code := h.logic.authorizeEnvironmentService(r.Context(), uid, eid, sid, "update")
	if code != 0 {
		response.Error(w, code)
		return
	}
	if svc.DeliveryType != DeliveryTypeK8sYAML || !IsEnvironmentCompatible(env.EnvironmentType, env.BindingStatus, svc.DeliveryType) {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "此表单用于Kubernetes YAML服务；Helm请使用Chart/values配置")
		return
	}
	repo, ok := h.logic.repo.(*Repository)
	if !ok {
		response.Error(w, errcode.CodeDBError)
		return
	}
	ctx := r.Context()
	tx, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		response.Error(w, errcode.CodeDBError)
		return
	}
	defer tx.Rollback()
	if err = servicestate.LockActive(ctx, tx, env.ProjectID, sid); err != nil {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	c, err := readServiceConfig(ctx, tx, eid, sid, true)
	if err != nil || c.Token != req.Token || !c.Bound {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "配置已有更新或服务尚未加入此环境，请重新加载后操作")
		return
	}
	for _, secret := range req.Assets.Secrets {
		if secret.CredentialID == 0 {
			continue
		}
		var projectID int64
		var kind, data string
		if tx.QueryRowContext(ctx, `SELECT project_id,type,data FROM credential WHERE id=?`, secret.CredentialID).Scan(&projectID, &kind, &data) != nil || kind != "secret" || (projectID != 0 && projectID != env.ProjectID) {
			response.ErrorWithMsg(w, errcode.CodeParamInvalid, "凭据不可用或不属于当前项目")
			return
		}
		allowed, e := h.logic.authorizer.Authorize(ctx, uid, authz.Request{Resource: "credential", Action: "view", ProjectID: projectID, EnvironmentID: eid, Environment: env.Code})
		if e != nil || !allowed {
			response.Error(w, errcode.CodeNoPermission)
			return
		}
		if repo.decryptor == nil {
			response.Error(w, errcode.CodeInternalError)
			return
		}
		plain, e := repo.decryptor.Decrypt(data)
		var fields map[string]interface{}
		if e != nil || json.Unmarshal([]byte(plain), &fields) != nil {
			response.ErrorWithMsg(w, errcode.CodeParamInvalid, "凭据无法读取，请重新创建加密凭据")
			return
		}
		if _, ok := fields[secret.CredentialKey].(string); !ok {
			response.ErrorWithMsg(w, errcode.CodeParamInvalid, fmt.Sprintf("凭据 #%d 缺少所选文本字段", secret.CredentialID))
			return
		}
	}
	// Existing created PVCs cannot be silently replaced/reconfigured. Detaching
	// retains data; a new named entry creates a separate claim.
	for _, old := range c.Assets.Volumes {
		if old.ExistingClaim != "" {
			continue
		}
		for _, next := range req.Assets.Volumes {
			if next.Name == old.Name && (next.Capacity != old.Capacity || next.StorageClass != old.StorageClass || next.AccessMode != old.AccessMode || next.ExistingClaim != old.ExistingClaim) {
				response.ErrorWithMsg(w, errcode.CodeParamInvalid, "已有新建PVC的容量、StorageClass和访问模式不可在此改写；请另建命名存储或使用集群扩容流程")
				return
			}
		}
	}
	raw, _ := json.Marshal(req.Assets)
	_, err = tx.ExecContext(ctx, `INSERT INTO environment_resource(environment_id,name,kind,manifest,revision,created_by) VALUES(?,?,?,?,1,?) ON DUPLICATE KEY UPDATE manifest=VALUES(manifest),revision=revision+1`, eid, serviceassets.RowName(sid), serviceassets.ResourceKind, string(raw), uid)
	if err != nil {
		response.Error(w, errcode.CodeDBError)
		return
	}
	target, err := workflow.NewDefaultYAMLResolver(workflow.NewSQLYAMLSourceStoreTx(tx)).RenderYAMLConfig(ctx, workflow.WorkflowRun{ProjectID: env.ProjectID, EnvironmentID: eid}, workflow.YAMLDeployConfig{ServiceID: sid, ServiceRevisionID: c.RevisionID})
	if err == nil {
		_, err = workflow.ValidateYAMLManifest(target.Manifest, env.Namespace)
	}
	if err != nil {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "配置校验失败："+err.Error())
		return
	}
	after, err := readServiceConfig(ctx, tx, eid, sid, true)
	if err != nil {
		response.Error(w, errcode.CodeDBError)
		return
	}
	preview := r.URL.Query().Get("preview") == "1"
	if !preview {
		if tx.Commit() != nil {
			response.Error(w, errcode.CodeDBError)
			return
		}
	}
	response.Success(w, map[string]interface{}{"saved": !preview, "validated": true, "config": after, "manifest": string(target.Manifest), "secret_count": len(target.AssetSecrets)})
}
