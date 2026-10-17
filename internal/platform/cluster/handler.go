package cluster

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"github.com/ChenJustin666/axiops/internal/shared/middleware"
	"github.com/ChenJustin666/axiops/internal/shared/response"
)

// Handler 集群 HTTP 处理器
type Handler struct {
	logic *Logic
}

func NewHandler(logic *Logic) *Handler {
	return &Handler{logic: logic}
}

// List 集群列表
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	projectID, _ := strconv.ParseInt(r.URL.Query().Get("project_id"), 10, 64)
	clusters, code := h.logic.List(r.Context(), middleware.GetUserID(r.Context()), projectID)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, clusters)
}

// GetByID 集群详情
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := getPathID(r)
	if id == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的集群 ID")
		return
	}

	cluster, code := h.logic.GetByID(r.Context(), middleware.GetUserID(r.Context()), id)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, cluster)
}

// Create 创建集群
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateClusterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "请求参数格式错误")
		return
	}
	if req.Name == "" || req.Code == "" || req.Kubeconfig == "" {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "集群名称、编码和 kubeconfig 不能为空")
		return
	}

	createdBy := middleware.GetUserID(r.Context())
	id, code := h.logic.Create(r.Context(), &req, createdBy)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, map[string]int64{"id": id})
}

// Update 更新集群
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := getPathID(r)
	if id == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的集群 ID")
		return
	}

	var req UpdateClusterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "请求参数格式错误")
		return
	}

	code := h.logic.Update(r.Context(), middleware.GetUserID(r.Context()), id, &req)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, nil)
}

// Delete 删除集群
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := getPathID(r)
	if id == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的集群 ID")
		return
	}

	code := h.logic.Delete(r.Context(), middleware.GetUserID(r.Context()), id)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, nil)
}

// TestConnect 测试集群连接
func (h *Handler) TestConnect(w http.ResponseWriter, r *http.Request) {
	id := getPathID(r)
	if id == 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的集群 ID")
		return
	}

	code := h.logic.TestConnect(r.Context(), middleware.GetUserID(r.Context()), id)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, map[string]string{"message": "连接测试已触发"})
}

func getPathID(r *http.Request) int64 {
	idStr := r.PathValue("id")
	if idStr == "" {
		return 0
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0
	}
	return id
}
