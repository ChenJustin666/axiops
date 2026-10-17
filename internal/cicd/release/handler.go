package release

import (
	"encoding/json"
	"net/http"
)

// Legacy releases never executed deployment and lacked environment-scoped
// authorization. Keep explicit tombstones for old clients; all real execution
// and history is handled by workflow runs and deployment snapshots.
type Handler struct{}

func NewHandler(_ *Logic) *Handler { return &Handler{} }
func retired(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusGone)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 410, "msg": "Legacy release API retired. Use environment-bound workflows and deployment snapshots.", "data": nil})
}
func (*Handler) ListByService(w http.ResponseWriter, r *http.Request) { retired(w, r) }
func (*Handler) GetByID(w http.ResponseWriter, r *http.Request)       { retired(w, r) }
func (*Handler) Create(w http.ResponseWriter, r *http.Request)        { retired(w, r) }
func (*Handler) Approve(w http.ResponseWriter, r *http.Request)       { retired(w, r) }
func (*Handler) Rollback(w http.ResponseWriter, r *http.Request)      { retired(w, r) }
