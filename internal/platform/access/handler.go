package access

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

func (h *Handler) ListBindings(w http.ResponseWriter, r *http.Request) {
	subjectID, _ := strconv.ParseInt(r.URL.Query().Get("subject_id"), 10, 64)
	bindings, code := h.logic.ListBindings(r.Context(), middleware.GetUserID(r.Context()), r.URL.Query().Get("subject_type"), subjectID)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, bindings)
}

func (h *Handler) CreateBinding(w http.ResponseWriter, r *http.Request) {
	var req CreateBindingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	id, code := h.logic.CreateBinding(r.Context(), middleware.GetUserID(r.Context()), &req)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, map[string]int64{"id": id})
}

func (h *Handler) UpdateBinding(w http.ResponseWriter, r *http.Request) {
	var req UpdateBindingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	code := h.logic.UpdateBinding(r.Context(), middleware.GetUserID(r.Context()), pathID(r), &req)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, nil)
}

func (h *Handler) DeleteBinding(w http.ResponseWriter, r *http.Request) {
	code := h.logic.DeleteBinding(r.Context(), middleware.GetUserID(r.Context()), pathID(r))
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, nil)
}

func (h *Handler) CurrentMenus(w http.ResponseWriter, r *http.Request) {
	menus, code := h.logic.CurrentMenus(r.Context(), middleware.GetUserID(r.Context()))
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, menus)
}

func (h *Handler) CurrentAccess(w http.ResponseWriter, r *http.Request) {
	policies, code := h.logic.CurrentAccess(r.Context(), middleware.GetUserID(r.Context()))
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, policies)
}

func (h *Handler) ListMenus(w http.ResponseWriter, r *http.Request) {
	menus, code := h.logic.ListMenus(r.Context(), middleware.GetUserID(r.Context()))
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, menus)
}

func pathID(r *http.Request) int64 {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	return id
}
