package notification

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"github.com/ChenJustin666/axiops/internal/shared/middleware"
	"github.com/ChenJustin666/axiops/internal/shared/response"
)

type Handler struct{ logic *Logic }

func NewHandler(logic *Logic) *Handler { return &Handler{logic: logic} }
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	projectID := pathID(r, "project_id")
	var req CreateRequest
	if projectID == 0 || json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req) != nil {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	id, err := h.logic.Create(r.Context(), middleware.GetUserID(r.Context()), projectID, req)
	if err != nil {
		writeError(w, err)
		return
	}
	response.Success(w, map[string]int64{"id": id})
}
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	projectID, _ := strconv.ParseInt(r.URL.Query().Get("project_id"), 10, 64)
	items, err := h.logic.List(r.Context(), middleware.GetUserID(r.Context()), projectID)
	if err != nil {
		writeError(w, err)
		return
	}
	response.Success(w, items)
}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := pathID(r, "id")
	projectID, _ := strconv.ParseInt(r.URL.Query().Get("project_id"), 10, 64)
	if err := h.logic.Delete(r.Context(), middleware.GetUserID(r.Context()), id, projectID); err != nil {
		writeError(w, err)
		return
	}
	response.Success(w, map[string]bool{"deleted": true})
}
func (h *Handler) ListDeliveries(w http.ResponseWriter, r *http.Request) {
	projectID, _ := strconv.ParseInt(r.URL.Query().Get("project_id"), 10, 64)
	page := queryInt(r, "page", 1)
	pageSize := queryInt(r, "page_size", 20)
	if pageSize > 100 {
		pageSize = 100
	}
	items, total, err := h.logic.ListDeliveries(r.Context(), middleware.GetUserID(r.Context()), projectID, r.URL.Query().Get("status"), page, pageSize)
	if err != nil {
		writeError(w, err)
		return
	}
	response.SuccessWithPage(w, items, total, page, pageSize)
}
func pathID(r *http.Request, name string) int64 {
	id, _ := strconv.ParseInt(r.PathValue(name), 10, 64)
	if id <= 0 {
		return 0
	}
	return id
}
func queryInt(r *http.Request, name string, fallback int) int {
	value, err := strconv.Atoi(r.URL.Query().Get(name))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
func writeError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrNoPermission) {
		response.Error(w, errcode.CodeNoPermission)
		return
	}
	if errors.Is(err, ErrInvalid) {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	response.Error(w, errcode.CodeInternalError)
}
