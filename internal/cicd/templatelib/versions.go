package templatelib

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/ChenJustin666/axiops/internal/cicd/helmconfig"
	"github.com/ChenJustin666/axiops/internal/cicd/templateversion"
	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"github.com/ChenJustin666/axiops/internal/shared/middleware"
	"github.com/ChenJustin666/axiops/internal/shared/response"
)

type versionSummary struct {
	Revision    int       `json:"revision"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	ContentHash string    `json:"content_hash"`
	CreatedAt   time.Time `json:"created_at"`
}

func (h *Handler) Versions(w http.ResponseWriter, r *http.Request) {
	if !h.allowed(w, r, "view") {
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if id <= 0 {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	rows, err := h.db.QueryContext(r.Context(), `SELECT revision,JSON_UNQUOTE(JSON_EXTRACT(snapshot,'$.name')),JSON_UNQUOTE(JSON_EXTRACT(snapshot,'$.description')),content_hash,created_at FROM delivery_template_version WHERE template_id=? ORDER BY revision DESC LIMIT 200`, id)
	if err != nil {
		response.Error(w, errcode.CodeDBError)
		return
	}
	defer rows.Close()
	items := []versionSummary{}
	for rows.Next() {
		var item versionSummary
		if rows.Scan(&item.Revision, &item.Name, &item.Description, &item.ContentHash, &item.CreatedAt) != nil {
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

func (h *Handler) Version(w http.ResponseWriter, r *http.Request) {
	if !h.allowed(w, r, "view") {
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	rev, _ := strconv.Atoi(r.PathValue("revision"))
	v, err := templateversion.Load(r.Context(), h.db, templateversion.Ref{ID: id, Revision: rev}, false)
	if err != nil {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, err.Error())
		return
	}
	var t Template
	if json.Unmarshal(v.Snapshot, &t) != nil {
		response.Error(w, errcode.CodeDBError)
		return
	}
	response.Success(w, map[string]interface{}{"template": t, "definition": v.Definition, "content_hash": v.ContentHash})
}

func (h *Handler) ServiceTemplate(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if !h.allowed(w, r, "view") {
		return
	}
	var pid, latest int64
	if id <= 0 || h.db.QueryRowContext(r.Context(), `SELECT project_id,current_revision_id FROM app_service WHERE id=? AND status='active' AND delivery_type='helm'`, id).Scan(&pid, &latest) != nil {
		response.Error(w, errcode.CodeServiceNotFound)
		return
	}
	allowed, err := h.auth.Authorize(r.Context(), middleware.GetUserID(r.Context()), authz.Request{Resource: "service", Action: "view", ProjectID: pid, ServiceID: id})
	if err != nil {
		response.Error(w, errcode.CodeDBError)
		return
	}
	if !allowed {
		response.Error(w, errcode.CodeNoPermission)
		return
	}
	rid, _ := strconv.ParseInt(r.URL.Query().Get("revision_id"), 10, 64)
	if rid == 0 {
		rid = latest
	}
	var source, definition string
	var number int64
	if h.db.QueryRowContext(r.Context(), `SELECT source_commit,definition,revision_number FROM service_revision WHERE id=? AND service_id=? AND definition_format='helm_chart'`, rid, id).Scan(&source, &definition, &number) != nil {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	ref, linked := templateversion.ParseSource(source)
	mode := "template"
	if !linked {
		mode = "custom"
		var prior string
		err = h.db.QueryRowContext(r.Context(), `SELECT source_commit FROM service_revision WHERE service_id=? AND revision_number<? AND source_commit LIKE 'template:%' AND definition_format='helm_chart' ORDER BY revision_number DESC LIMIT 1`, id, number).Scan(&prior)
		if err != nil && err != sql.ErrNoRows {
			response.Error(w, errcode.CodeDBError)
			return
		}
		ref, linked = templateversion.ParseSource(prior)
	}
	result := map[string]interface{}{"mode": "unlinked", "service_id": id, "revision_id": rid}
	if !linked {
		response.Success(w, result)
		return
	}
	var name, status string
	var latestTemplate int
	err = h.db.QueryRowContext(r.Context(), `SELECT name,status,revision FROM delivery_template WHERE id=? AND kind='helm'`, ref.ID).Scan(&name, &status, &latestTemplate)
	if err != nil && err != sql.ErrNoRows {
		response.Error(w, errcode.CodeDBError)
		return
	}
	v, versionErr := templateversion.Load(r.Context(), h.db, ref, false)
	available := versionErr == nil
	if mode == "template" && (!available || !helmconfig.SameFiles(definition, v.Definition)) {
		mode = "custom"
	}
	result["mode"] = mode
	result["template"] = ref
	result["name"] = name
	result["latest_revision"] = latestTemplate
	result["archived"] = status != "active"
	result["available"] = available
	result["update_available"] = mode == "template" && status == "active" && latestTemplate > ref.Revision
	response.Success(w, result)
}

type templateUsage struct {
	ProjectID       int64  `json:"project_id"`
	ServiceID       int64  `json:"service_id"`
	ServiceName     string `json:"service_name"`
	EnvironmentID   int64  `json:"environment_id"`
	EnvironmentName string `json:"environment_name"`
	Revision        int    `json:"revision"`
}

func (h *Handler) Usages(w http.ResponseWriter, r *http.Request) {
	if !h.allowed(w, r, "view") {
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if id <= 0 {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	// Only return authorized service/environment identities, never values or
	// names from other projects through a shared template's consumer list.
	rows, err := h.db.QueryContext(r.Context(), `SELECT s.project_id,s.id,s.name,COALESCE(e.id,0),COALESCE(e.name,''),COALESCE(e.code,''),COALESCE(e.cluster_id,0),sr.source_commit
	 FROM app_service s JOIN service_revision sr ON sr.id=s.current_revision_id AND sr.service_id=s.id
	 LEFT JOIN project_environment e ON 1=0 WHERE s.status='active' AND s.delivery_type='helm' AND sr.source_commit LIKE ?
	 UNION ALL SELECT s.project_id,s.id,s.name,e.id,e.name,e.code,e.cluster_id,sr.source_commit
	 FROM environment_service_override o JOIN app_service s ON s.id=o.service_id
	 JOIN project_environment e ON e.id=o.project_environment_id AND e.project_id=s.project_id
	 JOIN service_revision sr ON sr.id=o.service_revision_id AND sr.service_id=s.id
	 WHERE s.status='active' AND e.status='active' AND s.delivery_type='helm' AND sr.source_commit LIKE ?
	 ORDER BY 1,2,4`, fmt.Sprintf("template:%d@%%", id), fmt.Sprintf("template:%d@%%", id))
	if err != nil {
		response.Error(w, errcode.CodeDBError)
		return
	}
	type pending struct {
		item         templateUsage
		code, source string
		cluster      int64
	}
	var all []pending
	for rows.Next() {
		var p pending
		if rows.Scan(&p.item.ProjectID, &p.item.ServiceID, &p.item.ServiceName, &p.item.EnvironmentID, &p.item.EnvironmentName, &p.code, &p.cluster, &p.source) != nil {
			rows.Close()
			response.Error(w, errcode.CodeDBError)
			return
		}
		all = append(all, p)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		response.Error(w, errcode.CodeDBError)
		return
	}
	items := []templateUsage{}
	for _, p := range all {
		req := authz.Request{Resource: "service", Action: "view", ProjectID: p.item.ProjectID, ServiceID: p.item.ServiceID, EnvironmentID: p.item.EnvironmentID, Environment: p.code, ClusterID: p.cluster}
		allowed, err := h.auth.Authorize(r.Context(), middleware.GetUserID(r.Context()), req)
		if err != nil {
			response.Error(w, errcode.CodeDBError)
			return
		}
		if !allowed {
			continue
		}
		if p.item.EnvironmentID > 0 {
			req.Resource = "environment"
			allowed, err = h.auth.Authorize(r.Context(), middleware.GetUserID(r.Context()), req)
			if err != nil {
				response.Error(w, errcode.CodeDBError)
				return
			}
			if !allowed {
				continue
			}
		}
		ref, ok := templateversion.ParseSource(p.source)
		if !ok || ref.ID != id {
			continue
		}
		p.item.Revision = ref.Revision
		items = append(items, p.item)
	}
	response.Success(w, items)
}
