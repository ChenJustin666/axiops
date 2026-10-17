package project

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"github.com/ChenJustin666/axiops/internal/shared/middleware"
	"github.com/ChenJustin666/axiops/internal/shared/response"
)

// Handler 项目 HTTP 处理器
type Handler struct {
	logic *Logic
}

func NewHandler(logic *Logic) *Handler {
	return &Handler{logic: logic}
}

// List 项目列表
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	keyword := r.URL.Query().Get("keyword")
	status := r.URL.Query().Get("status")
	page := getIntParam(r, "page", 1)
	pageSize := getIntParam(r, "page_size", 20)

	projects, total, code := h.logic.List(r.Context(), middleware.GetUserID(r.Context()), keyword, status, page, pageSize)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.SuccessWithPage(w, projects, total, page, pageSize)
}

// MyProjects 我参与的项目
func (h *Handler) MyProjects(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	projects, code := h.logic.ListByUser(r.Context(), userID)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, projects)
}

// GetByID 项目详情
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := getPathID(r)
	if id == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的项目 ID")
		return
	}

	project, code := h.logic.GetByID(r.Context(), middleware.GetUserID(r.Context()), id)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, project)
}

// Create 创建项目
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "请求参数格式错误")
		return
	}
	if req.Name == "" || req.Code == "" || !IsValidDeliveryType(req.DeliveryType) {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "项目名称、编码和交付类型不能为空，交付类型仅支持 k8s_yaml、helm、host")
		return
	}

	createdBy := middleware.GetUserID(r.Context())
	id, code := h.logic.Create(r.Context(), &req, createdBy)
	if code != errcode.CodeSuccess {
		if code == errcode.CodeProjectNameExists {
			response.ErrorWithMsg(w, code, "项目编码已被现有项目使用，请更换编码；停用项目仍保留编码")
			return
		}
		response.Error(w, code)
		return
	}
	response.Success(w, map[string]int64{"id": id})
}

// GetDeliveryProfile returns the immutable delivery type and its migration state.
func (h *Handler) GetDeliveryProfile(w http.ResponseWriter, r *http.Request) {
	id := getPathID(r)
	if id == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的项目 ID")
		return
	}
	profile, code := h.logic.GetDeliveryProfile(r.Context(), middleware.GetUserID(r.Context()), id)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, profile)
}

// ConfirmDeliveryProfile confirms the type of a legacy project exactly once.
func (h *Handler) ConfirmDeliveryProfile(w http.ResponseWriter, r *http.Request) {
	id := getPathID(r)
	if id == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的项目 ID")
		return
	}
	var req ConfirmDeliveryProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "请求参数格式错误")
		return
	}
	code := h.logic.ConfirmDeliveryProfile(r.Context(), middleware.GetUserID(r.Context()), id, &req)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, nil)
}

// Update 更新项目
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := getPathID(r)
	if id == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的项目 ID")
		return
	}

	var req UpdateProjectRequest
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

// Delete 删除项目
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := getPathID(r)
	if id == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的项目 ID")
		return
	}

	var req DeleteProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ConfirmName == "" {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "请输入完整项目名称确认删除")
		return
	}
	check, code := h.logic.Delete(r.Context(), middleware.GetUserID(r.Context()), id, &req)
	if code != errcode.CodeSuccess {
		if code == errcode.CodeProjectInUse && check != nil {
			response.ErrorWithMsg(w, code, deletionBlockerMessage(check))
			return
		}
		if code == errcode.CodeParamInvalid {
			response.ErrorWithMsg(w, code, "项目名称不匹配，请重新打开删除确认")
			return
		}
		response.Error(w, code)
		return
	}
	response.Success(w, nil)
}

func (h *Handler) CheckDeletion(w http.ResponseWriter, r *http.Request) {
	check, code := h.logic.CheckDeletion(r.Context(), middleware.GetUserID(r.Context()), getPathID(r))
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, check)
}

// AddMember 添加项目成员
func (h *Handler) AddMember(w http.ResponseWriter, r *http.Request) {
	id := getPathID(r)
	if id == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的项目 ID")
		return
	}

	var req AddMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "请求参数格式错误")
		return
	}

	operatorID := middleware.GetUserID(r.Context())
	code := h.logic.AddMember(r.Context(), id, &req, operatorID)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, nil)
}

// RemoveMember 移除项目成员
func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	id := getPathID(r)
	if id == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的项目 ID")
		return
	}

	userIDStr := r.URL.Query().Get("user_id")
	userID, _ := strconv.ParseInt(userIDStr, 10, 64)
	if userID == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "user_id 不能为空")
		return
	}

	code := h.logic.RemoveMember(r.Context(), id, userID, middleware.GetUserID(r.Context()))
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, nil)
}

// --- 工具函数 ---

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

func getIntParam(r *http.Request, key string, defaultVal int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return defaultVal
	}
	return v
}
