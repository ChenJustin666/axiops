package build

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"github.com/ChenJustin666/axiops/internal/shared/middleware"
	"github.com/ChenJustin666/axiops/internal/shared/response"
)

type ImageRegistry struct {
	ID           int64    `json:"id"`
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	URL          string   `json:"url"`
	Namespaces   []string `json:"namespaces"`
	CredentialID int64    `json:"credential_id"`
	ProjectID    int64    `json:"project_id"`
	Status       string   `json:"status"`
}

// RegistryConnectivityRequest is the non-secret payload used by the explicit
// connection test endpoint.
type RegistryConnectivityRequest struct {
	URL          string   `json:"url"`
	Type         string   `json:"type"`
	Namespaces   []string `json:"namespaces"`
	CredentialID int64    `json:"credential_id"`
}

// ListImageRegistries lists the CI/CD registry catalog, separate from legacy
// console registry IDs. Global entries can be consumed by authorized projects.
func (h *Handler) ListImageRegistries(w http.ResponseWriter, r *http.Request) {
	projectID, err := strconv.ParseInt(r.URL.Query().Get("project_id"), 10, 64)
	if r.URL.Query().Get("project_id") == "" {
		projectID, err = 0, nil
	}
	if err != nil || projectID < 0 {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	resource, action := "build", "view"
	if projectID == 0 {
		resource, action = "build_governance", "view"
	}
	allowed, err := h.logic.authorizer.Authorize(r.Context(), middleware.GetUserID(r.Context()), authz.Request{Resource: resource, Action: action, ProjectID: projectID})
	if err != nil {
		response.Error(w, errcode.CodeDBError)
		return
	}
	if !allowed {
		response.Error(w, errcode.CodeNoPermission)
		return
	}
	rows, err := h.logic.repo.db.QueryContext(r.Context(), `SELECT id,name,type,url,COALESCE(namespaces,JSON_ARRAY()),credential_id,project_id,status FROM image_registry WHERE project_id IN (0,?) ORDER BY id DESC`, projectID)
	if err != nil {
		response.Error(w, errcode.CodeDBError)
		return
	}
	defer rows.Close()
	items := []ImageRegistry{}
	for rows.Next() {
		var item ImageRegistry
		var namespaces string
		if err := rows.Scan(&item.ID, &item.Name, &item.Type, &item.URL, &namespaces, &item.CredentialID, &item.ProjectID, &item.Status); err != nil {
			response.Error(w, errcode.CodeDBError)
			return
		}
		if err := json.Unmarshal([]byte(namespaces), &item.Namespaces); err != nil {
			response.Error(w, errcode.CodeDBError)
			return
		}
		items = append(items, item)
	}
	if rows.Err() != nil {
		response.Error(w, errcode.CodeDBError)
		return
	}
	response.Success(w, items)
}

// SaveImageRegistry manages global build resources using the same permission
// as platform CI settings. Credentials remain references to encrypted records.
func (h *Handler) SaveImageRegistry(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	allowed, err := h.logic.authorizer.Authorize(r.Context(), userID, authz.Request{Resource: "build_governance", Action: "update"})
	if err != nil {
		response.Error(w, errcode.CodeDBError)
		return
	}
	if !allowed {
		response.Error(w, errcode.CodeNoPermission)
		return
	}
	var item ImageRegistry
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&item) != nil || !validImageRegistry(&item) {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "请填写名称、有效仓库地址和镜像仓库凭据；命名空间仅支持小写字母、数字及合法的分隔符")
		return
	}
	allowed, err = h.logic.authorizer.Authorize(r.Context(), userID, authz.Request{Resource: "credential", Action: "view"})
	if err != nil {
		response.Error(w, errcode.CodeDBError)
		return
	}
	if !allowed {
		response.Error(w, errcode.CodeNoPermission)
		return
	}
	var count int
	if err := h.logic.repo.db.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM credential WHERE id=? AND project_id=0 AND type IN ('harbor_auth','registry_auth')`, item.CredentialID).Scan(&count); err != nil {
		response.Error(w, errcode.CodeDBError)
		return
	}
	if count != 1 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "请选择平台级镜像仓库凭据")
		return
	}
	if err := h.logic.TestRegistryConnection(r.Context(), item); err != nil {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, err.Error())
		return
	}
	namespaces, _ := json.Marshal(item.Namespaces)
	if item.ID == 0 {
		result, e := h.logic.repo.db.ExecContext(r.Context(), `INSERT INTO image_registry(name,type,url,namespaces,credential_id,project_id,status,created_by) VALUES(?,?,?,?,?,0,?,?)`, item.Name, item.Type, item.URL, string(namespaces), item.CredentialID, item.Status, userID)
		err = e
		if err == nil {
			item.ID, err = result.LastInsertId()
		}
	} else {
		if err = h.logic.repo.db.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM image_registry WHERE id=? AND project_id=0`, item.ID).Scan(&count); err == nil && count == 1 {
			_, err = h.logic.repo.db.ExecContext(r.Context(), `UPDATE image_registry SET name=?,type=?,url=?,namespaces=?,credential_id=?,status=? WHERE id=? AND project_id=0`, item.Name, item.Type, item.URL, string(namespaces), item.CredentialID, item.Status, item.ID)
		} else if err == nil {
			response.Error(w, errcode.CodeParamInvalid)
			return
		}
	}
	if err != nil {
		log.Printf("save build registry id=%d: %v", item.ID, err)
		response.Error(w, errcode.CodeDBError)
		return
	}
	response.Success(w, map[string]int64{"id": item.ID})
}

