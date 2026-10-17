package role

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"github.com/ChenJustin666/axiops/internal/shared/middleware"
	"github.com/ChenJustin666/axiops/internal/shared/response"
)

// Handler 角色 HTTP 处理器
type Handler struct {
	logic *Logic
}

func NewHandler(logic *Logic) *Handler {
	return &Handler{logic: logic}
}

// List 角色列表
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	roles, code := h.logic.List(r.Context(), middleware.GetUserID(r.Context()))
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, roles)
}

// GetByID 角色详情
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := getPathID(r)
	if id == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的角色 ID")
		return
	}

	role, code := h.logic.GetByID(r.Context(), middleware.GetUserID(r.Context()), id)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, role)
}

// Create 创建角色
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "请求参数格式错误")
		return
	}

	if req.Name == "" || req.Code == "" {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "角色名称和编码不能为空")
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

// Update 更新角色
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := getPathID(r)
	if id == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的角色 ID")
		return
	}

	var req UpdateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "请求参数格式错误")
		return
	}

	code := h.logic.Update(r.Context(), middleware.GetUserID(r.Context()), id, &req)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, nil)
}

// Delete 删除角色
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := getPathID(r)
	if id == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的角色 ID")
		return
	}

	code := h.logic.Delete(r.Context(), middleware.GetUserID(r.Context()), id)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, nil)
}

func getPathID(r *http.Request) int64 {
	idStr := r.PathValue("id")
	if idStr == "" {
		return 0
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0
	}
	return id
}
