package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"github.com/ChenJustin666/axiops/internal/shared/middleware"
	"github.com/ChenJustin666/axiops/internal/shared/response"
)

// One ownership query per list, not one Kubernetes/SQL query per row.
func (l *Logic) RestartCapabilities(ctx context.Context, uid, eid int64) (map[string]bool, error) {
	r, ok := l.repo.(*Repository)
	if !ok {
		return nil, fmt.Errorf("restart repository unavailable")
	}
	env, err := r.FindEnvironmentTarget(ctx, eid)
	if err != nil {
		return nil, err
	}
	if env == nil {
		return map[string]bool{}, nil
	}
	owners, err := restartOwners(ctx, r.db, env)
	if err != nil {
		return nil, err
	}
	result := map[string]bool{}
	for key, owner := range owners {
		code := l.authorizeRestart(ctx, uid, env, owner.Guard)
		if code == errcode.CodeDBError {
			return nil, fmt.Errorf("restart authorization failed")
		}
		result[key] = code == 0
	}
	return result, nil
}

func (h *Handler) PreviewRestart(w http.ResponseWriter, r *http.Request) {
	p, code, message := h.logic.PreviewRestart(r.Context(), middleware.GetUserID(r.Context()), pathID(r, "id"), r.URL.Query().Get("kind"), r.URL.Query().Get("name"))
	if code != 0 {
		response.ErrorWithMsg(w, code, message)
		return
	}
	response.Success(w, p)
}
func (h *Handler) SubmitRestart(w http.ResponseWriter, r *http.Request) {
	var req RestartRequest
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&req) != nil {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	id, code, message := h.logic.SubmitRestart(r.Context(), middleware.GetUserID(r.Context()), pathID(r, "id"), req)
	if code != 0 {
		response.ErrorWithMsg(w, code, message)
		return
	}
	response.Success(w, map[string]int64{"id": id})
}
