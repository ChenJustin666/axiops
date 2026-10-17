package auth

import (
	"encoding/json"
	"net/http"

	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"github.com/ChenJustin666/axiops/internal/shared/middleware"
	"github.com/ChenJustin666/axiops/internal/shared/response"
)

// Handler 认证 HTTP 处理器
type Handler struct {
	logic *Logic
}

func NewHandler(logic *Logic) *Handler {
	return &Handler{logic: logic}
}

// Login 登录
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "请求参数格式错误")
		return
	}

	if req.Username == "" || req.Password == "" {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "用户名和密码不能为空")
		return
	}

	clientIP := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		clientIP = forwarded
	}

	resp, code := h.logic.Login(r.Context(), &req, clientIP)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}

	response.Success(w, resp)
}

// RefreshToken 刷新 Token
func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "请求参数格式错误")
		return
	}

	if req.RefreshToken == "" {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "refresh_token 不能为空")
		return
	}

	resp, code := h.logic.RefreshToken(r.Context(), &req)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}

	response.Success(w, resp)
}

// GetCurrentUser 获取当前用户信息
func (h *Handler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == 0 {
		response.Error(w, errcode.CodeTokenInvalid)
		return
	}

	info, code := h.logic.GetCurrentUser(r.Context(), userID)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}

	response.Success(w, info)
}

// Logout 登出（前端删除 Token 即可，后端可记录审计）
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	response.Success(w, nil)
}
