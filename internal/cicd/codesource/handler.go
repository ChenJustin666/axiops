package codesource

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"github.com/ChenJustin666/axiops/internal/shared/middleware"
	"github.com/ChenJustin666/axiops/internal/shared/response"
	"github.com/ChenJustin666/axiops/pkg/gitprovider"
)

type Handler struct{ logic *Logic }

func NewHandler(logic *Logic) *Handler { return &Handler{logic: logic} }

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	projectID := pathID(r, "project_id")
	var request CreateRequest
	if projectID == 0 || json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request) != nil {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	id, err := h.logic.Create(r.Context(), middleware.GetUserID(r.Context()), projectID, request)
	if err != nil {
		log.Printf("code source create failed user=%d project=%d name=%q type=%q: %v", middleware.GetUserID(r.Context()), projectID, request.Name, request.Type, err)
		writeError(w, err)
		return
	}
	response.Success(w, map[string]int64{"id": id})
}

// CreateGlobal creates a reusable platform-level source. The authorization
// check is performed with project_id=0, so only platform-scoped grants can
// create connections shared by multiple projects.
func (h *Handler) CreateGlobal(w http.ResponseWriter, r *http.Request) {
	var request CreateRequest
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request) != nil {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	id, err := h.logic.Create(r.Context(), middleware.GetUserID(r.Context()), 0, request)
	if err != nil {
		log.Printf("global code source create failed user=%d name=%q type=%q: %v", middleware.GetUserID(r.Context()), request.Name, request.Type, err)
		writeError(w, err)
		return
	}
	response.Success(w, map[string]int64{"id": id})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	projectID := queryID(r, "project_id")
	sources, err := h.logic.List(r.Context(), middleware.GetUserID(r.Context()), projectID)
	if err != nil {
		writeError(w, err)
		return
	}
	response.Success(w, sources)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	source, err := h.logic.Get(r.Context(), middleware.GetUserID(r.Context()), pathID(r, "id"))
	if err != nil {
		writeError(w, err)
		return
	}
	response.Success(w, source)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := pathID(r, "id")
	var request UpdateRequest
	if id == 0 || json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request) != nil {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	if err := h.logic.Update(r.Context(), middleware.GetUserID(r.Context()), id, request); err != nil {
		log.Printf("code source update failed user=%d source=%d: %v", middleware.GetUserID(r.Context()), id, err)
		writeError(w, err)
		return
	}
	response.Success(w, map[string]bool{"updated": true})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.logic.Delete(r.Context(), middleware.GetUserID(r.Context()), pathID(r, "id")); err != nil {
		writeError(w, err)
		return
	}
	response.Success(w, map[string]bool{"deleted": true})
}

func (h *Handler) Check(w http.ResponseWriter, r *http.Request) {
	if err := h.logic.Check(r.Context(), middleware.GetUserID(r.Context()), pathID(r, "id")); err != nil {
		log.Printf("code source check failed user=%d source=%d: %v", middleware.GetUserID(r.Context()), pathID(r, "id"), err)
		writeProviderError(w, err)
		return
	}
	response.Success(w, map[string]bool{"configuration_valid": true})
}

func (h *Handler) ListRepositories(w http.ResponseWriter, r *http.Request) {
	items, err := h.logic.ListRepositories(r.Context(), middleware.GetUserID(r.Context()), pathID(r, "id"))
	if err != nil {
		log.Printf("code source repository listing failed user=%d source=%d: %v", middleware.GetUserID(r.Context()), pathID(r, "id"), err)
		writeProviderError(w, err)
		return
	}
	response.Success(w, items)
}

func (h *Handler) ListRepoFiles(w http.ResponseWriter, r *http.Request) {
	items, err := h.logic.ListRepoFiles(r.Context(), middleware.GetUserID(r.Context()), pathID(r, "id"),
		strings.TrimSpace(r.URL.Query().Get("repo_path")), strings.TrimSpace(r.URL.Query().Get("ref")), strings.TrimSpace(r.URL.Query().Get("path")))
	if err != nil {
		log.Printf("code source repository file listing failed user=%d source=%d: %v", middleware.GetUserID(r.Context()), pathID(r, "id"), err)
		writeProviderError(w, err)
		return
	}
	response.Success(w, items)
}

