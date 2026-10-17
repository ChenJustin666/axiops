package appsvc

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"github.com/ChenJustin666/axiops/internal/shared/middleware"
	"github.com/ChenJustin666/axiops/internal/shared/response"
)

type Handler struct {
	logic *Logic
}

func NewHandler(logic *Logic) *Handler {
	return &Handler{logic: logic}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	projectID := getInt64Param(r, "project_id", 0)
	keyword := r.URL.Query().Get("keyword")
	status := r.URL.Query().Get("status")
	page := getIntParam(r, "page", 1)
	pageSize := getIntParam(r, "page_size", 20)

	services, total, code := h.logic.List(r.Context(), middleware.GetUserID(r.Context()), projectID, keyword, status, page, pageSize)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.SuccessWithPage(w, services, total, page, pageSize)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := getPathID(r)
	if id == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的服务 ID")
		return
	}
	svc, code := h.logic.GetByID(r.Context(), middleware.GetUserID(r.Context()), id)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, svc)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateAppServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "请求参数格式错误")
		return
	}
	if req.Name == "" || req.Code == "" || req.ProjectID == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "名称、编码和项目 ID 不能为空")
		return
	}
	if !validServiceName(req.Name) {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, serviceNameHint)
		return
	}

	createdBy := middleware.GetUserID(r.Context())
	id, code := h.logic.Create(r.Context(), &req, createdBy)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, map[string]int64{"id": id})
}

func (h *Handler) CreateForProject(w http.ResponseWriter, r *http.Request) {
	projectID := getNamedPathID(r, "project_id")
	if projectID == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的项目 ID")
		return
	}
	var req CreateAppServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "请求参数格式错误")
		return
	}
	req.ProjectID = projectID
	if req.Name == "" || req.Code == "" {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "名称和编码不能为空")
		return
	}
	if !validServiceName(req.Name) {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, serviceNameHint)
		return
	}
	id, code := h.logic.Create(r.Context(), &req, middleware.GetUserID(r.Context()))
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, map[string]int64{"id": id})
}

func (h *Handler) CreateRevision(w http.ResponseWriter, r *http.Request) {
	id := getPathID(r)
	if id == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的服务 ID")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8<<20)
	var req CreateServiceRevisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Definition == "" && req.Template == nil {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "definition 不能为空")
		return
	}
	revision, code := h.logic.CreateRevision(r.Context(), middleware.GetUserID(r.Context()), id, &req)
	if code != errcode.CodeSuccess {
		if code == errcode.CodeServiceDefinitionInvalid {
			if req.Template != nil {
				response.ErrorWithMsg(w, code, "模板版本不可用或 Chart 内容与固定版本不一致，请重新选择已保存的 Helm 模板；本次未保存")
				return
			}
			var chart struct {
				SchemaVersion int `json:"schema_version"`
			}
			if json.Unmarshal([]byte(req.Definition), &chart) == nil && chart.SchemaVersion == 1 {
				response.ErrorWithMsg(w, code, "Helm Chart 校验失败，请检查文件、依赖和默认参数；本次未保存")
				return
			}
			if err := validateYAMLDefinition(req.Definition); err != nil {
				response.ErrorWithMsg(w, code, err.Error()+"；本次未保存")
				return
			}
		}
		response.Error(w, code)
		return
	}
	response.Success(w, revision)
}

func (h *Handler) ListRevisions(w http.ResponseWriter, r *http.Request) {
	id := getPathID(r)
	if id == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的服务 ID")
		return
	}
	revisions, code := h.logic.ListRevisions(r.Context(), middleware.GetUserID(r.Context()), id, r.URL.Query().Get("latest") == "1")
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, revisions)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := getPathID(r)
	if id == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的服务 ID")
		return
	}
	var req UpdateAppServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "请求参数格式错误")
		return
	}
	code := h.logic.Update(r.Context(), middleware.GetUserID(r.Context()), id, &req)
	if code != errcode.CodeSuccess {
		if code == errcode.CodeParamInvalid && req.Name != "" && !validServiceName(req.Name) {
			response.ErrorWithMsg(w, code, serviceNameHint)
			return
		}
		response.Error(w, code)
		return
	}
	response.Success(w, nil)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := getPathID(r)
	if id == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的服务 ID")
		return
	}
	_, code := h.logic.DeleteChecked(r.Context(), middleware.GetUserID(r.Context()), id)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, nil)
}

func getPathID(r *http.Request) int64 {
	return getNamedPathID(r, "id")
}

func (h *Handler) CheckDeletion(w http.ResponseWriter, r *http.Request) {
	check, code := h.logic.CheckDeletion(r.Context(), middleware.GetUserID(r.Context()), getPathID(r))
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, check)
}
func (h *Handler) Restore(w http.ResponseWriter, r *http.Request) {
	code := h.logic.Restore(r.Context(), middleware.GetUserID(r.Context()), getPathID(r))
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, nil)
}
func (h *Handler) Purge(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ConfirmCode string `json:"confirm_code"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	code := h.logic.Purge(r.Context(), middleware.GetUserID(r.Context()), getPathID(r), req.ConfirmCode)
	if code != errcode.CodeSuccess {
		if code == errcode.CodeParamInvalid {
			response.ErrorWithMsg(w, code, "仅支持彻底删除垃圾桶中的服务，请输入完整服务编码确认")
		} else {
			response.Error(w, code)
		}
		return
	}
	response.Success(w, nil)
}
func (h *Handler) Rename(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	code := h.logic.Rename(r.Context(), middleware.GetUserID(r.Context()), getPathID(r), req.Name)
	if code != errcode.CodeSuccess {
		message := errcode.Message(code)
		if code == errcode.CodeParamInvalid {
			message = "名称须为 1–64 个字符且不能包含控制字符；已删除服务请先恢复"
		}
		response.ErrorWithMsg(w, code, message)
		return
	}
	response.Success(w, nil)
}
func (h *Handler) DetachEnvironment(w http.ResponseWriter, r *http.Request) {
	revision, _ := strconv.ParseInt(r.URL.Query().Get("revision"), 10, 64)
	code := h.logic.DetachEnvironment(r.Context(), middleware.GetUserID(r.Context()), getPathID(r), getNamedPathID(r, "environment_id"), revision)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, nil)
}

func getNamedPathID(r *http.Request, name string) int64 {
	idStr := r.PathValue(name)
	if idStr == "" {
		return 0
	}
	id, _ := strconv.ParseInt(idStr, 10, 64)
	return id
}

func getIntParam(r *http.Request, key string, def int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return def
	}
	return v
}

func getInt64Param(r *http.Request, key string, def int64) int64 {
	s := r.URL.Query().Get(key)
	if s == "" {
		return def
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return def
	}
	return v
}
