package user

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"github.com/ChenJustin666/axiops/internal/shared/middleware"
	"github.com/ChenJustin666/axiops/internal/shared/response"
)

// Handler 用户 HTTP 处理器
type Handler struct {
	logic *Logic
}

func NewHandler(logic *Logic) *Handler {
	return &Handler{logic: logic}
}

// List 用户列表
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	keyword := r.URL.Query().Get("keyword")
	status := -1
	if s := r.URL.Query().Get("status"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			status = v
		}
	}
	page := getIntParam(r, "page", 1)
	pageSize := getIntParam(r, "page_size", 20)

	req := &ListUsersRequest{
		Keyword:  keyword,
		Status:   status,
		Page:     page,
		PageSize: pageSize,
	}

	users, total, code := h.logic.List(r.Context(), middleware.GetUserID(r.Context()), req)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}

	response.SuccessWithPage(w, users, total, page, pageSize)
}

// GetByID 获取用户详情
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := getPathID(r)
	if id == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的用户 ID")
		return
	}

	user, code := h.logic.GetByID(r.Context(), middleware.GetUserID(r.Context()), id)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}

	response.Success(w, user)
}

// Create 创建用户
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "请求参数格式错误")
		return
	}

	if req.Username == "" || req.Password == "" {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "用户名和密码不能为空")
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

// Update 更新用户
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := getPathID(r)
	if id == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的用户 ID")
		return
	}

	var req UpdateUserRequest
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

// Delete 删除用户
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := getPathID(r)
	if id == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的用户 ID")
		return
	}

	code := h.logic.Delete(r.Context(), middleware.GetUserID(r.Context()), id)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}

	response.Success(w, nil)
}

// ResetPassword 重置密码（管理员）
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	id := getPathID(r)
	if id == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的用户 ID")
		return
	}

	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "请求参数格式错误")
		return
	}

	code := h.logic.ResetPassword(r.Context(), middleware.GetUserID(r.Context()), id, req.NewPassword)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}

	response.Success(w, nil)
}

// UpdatePassword 修改自己的密码
func (h *Handler) UpdatePassword(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var req UpdatePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "请求参数格式错误")
		return
	}

	code := h.logic.UpdatePassword(r.Context(), userID, &req)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}

	response.Success(w, nil)
}

// SetRoles 设置用户角色
func (h *Handler) SetRoles(w http.ResponseWriter, r *http.Request) {
	id := getPathID(r)
	if id == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的用户 ID")
		return
	}

	var req SetUserRolesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "请求参数格式错误")
		return
	}

	operatorID := middleware.GetUserID(r.Context())
	code := h.logic.SetRoles(r.Context(), id, req.RoleIDs, operatorID)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}

	response.Success(w, nil)
}

// --- 工具函数 ---

func getPathID(r *http.Request) int64 {
	// go-zero 路由参数通过 pathvar 获取
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

func getIntParam(r *http.Request, key string, defaultVal int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	if v <= 0 {
		return defaultVal
	}
	return v
}
