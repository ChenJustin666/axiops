package department

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"github.com/ChenJustin666/axiops/internal/shared/middleware"
	"github.com/ChenJustin666/axiops/internal/shared/response"
)

type Handler struct{ logic *Logic }

func NewHandler(logic *Logic) *Handler { return &Handler{logic: logic} }

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	items, code := h.logic.List(r.Context(), middleware.GetUserID(r.Context()))
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, items)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if json.NewDecoder(r.Body).Decode(&req) != nil {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	id, code := h.logic.Create(r.Context(), middleware.GetUserID(r.Context()), &req)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, map[string]int64{"id": id})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req UpdateRequest
	if json.NewDecoder(r.Body).Decode(&req) != nil {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	code := h.logic.Update(r.Context(), middleware.GetUserID(r.Context()), pathID(r), &req)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, nil)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	code := h.logic.Delete(r.Context(), middleware.GetUserID(r.Context()), pathID(r))
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, nil)
}

func pathID(r *http.Request) int64 {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	return id
}