// TestImageRegistry verifies the remote registry before a catalog entry is
// saved. It is also called by the UI for an explicit "测试连接" action.
func (h *Handler) TestImageRegistry(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	allowed, err := h.logic.authorizer.Authorize(r.Context(), userID, authz.Request{Resource: "build_governance", Action: "update"})
	if err != nil {
		response.Error(w, errcode.CodeDBError)
		return
	}
	if !allowed {
		response.Error(w, errcode.CodeNoPermission)
		return
	}
	var req RegistryConnectivityRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&req) != nil {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "镜像仓库配置格式无效")
		return
	}
	item := ImageRegistry{Name: "connection-test", URL: req.URL, Type: req.Type, Namespaces: req.Namespaces, CredentialID: req.CredentialID, ProjectID: 0, Status: "active"}
	if !validImageRegistry(&item) {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "请填写有效的镜像仓库地址、类型和命名空间")
		return
	}
	if err := h.logic.TestRegistryConnection(r.Context(), item); err != nil {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, err.Error())
		return
	}
	response.Success(w, map[string]string{"status": "verified"})
}

func (l *Logic) TestRegistryConnection(ctx context.Context, item ImageRegistry) error {
	auth, err := l.registryAuthForCredential(ctx, item.CredentialID, item.URL, item.Namespaces)
	if err != nil {
		return err
	}
	checkCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	return testRegistryConnectivity(checkCtx, item.URL, auth)
}

var namespacePattern = regexp.MustCompile(`^[a-z0-9]+(?:(?:[._]|__|-+)[a-z0-9]+)*(?:/[a-z0-9]+(?:(?:[._]|__|-+)[a-z0-9]+)*)*$`)

func validImageRegistry(item *ImageRegistry) bool {
	item.Name = strings.TrimSpace(item.Name)
	item.URL = strings.TrimRight(strings.TrimSpace(item.URL), "/")
	if !strings.Contains(item.URL, "://") {
		item.URL = "https://" + item.URL
	}
	u, err := url.Parse(item.URL)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" || (u.Scheme != "https" && u.Scheme != "http") || strings.Contains(u.Path, "..") {
		return false
	}
	if len(item.Namespaces) > 100 {
		return false
	}
	cleaned := make([]string, 0, len(item.Namespaces))
	seen := map[string]bool{}
	for _, namespace := range item.Namespaces {
		namespace = strings.TrimSpace(namespace)
		if len(namespace) > 200 || !namespacePattern.MatchString(namespace) {
			return false
		}
		if !seen[namespace] {
			cleaned = append(cleaned, namespace)
			seen[namespace] = true
		}
	}
	item.Namespaces = cleaned
	if item.Status == "" {
		item.Status = "active"
	}
	switch item.Type {
	case "acr", "harbor", "dockerhub", "ecr", "registry":
	default:
		return false
	}
	return item.ID >= 0 && item.ProjectID == 0 && item.Name != "" && len(item.Name) <= 64 && len(item.URL) <= 256 && item.CredentialID > 0 && (item.Status == "active" || item.Status == "inactive")
}
