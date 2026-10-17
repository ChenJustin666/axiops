package hostimport

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

func (h *Handler) Preview(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	var request Request
	if id == 0 || json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request) != nil {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	item, code := h.logic.Preview(r.Context(), middleware.GetUserID(r.Context()), id, request)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	item.Definition = ""
	response.Success(w, item)
}

func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	var request Request
	if id == 0 || json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request) != nil {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	item, code := h.logic.Import(r.Context(), middleware.GetUserID(r.Context()), id, request)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, item)
}

func pathID(r *http.Request) int64 {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if id < 1 {
		return 0
	}
	return id
}
