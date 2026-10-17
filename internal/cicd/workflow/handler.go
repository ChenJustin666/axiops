package workflow

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
	projectID := pathID(r, "project_id")
	if projectID == 0 {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	items, code := h.logic.List(r.Context(), middleware.GetUserID(r.Context()), projectID)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, items)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	projectID := pathID(r, "project_id")
	var req CreateWorkflowRequest
	if projectID == 0 || json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&req) != nil {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	id, code := h.logic.Create(r.Context(), middleware.GetUserID(r.Context()), projectID, &req)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, map[string]int64{"id": id})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := pathID(r, "id")
	if id == 0 {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	workflow, code := h.logic.Get(r.Context(), middleware.GetUserID(r.Context()), id)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, workflow)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := pathID(r, "id")
	var req UpdateWorkflowRequest
	if id == 0 || json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&req) != nil {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	workflow, code := h.logic.Update(r.Context(), middleware.GetUserID(r.Context()), id, &req)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, workflow)
}

func (h *Handler) Validate(w http.ResponseWriter, r *http.Request) {
	id := pathID(r, "id")
	var definition Definition
	if id == 0 || json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&definition) != nil {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	issues, code := h.logic.Validate(r.Context(), middleware.GetUserID(r.Context()), id, definition)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, map[string]interface{}{"valid": len(issues) == 0, "issues": issues})
}

func (h *Handler) Run(w http.ResponseWriter, r *http.Request) {
	id := pathID(r, "id")
	var req RunRequest
	if id == 0 || json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req) != nil {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	runID, code, message := h.logic.RunWithValidation(r.Context(), middleware.GetUserID(r.Context()), id, &req)
	if code != errcode.CodeSuccess {
		if message != "" {
			response.ErrorWithMsg(w, code, message)
			return
		}
		response.Error(w, code)
		return
	}
	response.Success(w, map[string]int64{"id": runID})
}

func (h *Handler) GetRun(w http.ResponseWriter, r *http.Request) {
	id := pathID(r, "id")
	if id == 0 {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	run, code := h.logic.GetRun(r.Context(), middleware.GetUserID(r.Context()), id)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	// Execution-only snapshots may contain environment configuration and are
	// not part of the workflow viewer's permissions or API contract.
	for i := range run.Tasks {
		run.Tasks[i].NodeSnapshot = publicNodeSnapshot(run.Tasks[i].NodeSnapshot)
	}
	response.Success(w, run)
}

func (h *Handler) ListRuns(w http.ResponseWriter, r *http.Request) {
	id := pathID(r, "id")
	if id == 0 {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	page := queryInt(r, "page", 1)
	pageSize := queryInt(r, "page_size", 20)
	items, total, code := h.logic.ListRuns(r.Context(), middleware.GetUserID(r.Context()), id, page, pageSize)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	for i := range items {
		for j := range items[i].Tasks {
			items[i].Tasks[j].NodeSnapshot = publicNodeSnapshot(items[i].Tasks[j].NodeSnapshot)
		}
	}
	response.SuccessWithPage(w, items, total, page, pageSize)
}

func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	id := pathID(r, "id")
	if id == 0 {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	code := h.logic.Cancel(r.Context(), middleware.GetUserID(r.Context()), id)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, map[string]bool{"accepted": true})
}

func (h *Handler) Approve(w http.ResponseWriter, r *http.Request) {
	id := pathID(r, "id")
	if id == 0 {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	code := h.logic.Approve(r.Context(), middleware.GetUserID(r.Context()), id)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, map[string]bool{"approved": true})
}

func (h *Handler) Rollback(w http.ResponseWriter, r *http.Request) {
	snapshotID := pathID(r, "id")
	var req RollbackRequest
	if snapshotID == 0 || json.NewDecoder(http.MaxBytesReader(w, r.Body, 16384)).Decode(&req) != nil {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	runID, code, message := h.logic.SubmitRollback(r.Context(), middleware.GetUserID(r.Context()), snapshotID, r.Header.Get("Idempotency-Key"), req)
	if code != errcode.CodeSuccess {
		if message != "" {
			response.ErrorWithMsg(w, code, message)
			return
		}
		response.Error(w, code)
		return
	}
	response.Success(w, map[string]int64{"workflow_run_id": runID})
}

func (h *Handler) RollbackOptions(w http.ResponseWriter, r *http.Request) {
	items, code := h.logic.RollbackOptions(r.Context(), middleware.GetUserID(r.Context()), pathID(r, "id"))
	if code != 0 {
		response.Error(w, code)
		return
	}
	response.Success(w, items)
}

func (h *Handler) PreviewRollback(w http.ResponseWriter, r *http.Request) {
	sid, _ := strconv.ParseInt(r.URL.Query().Get("service_id"), 10, 64)
	snapshotID, _ := strconv.ParseInt(r.URL.Query().Get("snapshot_id"), 10, 64)
	preview, code, message := h.logic.PreviewRollback(r.Context(), middleware.GetUserID(r.Context()), pathID(r, "id"), sid, snapshotID)
	if code != 0 {
		if message != "" {
			response.ErrorWithMsg(w, code, message)
		} else {
			response.Error(w, code)
		}
		return
	}
	response.Success(w, preview)
}

func pathID(r *http.Request, name string) int64 {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id <= 0 {
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
