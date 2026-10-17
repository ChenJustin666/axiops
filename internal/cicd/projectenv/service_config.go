package projectenv

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/ChenJustin666/axiops/internal/cicd/serviceassets"
	"net/http"
	"sort"
	"strings"

	"github.com/ChenJustin666/axiops/internal/cicd/appsvc"
	"github.com/ChenJustin666/axiops/internal/cicd/helmconfig"
	"github.com/ChenJustin666/axiops/internal/cicd/templateversion"
	"github.com/ChenJustin666/axiops/internal/shared/authz"

	"github.com/ChenJustin666/axiops/internal/cicd/servicestate"
	"github.com/ChenJustin666/axiops/internal/cicd/workflow"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"github.com/ChenJustin666/axiops/internal/shared/middleware"
	"github.com/ChenJustin666/axiops/internal/shared/response"
)

// ServiceConfig is a read-only, consistent view. Reading never creates a binding
// or variable set. Token also covers inherited inputs, not only local overrides.
type ServiceConfig struct {
	Helm                      *helmconfig.Settings        `json:"helm,omitempty"`
	Adjustments               []string                    `json:"adjustments,omitempty"`
	EnvironmentID             int64                       `json:"environment_id"`
	ServiceID                 int64                       `json:"service_id"`
	RevisionID                int64                       `json:"revision_id"`
	LatestRevisionID          int64                       `json:"latest_revision_id"`
	Bound                     bool                        `json:"bound"`
	Definition                string                      `json:"definition"`
	Format                    string                      `json:"format"`
	ImageRef                  string                      `json:"image_ref"`
	Replicas                  *int                        `json:"replicas"`
	Variables                 []ConfigVariable            `json:"variables"`
	Token                     string                      `json:"token"`
	Assets                    serviceassets.Settings      `json:"assets"`
	EnvironmentYAMLRevisionID int64                       `json:"environment_yaml_revision_id"`
	Manifest                  string                      `json:"manifest,omitempty"`
	FinalValues               string                      `json:"final_values,omitempty"`
	ConfigDigest              string                      `json:"config_digest,omitempty"`
	CurrentDeployment         *workflow.CurrentDeployment `json:"current_deployment,omitempty"`
}
type ConfigVariable struct {
	CredentialID int64  `json:"credential_id,omitempty"`
	Key          string `json:"key"`
	Value        string `json:"value"`
	Secret       bool   `json:"secret"`
	Origin       string `json:"origin"`
}
type ConfigTarget struct {
	EnvironmentID int64  `json:"environment_id"`
	Token         string `json:"token"`
}
type SaveConfigRequest struct {
	HelmCredentials         map[string]*helmconfig.CredentialBinding `json:"helm_credentials,omitempty"`
	validateHelmCredentials func(context.Context, *sql.Tx, int64, map[string]*helmconfig.CredentialBinding) error
	Template                *templateversion.Ref `json:"template,omitempty"`
	Helm                    *helmconfig.Settings `json:"helm,omitempty"`
	Source                  *ConfigTarget        `json:"source,omitempty"`
	Targets                 []ConfigTarget       `json:"targets"`
	RevisionID              int64                `json:"revision_id"`
	// nil means restore inheritance. Unmentioned keys remain untouched.
	Variables       map[string]*string      `json:"variables"`
	Definition      *string                 `json:"definition,omitempty"`
	ResetDefinition bool                    `json:"reset_definition,omitempty"`
	Assets          *serviceassets.Settings `json:"assets,omitempty"`
	validateAssets  func(context.Context, *sql.Tx, int64, serviceassets.Settings, serviceassets.Settings) error
	Result          []*ServiceConfig `json:"-"`
}
type configQuery interface {
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

func readServiceConfig(ctx context.Context, q configQuery, environmentID, serviceID int64, lock bool) (*ServiceConfig, error) {
	c := &ServiceConfig{EnvironmentID: environmentID, ServiceID: serviceID, Variables: []ConfigVariable{}}
	suffix := ""
	if lock {
		suffix = " FOR UPDATE"
	}
	var ovID, ovRevision, bindingRevision int64
	var replicas sql.NullInt64
	var contextStamp string
	err := q.QueryRowContext(ctx, `SELECT svc.delivery_type,svc.current_revision_id, COALESCE(ov.id,0),COALESCE(ov.service_revision_id,0),
 COALESCE(ov.revision,0),COALESCE(ov.image_ref,''),ov.replicas,COALESCE(ov.environment_yaml_revision_id,0),
 CONCAT(env.project_id,':',env.cluster_id,':',env.namespace,':',env.code,':',env.status,':',env.binding_status,':',env.is_production,':',env.deploy_approval)
 FROM app_service svc JOIN project_environment env ON env.project_id=svc.project_id
 LEFT JOIN environment_service_override ov ON ov.service_id=svc.id AND ov.project_environment_id=env.id
 WHERE svc.id=? AND env.id=? AND svc.status='active' AND env.status='active'`+suffix, serviceID, environmentID).
		Scan(&c.Format, &c.LatestRevisionID, &ovID, &bindingRevision, &ovRevision, &c.ImageRef, &replicas, &c.EnvironmentYAMLRevisionID, &contextStamp)
	if err != nil {
		return nil, err
	}
	c.Bound = ovID > 0
	c.RevisionID = bindingRevision
	if c.RevisionID == 0 {
		c.RevisionID = c.LatestRevisionID
	}
	if replicas.Valid {
		v := int(replicas.Int64)
		c.Replicas = &v
	}
	if c.RevisionID > 0 {
		err = q.QueryRowContext(ctx, `SELECT definition,definition_format FROM service_revision WHERE id=? AND service_id=?`, c.RevisionID, serviceID).Scan(&c.Definition, &c.Format)
		if err != nil {
			return nil, err
		}
	}
	if c.EnvironmentYAMLRevisionID > 0 {
		if err = q.QueryRowContext(ctx, `SELECT definition FROM environment_service_yaml_revision WHERE id=? AND environment_id=? AND service_id=? AND base_revision_id=?`, c.EnvironmentYAMLRevisionID, environmentID, serviceID, c.RevisionID).Scan(&c.Definition); err != nil {
			return nil, err
		}
	}
	rows, err := q.QueryContext(ctx, `SELECT vs.id,vs.revision,vs.service_id,vs.scope,vs.name,COALESCE(vi.variable_key,''),
 CASE WHEN vi.is_secret=1 THEN '' ELSE COALESCE(vi.value_text,'') END,COALESCE(vi.is_secret,0),COALESCE(vi.credential_id,0)
 FROM variable_set vs LEFT JOIN variable_item vi ON vi.variable_set_id=vs.id
 WHERE vs.project_id=(SELECT project_id FROM app_service WHERE id=?) AND (vs.name='default' OR ?='helm_chart') AND
 ((vs.scope='environment' AND vs.environment_id=? AND vs.service_id IN(0,?)) OR
 (vs.scope='project' AND vs.environment_id=0 AND vs.service_id=0) OR
 (vs.scope='service' AND vs.service_id=? AND vs.environment_id IN(0,?)))
 ORDER BY FIELD(vs.scope,'project','service','environment'),vs.service_id,(vs.name='default'),vs.id,vi.variable_key`+suffix, serviceID, c.Format, environmentID, serviceID, serviceID, environmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	stamps := []string{contextStamp, fmt.Sprintf("%d:%d", ovID, ovRevision)}
	for rows.Next() {
		var id, revision, sid, credential int64
		var scope, name string
		var v ConfigVariable
		if err = rows.Scan(&id, &revision, &sid, &scope, &name, &v.Key, &v.Value, &v.Secret, &credential); err != nil {
			return nil, err
		}
		stamps = append(stamps, fmt.Sprintf("%d:%d:%d:%s:%d", id, revision, sid, scope, credential))
		if v.Secret {
			v.CredentialID = credential
		}
		if v.Key == "" {
			continue
		}
		if c.Format != "helm_chart" && scope != "environment" {
			continue
		}
		v.Origin = "global"
		if scope == "environment" && sid == serviceID && name == "default" {
			v.Origin = "override"
		}
		c.Variables = append(c.Variables, v)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	c.Assets, err = serviceassets.Load(ctx, q, environmentID, serviceID)
	if err != nil {
		return nil, err
	}
	if c.Format == "helm_chart" {
		s, e := helmconfig.Load(ctx, q, environmentID, serviceID)
		if e != nil {
			return nil, e
		}
		c.Helm = &s
	}
	payload, _ := json.Marshal(struct {
		Config *ServiceConfig
		Stamps []string
	}{c, stamps})
	sum := sha256.Sum256(payload)
	c.Token = hex.EncodeToString(sum[:])
	return c, nil
}

func validConfigRequest(req *SaveConfigRequest) bool {
	if req == nil || len(req.Targets) == 0 || len(req.Targets) > 50 || req.RevisionID < 0 || len(req.Variables) > 200 {
		return false
	}
	if req.Template != nil && (!req.Template.Valid() || req.Helm == nil || len(req.Targets) != 1 || req.Source != nil || req.RevisionID != 0 || len(req.Variables) > 0) {
		return false
	}
	if req.RevisionID == 0 && len(req.Variables) == 0 && req.Definition == nil && req.Assets == nil && !req.ResetDefinition && req.Helm == nil {
		return false
	}
	if req.Helm != nil && (req.Helm.Validate() != nil || req.Definition != nil || req.Assets != nil || req.ResetDefinition) {
		return false
	}
	if len(req.HelmCredentials) > 200 || len(req.HelmCredentials) > 0 && req.Helm == nil {
		return false
	}
	if len(req.HelmCredentials) > 0 && (len(req.Targets) != 1 || req.Source != nil) {
		return false
	}
	for key, ref := range req.HelmCredentials {
		if !helmconfig.PathPattern.MatchString(key) || len(key) > 128 || ref != nil && (ref.CredentialID <= 0 || !helmconfig.CredentialKeyPattern.MatchString(ref.Key)) {
			return false
		}
		if _, exists := req.Variables[key]; exists {
			return false
		}
		for _, path := range []string{req.Helm.Image.RefPath, req.Helm.Image.RepositoryPath, req.Helm.Image.TagPath} {
			if path != "" && helmconfig.PathsOverlap(path, key) {
				return false
			}
		}
	}
	if req.Definition != nil || req.Assets != nil || req.ResetDefinition {
		if len(req.Targets) != 1 || req.Source != nil && (req.Assets != nil || req.ResetDefinition) || req.Definition != nil && (req.ResetDefinition || strings.TrimSpace(*req.Definition) == "" || len(*req.Definition) > 1024*1024) {
			return false
		}
		if req.Assets != nil && serviceassets.Validate(*req.Assets) != nil {
			return false
		}
	}
	seen := map[int64]bool{}
	for _, t := range req.Targets {
		if t.EnvironmentID <= 0 || len(t.Token) != 64 || seen[t.EnvironmentID] {
			return false
		}
		seen[t.EnvironmentID] = true
	}
	if req.Source != nil && (req.Source.EnvironmentID <= 0 || len(req.Source.Token) != 64 || seen[req.Source.EnvironmentID]) {
		return false
	}
	for key, value := range req.Variables {
		if !variablePattern.MatchString(key) || strings.TrimSpace(key) != key {
			return false
		}
		switch key {
		case "Product", "Service", "Namespace", "EnvName", "ClusterName", "Image":
			return false
		}
		if strings.HasSuffix(key, "-image") || value != nil && len(*value) > 65535 {
			return false
		}
	}
	return true
}

func (h *Handler) GetServiceConfig(w http.ResponseWriter, r *http.Request) {
	uid, eid, sid := middleware.GetUserID(r.Context()), pathID(r, "id"), pathID(r, "service_id")
	_, _, code := h.logic.authorizeEnvironmentService(r.Context(), uid, eid, sid, "view")
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	repo, ok := h.logic.repo.(*Repository)
	if !ok {
		response.Error(w, errcode.CodeDBError)
		return
	}
	tx, err := repo.db.BeginTx(r.Context(), &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		response.Error(w, errcode.CodeDBError)
		return
	}
	defer tx.Rollback()
	c, err := readServiceConfig(r.Context(), tx, eid, sid, false)
	if err != nil {
		response.ErrorWithMsg(w, errcode.CodeDBError, "环境配置读取失败，请重新加载")
		return
	}
	if c.Format == "k8s_yaml" && c.Bound {
		source, e := workflow.NewSQLYAMLSourceStoreTx(tx).LoadYAMLSource(r.Context(), eid, sid, c.RevisionID)
		if e != nil {
			response.ErrorWithMsg(w, errcode.CodeDBError, "环境配置来源读取失败")
			return
		}
		c.ConfigDigest = workflow.DeploymentConfigDigest(source)
		var projectID int64
		if source != nil {
			projectID = source.ProjectID
		}
		c.CurrentDeployment, e = workflow.LoadCurrentDeployment(r.Context(), tx, projectID, eid, sid)
		if e != nil {
			response.ErrorWithMsg(w, errcode.CodeDBError, "当前部署记录读取失败")
			return
		}
	}
	response.Success(w, c)
}

func (h *Handler) SaveServiceConfig(w http.ResponseWriter, r *http.Request) {
	var req SaveConfigRequest
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20)).Decode(&req) != nil || !validConfigRequest(&req) {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	uid, sid := middleware.GetUserID(r.Context()), pathID(r, "service_id")
	var projectID int64
	for _, t := range req.Targets {
		env, svc, code := h.logic.authorizeEnvironmentService(r.Context(), uid, t.EnvironmentID, sid, "update")
		if code != errcode.CodeSuccess {
			response.Error(w, code)
			return
		}
		if svc.DeliveryType != DeliveryTypeK8sYAML && svc.DeliveryType != DeliveryTypeHelm {
			response.Error(w, errcode.CodeParamInvalid)
			return
		}
		if !IsEnvironmentCompatible(env.EnvironmentType, env.BindingStatus, svc.DeliveryType) {
			response.Error(w, errcode.CodeParamInvalid)
			return
		}
		projectID = env.ProjectID
	}
	if req.Source != nil {
		_, _, code := h.logic.authorizeEnvironmentService(r.Context(), uid, req.Source.EnvironmentID, sid, "view")
		if code != errcode.CodeSuccess {
			response.Error(w, code)
			return
		}
	}
	if req.Template != nil {
		allowed, err := h.logic.authorizer.Authorize(r.Context(), uid, authz.Request{Resource: "service", Action: "update", ProjectID: projectID, ServiceID: sid})
		if err != nil {
			response.Error(w, errcode.CodeDBError)
			return
		}
		if !allowed {
			response.Error(w, errcode.CodeNoPermission)
			return
		}
		scope, err := h.logic.authorizer.Scopes(r.Context(), uid, "build_template", "view")
		if err != nil {
			response.Error(w, errcode.CodeDBError)
			return
		}
		if !scope.HasAnyAllow() {
			response.Error(w, errcode.CodeNoPermission)
			return
		}
	}
	repo, ok := h.logic.repo.(*Repository)
	if !ok {
		response.Error(w, errcode.CodeDBError)
		return
	}
	preview := r.URL.Query().Get("preview") == "1"
	if req.Helm != nil && req.Helm.Source != nil {
		if err := h.logic.validateValuesSource(r.Context(), uid, projectID, sid, &req); err != nil {
			response.ErrorWithMsg(w, errcode.CodeParamInvalid, err.Error())
			return
		}
	}
	req.validateAssets = func(ctx context.Context, tx *sql.Tx, eid int64, old, next serviceassets.Settings) error {
		return h.validateDraftAssets(ctx, tx, repo, uid, projectID, eid, old, next)
	}
	req.validateHelmCredentials = func(ctx context.Context, tx *sql.Tx, eid int64, refs map[string]*helmconfig.CredentialBinding) error {
		var code string
		var cluster int64
		if err := tx.QueryRowContext(ctx, `SELECT code,cluster_id FROM project_environment WHERE id=? AND project_id=?`, eid, projectID).Scan(&code, &cluster); err != nil {
			return err
		}
		for _, ref := range refs {
			if ref == nil {
				continue
			}
			var kind string
			var owner int64
			if err := tx.QueryRowContext(ctx, `SELECT type,project_id FROM credential WHERE id=?`, ref.CredentialID).Scan(&kind, &owner); err != nil || kind != "secret" || owner != 0 && owner != projectID {
				return fmt.Errorf("请选择当前项目可用的安全凭据")
			}
			allowed, err := h.logic.authorizer.Authorize(ctx, uid, authz.Request{Resource: "credential", Action: "view", ProjectID: projectID, EnvironmentID: eid, Environment: code, ClusterID: cluster})
			if err != nil || !allowed {
				return fmt.Errorf("无权使用目标环境的凭据")
			}
		}
		return nil
	}
	if err := repo.saveServiceConfig(r.Context(), uid, projectID, sid, &req, preview); err != nil {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, err.Error())
		return
	}
	response.Success(w, map[string]interface{}{"saved": !preview, "validated": true, "configs": req.Result})
}

func (r *Repository) saveServiceConfig(ctx context.Context, uid, projectID, sid int64, req *SaveConfigRequest, preview bool) error {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = r.saveServiceConfigTx(ctx, tx, uid, projectID, sid, req); err != nil {
		return err
	}
	if preview {
		return nil
	}
	return tx.Commit()
}

func (r *Repository) saveServiceConfigTx(ctx context.Context, tx *sql.Tx, uid, projectID, sid int64, req *SaveConfigRequest) error {
	if !validConfigRequest(req) {
		return fmt.Errorf("环境配置请求无效")
	}
	var err error
	if err = servicestate.LockActive(ctx, tx, projectID, sid); err != nil {
		return err
	}
	if req.RevisionID > 0 {
		var id int64
		if err = tx.QueryRowContext(ctx, `SELECT id FROM service_revision WHERE id=? AND service_id=?`, req.RevisionID, sid).Scan(&id); err != nil {
			return fmt.Errorf("模板版本不属于当前服务")
		}
	}
	all := append([]ConfigTarget{}, req.Targets...)
	if req.Source != nil {
		all = append(all, *req.Source)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].EnvironmentID < all[j].EnvironmentID })
	configs := map[int64]*ServiceConfig{}
	previousSources := map[int64]*workflow.YAMLSource{}
	for _, t := range all {
		c, err := readServiceConfig(ctx, tx, t.EnvironmentID, sid, true)
		if err != nil {
			return fmt.Errorf("环境 #%d 配置读取失败", t.EnvironmentID)
		}
		if c.Token != t.Token {
			return fmt.Errorf("环境 #%d 已有更新，请重新加载并查看差异，本次未保存任何目标", t.EnvironmentID)
		}
		configs[t.EnvironmentID] = c
		if c.Format == "k8s_yaml" && c.Bound && len(c.Assets.Routes) > 0 {
			previousSources[t.EnvironmentID], err = workflow.NewSQLYAMLSourceStoreTx(tx).LoadYAMLSource(ctx, t.EnvironmentID, sid, c.RevisionID)
			if err != nil {
				return err
			}
		}
	}
	upgradeRevisionID := int64(0)
	if req.Template != nil {
		c := configs[req.Targets[0].EnvironmentID]
		if c.Format != "helm_chart" {
			return fmt.Errorf("当前服务不是 Helm Chart")
		}
		upgradeRevisionID, err = prepareHelmTemplateRevision(ctx, tx, uid, sid, c.LatestRevisionID, *req.Template)
		if err != nil {
			return err
		}
	}
	for _, t := range req.Targets {
		c := configs[t.EnvironmentID]
		if req.Helm != nil && c.Format != "helm_chart" {
			return fmt.Errorf("当前服务不是 Helm Chart，不能保存 Helm values")
		}
		effectiveRevisionID := req.RevisionID
		if upgradeRevisionID > 0 {
			effectiveRevisionID = upgradeRevisionID
		}
		if (req.Definition != nil || req.Assets != nil || req.ResetDefinition) && c.Format != "k8s_yaml" {
			return fmt.Errorf("当前服务不支持 Kubernetes YAML 配置")
		}
		if c.EnvironmentYAMLRevisionID > 0 && req.RevisionID > 0 && req.RevisionID != c.RevisionID && !req.ResetDefinition {
			return fmt.Errorf("环境 #%d 有独立 YAML，请先比较并恢复公共定义再升级模板", t.EnvironmentID)
		}
		if !c.Bound && effectiveRevisionID == 0 {
			if c.RevisionID == 0 {
				if len(req.Targets) != 1 || req.Definition == nil || c.Format != "k8s_yaml" {
					return fmt.Errorf("请先填写当前环境的完整 YAML")
				}
				if err = appsvc.ValidateYAMLDefinition(*req.Definition); err != nil {
					return err
				}
				id, e := createInitialRevision(ctx, tx, uid, sid, *req.Definition)
				if e != nil {
					return e
				}
				c.RevisionID = id
			}
			effectiveRevisionID = c.RevisionID
		}
		if req.Definition != nil {
			if err = appsvc.ValidateYAMLDefinition(*req.Definition); err != nil {
				return err
			}
		}
		for key := range req.Variables {
			for _, v := range c.Variables {
				if v.Key == key && v.Secret {
					return fmt.Errorf("变量 %s 为凭据引用，请在目标环境单独管理", key)
				}
			}
		}
		if effectiveRevisionID > 0 {
			_, err = tx.ExecContext(ctx, `INSERT INTO environment_service_override(project_environment_id,service_id,service_revision_id,image_ref,replicas,revision,updated_by)
   VALUES(?,?,?,'',NULL,1,?) ON DUPLICATE KEY UPDATE service_revision_id=VALUES(service_revision_id),revision=revision+1,updated_by=VALUES(updated_by)`, t.EnvironmentID, sid, effectiveRevisionID, uid)
			if err != nil {
				return err
			}
		}
		if req.ResetDefinition {
			if _, err = tx.ExecContext(ctx, `UPDATE environment_service_override SET environment_yaml_revision_id=0,revision=revision+1,updated_by=? WHERE project_environment_id=? AND service_id=?`, uid, t.EnvironmentID, sid); err != nil {
				return err
			}
		}
		if req.Definition != nil && *req.Definition != c.Definition {
			assets := c.Assets
			if req.Assets != nil {
				assets = *req.Assets
			}
			if e := serviceassets.ValidateYAMLEditOwnership(c.Definition, *req.Definition, assets); e != nil {
				return e
			}
			baseID := c.RevisionID
			if effectiveRevisionID > 0 {
				baseID = effectiveRevisionID
			}
			insert, e := tx.ExecContext(ctx, `INSERT INTO environment_service_yaml_revision(environment_id,service_id,base_revision_id,definition,created_by) VALUES(?,?,?,?,?)`, t.EnvironmentID, sid, baseID, *req.Definition, uid)
			if e != nil {
				return e
			}
			id, e := insert.LastInsertId()
			if e != nil {
				return e
			}
			if _, err = tx.ExecContext(ctx, `UPDATE environment_service_override SET environment_yaml_revision_id=?,revision=revision+1,updated_by=? WHERE project_environment_id=? AND service_id=?`, id, uid, t.EnvironmentID, sid); err != nil {
				return err
			}
		}
		if req.Assets != nil {
			if req.validateAssets == nil && len(req.Assets.Secrets) > 0 {
				return fmt.Errorf("凭据引用必须经过授权校验")
			}
			if req.validateAssets != nil {
				if err = req.validateAssets(ctx, tx, t.EnvironmentID, c.Assets, *req.Assets); err != nil {
					return err
				}
			}
			value, _ := json.Marshal(req.Assets)
			if _, err = tx.ExecContext(ctx, `INSERT INTO environment_resource(environment_id,name,kind,manifest,revision,created_by) VALUES(?,?,?,?,1,?) ON DUPLICATE KEY UPDATE manifest=VALUES(manifest),revision=revision+1`, t.EnvironmentID, serviceassets.RowName(sid), serviceassets.ResourceKind, string(value), uid); err != nil {
				return err
			}
		}
		if req.Helm != nil {
			// Keep target credential field mappings; only authorized explicit edits may change them.
			next := *req.Helm
			next.SharedValues, next.SharedRevision = "", 0
			next.CredentialKeys = map[string]string{}
			if c.Helm != nil {
				for path, key := range c.Helm.CredentialKeys {
					next.CredentialKeys[path] = key
				}
			}
			for path, ref := range req.HelmCredentials {
				if ref == nil {
					delete(next.CredentialKeys, path)
				} else {
					next.CredentialKeys[path] = ref.Key
				}
			}
			value, e := json.Marshal(next)
			if e != nil {
				return e
			}
			if _, err = tx.ExecContext(ctx, `INSERT INTO environment_resource(environment_id,name,kind,manifest,revision,created_by) VALUES(?,?,?,?,1,?) ON DUPLICATE KEY UPDATE manifest=VALUES(manifest),revision=revision+1`, t.EnvironmentID, helmconfig.RowName(sid), helmconfig.Kind, string(value), uid); err != nil {
				return err
			}
		}
		if len(req.HelmCredentials) > 0 {
			if c.Format != "helm_chart" || req.validateHelmCredentials == nil {
				return fmt.Errorf("Helm 凭据引用必须经过授权校验")
			}
			if err = req.validateHelmCredentials(ctx, tx, t.EnvironmentID, req.HelmCredentials); err != nil {
				return err
			}
			if _, err = tx.ExecContext(ctx, `INSERT INTO variable_set(project_id,environment_id,service_id,name,scope,revision,created_by) VALUES(?,?,?,'default','environment',1,?) ON DUPLICATE KEY UPDATE revision=revision+1`, projectID, t.EnvironmentID, sid, uid); err != nil {
				return err
			}
			var setID int64
			if err = tx.QueryRowContext(ctx, `SELECT id FROM variable_set WHERE project_id=? AND environment_id=? AND service_id=? AND scope='environment' AND name='default'`, projectID, t.EnvironmentID, sid).Scan(&setID); err != nil {
				return err
			}
			for key, ref := range req.HelmCredentials {
				if _, err = tx.ExecContext(ctx, `DELETE FROM variable_item WHERE variable_set_id=? AND variable_key=?`, setID, key); err != nil {
					return err
				}
				if ref != nil {
					if _, err = tx.ExecContext(ctx, `INSERT INTO variable_item(variable_set_id,variable_key,value_text,credential_id,is_secret,created_by) VALUES(?,?,'',?,1,?)`, setID, key, ref.CredentialID, uid); err != nil {
						return err
					}
				}
			}
		}
		if len(req.Variables) > 0 {
			_, err = tx.ExecContext(ctx, `INSERT INTO variable_set(project_id,environment_id,service_id,name,scope,revision,created_by)
   VALUES(?,?,?,'default','environment',1,?) ON DUPLICATE KEY UPDATE revision=revision+1`, projectID, t.EnvironmentID, sid, uid)
			if err != nil {
				return err
			}
			var setID int64
			if err = tx.QueryRowContext(ctx, `SELECT id FROM variable_set WHERE project_id=? AND environment_id=? AND service_id=? AND scope='environment' AND name='default'`, projectID, t.EnvironmentID, sid).Scan(&setID); err != nil {
				return err
			}
			for key, value := range req.Variables {
				if _, err = tx.ExecContext(ctx, `DELETE FROM variable_item WHERE variable_set_id=? AND variable_key=?`, setID, key); err != nil {
					return err
				}
				if value != nil {
					if _, err = tx.ExecContext(ctx, `INSERT INTO variable_item(variable_set_id,variable_key,value_text,credential_id,is_secret,created_by) VALUES(?,?,?,0,0,?)`, setID, key, *value, uid); err != nil {
						return err
					}
				}
				// Legacy replicas override has higher priority; an explicit edit transfers
				// ownership to the service variable, including restoring inheritance.
				if key == "replicas" {
					if _, err = tx.ExecContext(ctx, `UPDATE environment_service_override SET replicas=NULL,revision=revision+1,updated_by=? WHERE project_environment_id=? AND service_id=?`, uid, t.EnvironmentID, sid); err != nil {
						return err
					}
				}
			}
		}
	}
	for _, t := range req.Targets {
		c, err := readServiceConfig(ctx, tx, t.EnvironmentID, sid, true)
		if err != nil {
			return err
		}
		if c.Format == "k8s_yaml" {
			if before := previousSources[t.EnvironmentID]; before != nil {
				after, e := workflow.NewSQLYAMLSourceStoreTx(tx).LoadYAMLSource(ctx, t.EnvironmentID, sid, c.RevisionID)
				if e != nil {
					return e
				}
				settings, notices, e := workflow.ReconcileYAMLRoutePorts(before, after)
				if e != nil {
					return e
				}
				if len(notices) > 0 {
					value, e := json.Marshal(settings)
					if e != nil {
						return e
					}
					if _, e = tx.ExecContext(ctx, `UPDATE environment_resource SET manifest=?,revision=revision+1 WHERE environment_id=? AND name=? AND kind=?`, string(value), t.EnvironmentID, serviceassets.RowName(sid), serviceassets.ResourceKind); e != nil {
						return e
					}
					c, e = readServiceConfig(ctx, tx, t.EnvironmentID, sid, true)
					if e != nil {
						return e
					}
					c.Adjustments = notices
				}
			}
			target, renderErr := workflow.NewDefaultYAMLResolver(workflow.NewSQLYAMLSourceStoreTx(tx)).RenderYAMLConfig(ctx, workflow.WorkflowRun{ProjectID: projectID, EnvironmentID: t.EnvironmentID}, workflow.YAMLDeployConfig{ServiceID: sid, ServiceRevisionID: c.RevisionID})
			err = renderErr
			if err == nil {
				_, err = workflow.ValidateYAMLManifest(target.Manifest, target.Namespace)
			}
			if err != nil {
				return fmt.Errorf("环境 #%d 配置校验失败：%s；本次未保存任何目标", t.EnvironmentID, err)
			}
			c.Manifest = string(target.Manifest)
			source, e := workflow.NewSQLYAMLSourceStoreTx(tx).LoadYAMLSource(ctx, t.EnvironmentID, sid, c.RevisionID)
			if e != nil {
				return e
			}
			c.ConfigDigest = workflow.DeploymentConfigDigest(source)
		}
		if c.Format == "helm_chart" {
			var manifest []byte
			manifest, c.FinalValues, err = workflow.RenderHelmServicePreview(ctx, tx, r.decryptor, projectID, t.EnvironmentID, sid, c.RevisionID)
			if err != nil {
				return fmt.Errorf("环境 #%d Chart/values 校验失败，请检查变量路径、类型、必填值和凭据；本次未保存任何目标", t.EnvironmentID)
			}
			c.Manifest = string(manifest)
		}
		req.Result = append(req.Result, c)
	}
	return nil
}

func createInitialRevision(ctx context.Context, tx *sql.Tx, uid, sid int64, definition string) (int64, error) {
	sum := sha256.Sum256([]byte(definition))
	if _, err := tx.ExecContext(ctx, `INSERT INTO service_revision_sequence(service_id,next_revision) VALUES(?,1)`, sid); err != nil {
		return 0, err
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO service_revision(service_id,revision_number,source_commit,definition,definition_format,content_hash,created_by) VALUES(?,1,'environment-create',?,'k8s_yaml',?,?)`, sid, definition, hex.EncodeToString(sum[:]), uid)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	_, err = tx.ExecContext(ctx, `UPDATE app_service SET current_revision_id=? WHERE id=?`, id, sid)
	return id, err
}
