package build

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

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

// ListRuns 构建列表（支持 service_id/project_id/status 过滤）
func (h *Handler) ListRuns(w http.ResponseWriter, r *http.Request) {
	serviceID := getInt64Param(r, "service_id", 0)
	projectID := getInt64Param(r, "project_id", 0)
	status := r.URL.Query().Get("status")
	page := getIntParam(r, "page", 1)
	pageSize := getIntParam(r, "page_size", 20)

	runs, total, code := h.logic.ListRuns(r.Context(), middleware.GetUserID(r.Context()), serviceID, projectID, status, page, pageSize)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.SuccessWithPage(w, runs, total, page, pageSize)
}

// GetDetail 构建详情
func (h *Handler) GetDetail(w http.ResponseWriter, r *http.Request) {
	id := getPathID(r)
	if id == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的构建 ID")
		return
	}
	detail, code := h.logic.GetRunDetail(r.Context(), middleware.GetUserID(r.Context()), id)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, detail)
}

// Trigger 触发构建
func (h *Handler) Trigger(w http.ResponseWriter, r *http.Request) {
	var req TriggerBuildRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "请求参数格式错误")
		return
	}
	if req.ServiceID == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "service_id 不能为空")
		return
	}

	userID := middleware.GetUserID(r.Context())
	id, code := h.logic.Trigger(r.Context(), &req, userID)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, map[string]int64{"id": id})
}

// Retry 重新构建
func (h *Handler) Retry(w http.ResponseWriter, r *http.Request) {
	id := getPathID(r)
	if id == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的构建 ID")
		return
	}
	userID := middleware.GetUserID(r.Context())
	newID, code := h.logic.Retry(r.Context(), id, userID)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, map[string]int64{"id": newID})
}

// Cancel 取消构建
func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	id := getPathID(r)
	if id == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的构建 ID")
		return
	}
	code := h.logic.Cancel(r.Context(), middleware.GetUserID(r.Context()), id)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, nil)
}

// GetBuildConfig 获取服务的构建配置
func (h *Handler) GetBuildConfig(w http.ResponseWriter, r *http.Request) {
	serviceID := getPathID(r)
	if serviceID == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的服务 ID")
		return
	}
	config, code := h.logic.GetBuildConfig(r.Context(), middleware.GetUserID(r.Context()), serviceID)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, config)
}

// SaveBuildConfig 保存构建配置
func (h *Handler) SaveBuildConfig(w http.ResponseWriter, r *http.Request) {
	var req CreateBuildConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "请求参数格式错误")
		return
	}
	pathID := getPathID(r)
	if pathID > 0 {
		req.ServiceID = pathID
	}
	if req.ServiceID == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "service_id 不能为空")
		return
	}

	userID := middleware.GetUserID(r.Context())
	code := h.logic.SaveBuildConfig(r.Context(), &req, userID)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, nil)
}

// ListTemplates 构建模板列表
func (h *Handler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	templates, code := h.logic.ListTemplates(r.Context(), middleware.GetUserID(r.Context()))
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, templates)
}

func (h *Handler) GetConcurrencyPolicy(w http.ResponseWriter, r *http.Request) {
	projectID := getInt64Param(r, "project_id", 0)
	policy, code := h.logic.GetConcurrencyPolicy(r.Context(), middleware.GetUserID(r.Context()), projectID)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, policy)
}

func (h *Handler) SaveConcurrencyPolicy(w http.ResponseWriter, r *http.Request) {
	var request SaveConcurrencyPolicyRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&request) != nil {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	policy, code := h.logic.SaveConcurrencyPolicy(r.Context(), middleware.GetUserID(r.Context()), request)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, policy)
}

func (h *Handler) GetPlatformBuildSettings(w http.ResponseWriter, r *http.Request) {
	settings, code := h.logic.GetPlatformBuildSettings(r.Context(), middleware.GetUserID(r.Context()))
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, settings)
}

func (h *Handler) SavePlatformBuildSettings(w http.ResponseWriter, r *http.Request) {
	var request SavePlatformBuildSettingsRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "平台 CI 配置格式错误")
		return
	}
	settings, code := h.logic.SavePlatformBuildSettings(r.Context(), middleware.GetUserID(r.Context()), request)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, settings)
}

func (h *Handler) Metrics(w http.ResponseWriter, r *http.Request) {
	projectID := getInt64Param(r, "project_id", 0)
	serviceID := getInt64Param(r, "service_id", 0)
	days := getIntParam(r, "days", 30)
	metrics, code := h.logic.Metrics(r.Context(), middleware.GetUserID(r.Context()), projectID, serviceID, days)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, metrics)
}

// GetLogs 获取构建日志
func (h *Handler) GetLogs(w http.ResponseWriter, r *http.Request) {
	id := getPathID(r)
	if id == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的构建 ID")
		return
	}
	logs, code := h.logic.GetLogs(r.Context(), middleware.GetUserID(r.Context()), id)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, logs)
}

// StreamLogs exposes the durable log snapshots over SSE. A Last-Event-ID
// cursor lets a reconnecting browser skip snapshots it already rendered.
func (h *Handler) StreamLogs(w http.ResponseWriter, r *http.Request) {
	id := getPathID(r)
	if id == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "invalid build ID")
		return
	}
	userID := middleware.GetUserID(r.Context())
	flusher, ok := w.(http.Flusher)
	if !ok {
		response.ErrorWithMsg(w, errcode.CodeInternalError, "streaming is not supported")
		return
	}
	if _, code := h.logic.GetLogs(r.Context(), userID, id); code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	lastEvent := r.Header.Get("Last-Event-ID")
	seen := map[int64]string{}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		logs, code := h.logic.GetLogs(r.Context(), userID, id)
		if code != errcode.CodeSuccess {
			return
		}
		for _, item := range logs {
			payload, _ := json.Marshal(item)
			eventID := logSnapshotID(item.ID, payload)
			if seen[item.ID] == eventID {
				continue
			}
			seen[item.ID] = eventID
			if eventID == lastEvent {
				continue
			}
			fmt.Fprintf(w, "id: %s\nevent: log\ndata: %s\n\n", eventID, payload)
		}
		fmt.Fprint(w, ": heartbeat\n\n")
		detail, _ := h.logic.GetRunDetail(r.Context(), userID, id)
		if detail != nil && isTerminalStatus(detail.Status) {
			fmt.Fprintf(w, "event: complete\ndata: {\"status\":%q}\n\n", detail.Status)
			flusher.Flush()
			return
		}
		flusher.Flush()
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
		}
	}
}

func logSnapshotID(id int64, payload []byte) string {
	// Bounded log tails and scheduler messages can change without gaining lines.
	return fmt.Sprintf("%d:%x", id, sha256.Sum256(payload))
}

func isTerminalStatus(status string) bool {
	switch status {
	case StatusSuccess, StatusFailed, StatusTimeout, StatusCancelled:
		return true
	}
	return false
}

func getPathID(r *http.Request) int64 {
	idStr := r.PathValue("id")
	if idStr == "" {
		return 0
	}
	id, _ := strconv.ParseInt(idStr, 10, 64)
	return id
}

func getIntParam(r *http.Request, key string, def int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return def
	}
	return v
}

func getInt64Param(r *http.Request, key string, def int64) int64 {
	s := r.URL.Query().Get(key)
	if s == "" {
		return def
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return def
	}
	return v
}
