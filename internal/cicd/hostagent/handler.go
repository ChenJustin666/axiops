package hostagent

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"github.com/ChenJustin666/axiops/internal/shared/middleware"
	"github.com/ChenJustin666/axiops/internal/shared/response"
)

type agentContextKey struct{}

type Handler struct{ logic *Logic }

func NewHandler(logic *Logic) *Handler { return &Handler{logic: logic} }

func (h *Handler) AgentAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agent, err := h.logic.Authenticate(r.Context(), r.Header.Get("Authorization"))
		if err != nil {
			writeHostAgentError(w, err)
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), agentContextKey{}, agent)))
	}
}

func (h *Handler) CreateRegistrationToken(w http.ResponseWriter, r *http.Request) {
	projectID := parsePathID(r, "project_id")
	var req RegistrationTokenRequest
	if decodeJSON(w, r, &req, 1<<20) != nil {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	result, err := h.logic.CreateRegistrationToken(r.Context(), middleware.GetUserID(r.Context()), projectID, req)
	if err != nil {
		writeHostAgentError(w, err)
		return
	}
	response.Success(w, result)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	projectID, _ := strconv.ParseInt(r.URL.Query().Get("project_id"), 10, 64)
	items, err := h.logic.List(r.Context(), middleware.GetUserID(r.Context()), projectID)
	if err != nil {
		writeHostAgentError(w, err)
		return
	}
	response.Success(w, items)
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if decodeJSON(w, r, &req, 1<<20) != nil {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	result, err := h.logic.Register(r.Context(), req)
	if err != nil {
		writeHostAgentError(w, err)
		return
	}
	response.Success(w, result)
}

func (h *Handler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	var req HeartbeatRequest
	if decodeJSON(w, r, &req, 1<<20) != nil {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	if err := h.logic.Heartbeat(r.Context(), agentFromContext(r.Context()), req); err != nil {
		writeHostAgentError(w, err)
		return
	}
	response.Success(w, map[string]bool{"accepted": true})
}

func (h *Handler) Poll(w http.ResponseWriter, r *http.Request) {
	command, err := h.logic.Poll(r.Context(), agentFromContext(r.Context()))
	if err != nil {
		writeHostAgentError(w, err)
		return
	}
	response.Success(w, command)
}

func (h *Handler) Report(w http.ResponseWriter, r *http.Request) {
	commandID := parsePathID(r, "id")
	var req ReportRequest
	if decodeJSON(w, r, &req, 2<<20) != nil {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	if err := h.logic.Report(r.Context(), agentFromContext(r.Context()), commandID, req); err != nil {
		writeHostAgentError(w, err)
		return
	}
	response.Success(w, map[string]bool{"accepted": true})
}

func agentFromContext(ctx context.Context) *Agent {
	agent, _ := ctx.Value(agentContextKey{}).(*Agent)
	return agent
}

func parsePathID(r *http.Request, key string) int64 {
	id, _ := strconv.ParseInt(r.PathValue(key), 10, 64)
	return id
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target interface{}, limit int64) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, limit))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func writeHostAgentError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalid):
		response.Error(w, errcode.CodeParamInvalid)
	case errors.Is(err, ErrNoPermission):
		response.Error(w, errcode.CodeNoPermission)
	case errors.Is(err, ErrUnauthorized):
		response.Error(w, errcode.CodeTokenInvalid)
	default:
		response.Error(w, errcode.CodeInternalError)
	}
}