func writeProviderError(w http.ResponseWriter, err error) {
	var apiErr *gitprovider.APIError
	switch {
	case errors.Is(err, ErrNoPermission), errors.Is(err, ErrNotFound), errors.Is(err, ErrInvalid):
		writeError(w, err)
	case errors.Is(err, gitprovider.ErrUnauthorized):
		response.ErrorWithMsg(w, errcode.CodeCodeSourceCheckFail, "Git 服务拒绝访问，请检查 Token 是否有效，以及 Token 所属账号是否有该仓库的读取权限")
	case errors.Is(err, gitprovider.ErrNetworkTimeout):
		response.ErrorWithMsg(w, errcode.CodeCodeSourceCheckFail, "代码源网络不可达，请检查 AxiOps 后端到 Git 服务的网络")
	case errors.Is(err, gitprovider.ErrNotFound):
		response.ErrorWithMsg(w, errcode.CodeCodeSourceCheckFail, "仓库、分支或 API 地址不存在，或当前 Token 无权查看私有仓库；请核对路径及仓库授权")
	case errors.Is(err, gitprovider.ErrRateLimit):
		response.ErrorWithMsg(w, errcode.CodeCodeSourceCheckFail, "Git 服务请求频率受限，请稍后重试")
	case errors.Is(err, gitprovider.ErrInvalidResponse):
		response.ErrorWithMsg(w, errcode.CodeCodeSourceCheckFail, "Git 服务返回的列表或分页格式异常，仓库列表未能完整加载，请查看服务端日志")
	case errors.As(err, &apiErr):
		log.Printf("git provider response status=%d body=%q", apiErr.Status, apiErr.Body)
		response.ErrorWithMsg(w, errcode.CodeCodeSourceCheckFail, fmt.Sprintf("代码源 API 返回 HTTP %d，请检查服务地址和 Token", apiErr.Status))
	default:
		response.ErrorWithMsg(w, errcode.CodeCodeSourceCheckFail, "代码源连接失败，请查看 cicd-api.err.log")
	}
}

func (h *Handler) ListBranches(w http.ResponseWriter, r *http.Request) {
	items, err := h.logic.ListBranches(r.Context(), middleware.GetUserID(r.Context()), pathID(r, "id"), strings.TrimSpace(r.URL.Query().Get("repo_path")))
	if err != nil {
		log.Printf("code source branches failed source=%d repo=%q: %v", pathID(r, "id"), r.URL.Query().Get("repo_path"), err)
		writeProviderError(w, err)
		return
	}
	response.Success(w, items)
}

func (h *Handler) ListTags(w http.ResponseWriter, r *http.Request) {
	items, err := h.logic.ListTags(r.Context(), middleware.GetUserID(r.Context()), pathID(r, "id"), strings.TrimSpace(r.URL.Query().Get("repo_path")))
	if err != nil {
		writeError(w, err)
		return
	}
	response.Success(w, items)
}

func (h *Handler) GetCommit(w http.ResponseWriter, r *http.Request) {
	item, err := h.logic.GetCommit(r.Context(), middleware.GetUserID(r.Context()), pathID(r, "id"),
		strings.TrimSpace(r.URL.Query().Get("repo_path")), strings.TrimSpace(r.URL.Query().Get("ref")))
	if err != nil {
		writeError(w, err)
		return
	}
	response.Success(w, item)
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNoPermission):
		response.Error(w, errcode.CodeNoPermission)
	case errors.Is(err, ErrNotFound):
		response.Error(w, errcode.CodeCodeSourceNotFound)
	case errors.Is(err, ErrInvalid):
		response.Error(w, errcode.CodeParamInvalid)
	default:
		response.Error(w, errcode.CodeInternalError)
	}
}

func pathID(r *http.Request, name string) int64 {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id <= 0 {
		return 0
	}
	return id
}

func queryID(r *http.Request, name string) int64 {
	id, _ := strconv.ParseInt(r.URL.Query().Get(name), 10, 64)
	if id < 0 {
		return 0
	}
	return id
}
