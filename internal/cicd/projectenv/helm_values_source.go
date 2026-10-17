package projectenv

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/ChenJustin666/axiops/internal/cicd/helmconfig"
	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"github.com/ChenJustin666/axiops/internal/shared/middleware"
	"github.com/ChenJustin666/axiops/internal/shared/response"
)

// Native edits receive structural checks; required/schema values are checked
// with the chosen environment when saving its configuration.
func (h *Handler) ValidateHelmChart(w http.ResponseWriter, r *http.Request) {
	uid, sid := middleware.GetUserID(r.Context()), pathID(r, "service_id")
	svc, err := h.logic.repo.FindService(r.Context(), sid)
	if err != nil || svc == nil || svc.DeliveryType != DeliveryTypeHelm {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	allowed, err := h.logic.authorizer.Authorize(r.Context(), uid, authz.Request{Resource: "service", Action: "update", ProjectID: svc.ProjectID, ServiceID: sid})
	if err != nil || !allowed {
		response.Error(w, errcode.CodeNoPermission)
		return
	}
	type file struct {
		Path    string `json:"path"`
		Content string `json:"content"`
		SHA256  string `json:"sha256"`
	}
	var req struct {
		Files []file `json:"files"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&req) != nil || len(req.Files) == 0 || len(req.Files) > 500 {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	for i := range req.Files {
		sum := sha256.Sum256([]byte(req.Files[i].Content))
		req.Files[i].SHA256 = hex.EncodeToString(sum[:])
	}
	raw, _ := json.Marshal(struct {
		SchemaVersion int    `json:"schema_version"`
		Files         []file `json:"files"`
	}{1, req.Files})
	chart, err := helmconfig.LoadChart(raw)
	if err == nil {
		err = helmconfig.ValidateChartSyntax(chart)
	}
	if err != nil {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "Chart 结构校验失败，请检查路径、Chart.yaml、values.yaml 和依赖")
		return
	}
	response.Success(w, map[string]string{"definition": string(raw)})
}

// Values files have an independent Git source; Chart updates never rewrite them.
func (l *Logic) readValuesSource(ctx context.Context, uid, pid int64, v *helmconfig.ValuesSource) (*helmconfig.ValuesSource, error) {
	if l.codeSources == nil || v == nil || !helmconfig.ValidValuesFilePath(v.Path) {
		return nil, fmt.Errorf("请选择有效的 values YAML 文件")
	}
	source, err := l.codeSources.Get(ctx, uid, v.CodeSourceID)
	if err != nil || source == nil || source.ProjectID != 0 && source.ProjectID != pid {
		return nil, fmt.Errorf("无权读取此代码源")
	}
	snapshot, err := l.codeSources.ReadFiles(ctx, uid, v.CodeSourceID, v.RepoPath, v.Ref, []string{v.Path})
	if err != nil || snapshot == nil || len(snapshot.Files) != 1 {
		return nil, fmt.Errorf("读取 values 失败，请检查代码源权限、仓库、分支和文件路径")
	}
	raw := string(snapshot.Files[0].Content)
	if _, err = helmconfig.ParseValues(raw); err != nil {
		return nil, err
	}
	result := *v
	result.CommitID = snapshot.CommitID
	result.Base = raw
	return &result, nil
}

func (h *Handler) ReadHelmValues(w http.ResponseWriter, r *http.Request) {
	uid, pid := middleware.GetUserID(r.Context()), pathID(r, "project_id")
	var req struct {
		ServiceID int64                   `json:"service_id"`
		Source    helmconfig.ValuesSource `json:"source"`
	}
	if pid <= 0 || json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&req) != nil || req.ServiceID <= 0 {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	svc, err := h.logic.repo.FindService(r.Context(), req.ServiceID)
	if err != nil || svc == nil || svc.ProjectID != pid || svc.DeliveryType != DeliveryTypeHelm {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	allowed, err := h.logic.authorizer.Authorize(r.Context(), uid, authz.Request{Resource: "service", Action: "view", ProjectID: pid, ServiceID: req.ServiceID})
	if err != nil || !allowed {
		response.Error(w, errcode.CodeNoPermission)
		return
	}
	result, err := h.logic.readValuesSource(r.Context(), uid, pid, &req.Source)
	if err != nil {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, err.Error())
		return
	}
	response.Success(w, result)
}

// Client-supplied provenance is checked against authorized Git content. Existing
// snapshots may be edited locally without requiring the branch to stay frozen.
func (l *Logic) validateValuesSource(ctx context.Context, uid, pid, sid int64, req *SaveConfigRequest) error {
	v := req.Helm.Source
	if l.codeSources == nil {
		return fmt.Errorf("代码源服务未配置")
	}
	source, err := l.codeSources.Get(ctx, uid, v.CodeSourceID)
	if err != nil || source == nil || source.ProjectID != 0 && source.ProjectID != pid {
		return fmt.Errorf("无权读取 values 来源")
	}
	repo, ok := l.repo.(*Repository)
	if !ok {
		return fmt.Errorf("配置服务不可用")
	}
	unchanged := true
	for _, t := range req.Targets {
		old, e := helmconfig.Load(ctx, repo.db, t.EnvironmentID, sid)
		if e != nil || !reflect.DeepEqual(old.Source, v) {
			unchanged = false
			break
		}
	}
	if unchanged {
		return nil
	}
	actual, err := l.readValuesSource(ctx, uid, pid, v)
	if err != nil {
		return err
	}
	if actual.CommitID != v.CommitID || actual.Base != v.Base {
		return fmt.Errorf("values 来源已变化，请重新读取并预览")
	}
	return nil
}
