package envworkload

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"github.com/ChenJustin666/axiops/internal/shared/middleware"
	"github.com/ChenJustin666/axiops/internal/shared/response"
)

type Handler struct {
	logic *Logic
}

func NewHandler(logic *Logic) *Handler { return &Handler{logic: logic} }

func envID(r *http.Request) int64 {
	raw := r.PathValue("id")
	id, _ := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	return id
}

// Workloads GET /environments/:id/workloads?type=deployment&search=
func (h *Handler) Workloads(w http.ResponseWriter, r *http.Request) {
	id := envID(r)
	if id == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的环境 ID")
		return
	}
	kind := r.URL.Query().Get("type")
	search := r.URL.Query().Get("search")
	items, code := h.logic.ListWorkloads(r.Context(), middleware.GetUserID(r.Context()), id, kind, search)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, items)
}

// Pods GET /environments/:id/pods?search=
func (h *Handler) Pods(w http.ResponseWriter, r *http.Request) {
	id := envID(r)
	if id == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的环境 ID")
		return
	}
	items, code := h.logic.ListPods(r.Context(), middleware.GetUserID(r.Context()), id, r.URL.Query().Get("search"))
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, items)
}

// Services GET /environments/:id/services
func (h *Handler) Services(w http.ResponseWriter, r *http.Request) {
	id := envID(r)
	if id == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的环境 ID")
		return
	}
	items, code := h.logic.ListServices(r.Context(), middleware.GetUserID(r.Context()), id)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, items)
}

// Ingresses GET /environments/:id/ingresses
func (h *Handler) Ingresses(w http.ResponseWriter, r *http.Request) {
	id := envID(r)
	if id == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的环境 ID")
		return
	}
	items, code := h.logic.ListIngresses(r.Context(), middleware.GetUserID(r.Context()), id)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, items)
}

// PVCs GET /environments/:id/pvcs
func (h *Handler) PVCs(w http.ResponseWriter, r *http.Request) {
	id := envID(r)
	if id == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的环境 ID")
		return
	}
	items, code := h.logic.ListPVCs(r.Context(), middleware.GetUserID(r.Context()), id)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, items)
}

// Configs GET /environments/:id/configs?type=configmap|secret
func (h *Handler) Configs(w http.ResponseWriter, r *http.Request) {
	id := envID(r)
	if id == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的环境 ID")
		return
	}
	kind := r.URL.Query().Get("type")
	if kind == "" {
		kind = "configmap"
	}
	items, code := h.logic.ListConfigs(r.Context(), middleware.GetUserID(r.Context()), id, kind)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, items)
}
