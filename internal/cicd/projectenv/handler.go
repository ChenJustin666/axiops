package projectenv

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

func NewHandler(logic *Logic) *Handler { return &Handler{logic: logic} }

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	projectID := pathID(r, "project_id")
	if projectID == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的项目 ID")
		return
	}
	environments, code := h.logic.List(r.Context(), middleware.GetUserID(r.Context()), projectID)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, environments)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := pathID(r, "id")
	if id == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的环境 ID")
		return
	}
	environment, code := h.logic.Get(r.Context(), middleware.GetUserID(r.Context()), id)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, environment)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	projectID := pathID(r, "project_id")
	if projectID == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的项目 ID")
		return
	}
	var req CreateEnvironmentRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&req); err != nil {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "请求参数格式错误")
		return
	}
	req.preview = r.URL.Query().Get("preview") == "1"
	id, code := h.logic.Create(r.Context(), middleware.GetUserID(r.Context()), projectID, &req)
	if code != errcode.CodeSuccess {
		if req.failure != nil {
			response.ErrorWithMsg(w, code, "环境配置校验失败，未创建环境；请检查 Chart/values、服务版本和必填项")
			return
		}
		response.Error(w, code)
		return
	}
	response.Success(w, map[string]interface{}{"id": id, "saved": !req.preview, "configs": req.configs})
}

func (h *Handler) Clone(w http.ResponseWriter, r *http.Request) {
	sourceID := pathID(r, "id")
	if sourceID == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的环境 ID")
		return
	}
	var req CloneEnvironmentRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "请求参数格式错误")
		return
	}
	id, code := h.logic.Clone(r.Context(), middleware.GetUserID(r.Context()), sourceID, &req)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, map[string]int64{"id": id})
}

func (h *Handler) GetVariableSet(w http.ResponseWriter, r *http.Request) {
	environmentID := pathID(r, "id")
	if environmentID == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的环境 ID")
		return
	}
	set, code := h.logic.GetVariableSet(r.Context(), middleware.GetUserID(r.Context()), environmentID)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, set)
}

func (h *Handler) GetServiceVariableSet(w http.ResponseWriter, r *http.Request) {
	environmentID, serviceID := pathID(r, "id"), pathID(r, "service_id")
	if environmentID == 0 || serviceID == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "invalid environment or service id")
		return
	}
	set, code := h.logic.GetServiceVariableSet(r.Context(), middleware.GetUserID(r.Context()), environmentID, serviceID)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, set)
}

func (h *Handler) UpdateVariableSet(w http.ResponseWriter, r *http.Request) {
	setID := pathID(r, "id")
	if setID == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的变量组 ID")
		return
	}
	var req UpdateVariableSetRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20)).Decode(&req); err != nil {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "请求参数格式错误")
		return
	}
	set, code := h.logic.UpdateVariableSet(r.Context(), middleware.GetUserID(r.Context()), setID, &req)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, set)
}

func (h *Handler) GetServiceOverride(w http.ResponseWriter, r *http.Request) {
	environmentID := pathID(r, "id")
	serviceID := pathID(r, "service_id")
	if environmentID == 0 || serviceID == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的环境或服务 ID")
		return
	}
	override, code := h.logic.GetServiceOverride(r.Context(), middleware.GetUserID(r.Context()), environmentID, serviceID)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, override)
}

func (h *Handler) SaveServiceOverride(w http.ResponseWriter, r *http.Request) {
	environmentID := pathID(r, "id")
	serviceID := pathID(r, "service_id")
	if environmentID == 0 || serviceID == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的环境或服务 ID")
		return
	}
	var req SaveServiceOverrideRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "请求参数格式错误")
		return
	}
	override, code := h.logic.SaveServiceOverride(r.Context(), middleware.GetUserID(r.Context()), environmentID, serviceID, &req)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, override)
}

func (h *Handler) ListResources(w http.ResponseWriter, r *http.Request) {
	id := pathID(r, "id")
	if id == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的环境 ID")
		return
	}
	items, code := h.logic.ListResources(r.Context(), middleware.GetUserID(r.Context()), id)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, items)
}

func (h *Handler) SaveResource(w http.ResponseWriter, r *http.Request) {
	id := pathID(r, "id")
	if id == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的环境 ID")
		return
	}
	var req SaveEnvironmentResourceRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20)).Decode(&req); err != nil {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "请求参数格式错误")
		return
	}
	item, code := h.logic.SaveResource(r.Context(), middleware.GetUserID(r.Context()), id, &req)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, item)
}

func (h *Handler) DeleteResource(w http.ResponseWriter, r *http.Request) {
	environmentID, resourceID := pathID(r, "id"), pathID(r, "resource_id")
	if environmentID == 0 || resourceID == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的环境或资源 ID")
		return
	}
	if code := h.logic.DeleteResource(r.Context(), middleware.GetUserID(r.Context()), environmentID, resourceID); code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, map[string]bool{"deleted": true})
}

func pathID(r *http.Request, name string) int64 {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id <= 0 {
		return 0
	}
	return id
}
