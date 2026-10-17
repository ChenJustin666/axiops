package credential

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"github.com/ChenJustin666/axiops/internal/shared/middleware"
	"github.com/ChenJustin666/axiops/internal/shared/response"
)

// Handler 凭据 HTTP 处理器
type Handler struct {
	logic *Logic
}

func NewHandler(logic *Logic) *Handler {
	return &Handler{logic: logic}
}

// List 凭据列表
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	credType := r.URL.Query().Get("type")
	projectID := int64(-1)
	if pid := r.URL.Query().Get("project_id"); pid != "" {
		if v, err := strconv.ParseInt(pid, 10, 64); err == nil {
			projectID = v
		}
	}

	creds, code := h.logic.List(r.Context(), middleware.GetUserID(r.Context()), credType, projectID)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, creds)
}

// GetByID 凭据详情
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := getPathID(r)
	if id == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的凭据 ID")
		return
	}

	cred, code := h.logic.GetByID(r.Context(), middleware.GetUserID(r.Context()), id)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, cred)
}

// Create 创建凭据
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateCredentialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "请求参数格式错误")
		return
	}
	if req.Name == "" || req.Type == "" || req.Data == "" {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "名称、类型和凭据数据不能为空")
		return
	}

	createdBy := middleware.GetUserID(r.Context())
	id, code := h.logic.Create(r.Context(), &req, createdBy)
	if code != errcode.CodeSuccess {
		log.Printf("credential create failed user=%d type=%q project=%d code=%d", createdBy, req.Type, req.ProjectID, code)
		response.Error(w, code)
		return
	}
	response.Success(w, map[string]int64{"id": id})
}

// Update 更新凭据
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := getPathID(r)
	if id == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的凭据 ID")
		return
	}

	var req UpdateCredentialRequest
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

// Delete 删除凭据
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := getPathID(r)
	if id == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的凭据 ID")
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

func (h *Handler) DeriveSecret(w http.ResponseWriter, r *http.Request) {
	var req DeriveSecretRequest
	id := getPathID(r)
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if id <= 0 || decoder.Decode(&req) != nil || decoder.Decode(new(interface{})) != io.EOF {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	nextID, code := h.logic.DeriveSecret(r.Context(), middleware.GetUserID(r.Context()), id, &req)
	if code != 0 {
		response.ErrorWithMsg(w, code, "凭据修改失败：请检查字段、权限，或重新加载已变更的凭据")
		return
	}
	response.Success(w, map[string]int64{"id": nextID})
}
