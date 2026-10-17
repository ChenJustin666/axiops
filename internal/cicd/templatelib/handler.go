package templatelib

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/ChenJustin666/axiops/internal/cicd/templateversion"
	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"github.com/ChenJustin666/axiops/internal/shared/middleware"
	"github.com/ChenJustin666/axiops/internal/shared/response"
)

type Handler struct {
	db   *sql.DB
	auth authz.Checker
}

func NewHandler(db *sql.DB, auth authz.Checker) *Handler { return &Handler{db, auth} }
func (h *Handler) allowed(w http.ResponseWriter, r *http.Request, action string) bool {
	// Existing template permissions: readers can consume shared templates;
	// writes require platform scope because these templates are cross-project.
	allowed := false
	var err error
	if action == "view" {
		var s authz.ScopeSet
		s, err = h.auth.Scopes(r.Context(), middleware.GetUserID(r.Context()), "build_template", action)
		allowed = s.HasAnyAllow()
	} else {
		allowed, err = h.auth.Authorize(r.Context(), middleware.GetUserID(r.Context()), authz.Request{Resource: "build_template", Action: action})
	}
	if err != nil {
		response.Error(w, errcode.CodeDBError)
		return false
	}
	if !allowed {
		response.Error(w, errcode.CodeNoPermission)
		return false
	}
	return true
}
func decode(w http.ResponseWriter, r *http.Request) (Template, bool) {
	var t Template
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 6<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&t); err != nil {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "模板请求格式无效或超过大小限制")
		return t, false
	}
	if decoder.Decode(new(interface{})) != io.EOF {
		response.Error(w, errcode.CodeParamInvalid)
		return t, false
	}
	if err := Validate(t); err != nil {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, err.Error())
		return t, false
	}
	return t, true
}

const columns = `id,kind,name,description,files,variables,revision,COALESCE(builtin_key,''),updated_at`

func scan(row interface{ Scan(...interface{}) error }) (Template, error) {
	var t Template
	var files, vars []byte
	err := row.Scan(&t.ID, &t.Kind, &t.Name, &t.Description, &files, &vars, &t.Revision, &t.BuiltinKey, &t.UpdatedAt)
	if err != nil {
		return t, err
	}
	if err = json.Unmarshal(files, &t.Files); err != nil {
		return t, err
	}
	err = json.Unmarshal(vars, &t.Variables)
	return t, err
}
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	if !h.allowed(w, r, "view") {
		return
	}
	rows, err := h.db.QueryContext(r.Context(), `SELECT `+columns+` FROM delivery_template WHERE status='active' AND (?='' OR kind=?) ORDER BY builtin_key IS NULL,id`, r.URL.Query().Get("kind"), r.URL.Query().Get("kind"))
	if err != nil {
		response.Error(w, errcode.CodeDBError)
		return
	}
	defer rows.Close()
	result := []Template{}
	for rows.Next() {
		t, err := scan(rows)
		if err != nil {
			response.Error(w, errcode.CodeDBError)
			return
		}
		result = append(result, t)
	}
	if rows.Err() != nil {
		response.Error(w, errcode.CodeDBError)
		return
	}
	response.Success(w, result)
}
func (h *Handler) Preview(w http.ResponseWriter, r *http.Request) {
	if !h.allowed(w, r, "view") {
		return
	}
	t, ok := decode(w, r)
	if !ok {
		return
	}
	definition, err := Snapshot(t)
	if err != nil {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, err.Error())
		return
	}
	response.Success(w, map[string]string{"definition": definition})
}
func (h *Handler) Save(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	action := "create"
	if r.Method == http.MethodPut {
		action = "update"
		if id <= 0 {
			response.Error(w, errcode.CodeParamInvalid)
			return
		}
	}
	if !h.allowed(w, r, action) {
		return
	}
	t, ok := decode(w, r)
	if !ok {
		return
	}
	files, _ := json.Marshal(t.Files)
	if t.Variables == nil {
		t.Variables = []Variable{}
	}
	vars, _ := json.Marshal(t.Variables)
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		response.Error(w, errcode.CodeDBError)
		return
	}
	defer tx.Rollback()
	var result sql.Result
	if action == "create" {
		result, err = tx.ExecContext(r.Context(), `INSERT INTO delivery_template(kind,name,description,files,variables,created_by) VALUES (?,?,?,?,?,?)`, t.Kind, t.Name, t.Description, string(files), string(vars), middleware.GetUserID(r.Context()))
	} else {
		result, err = tx.ExecContext(r.Context(), `UPDATE delivery_template SET name=?,description=?,files=?,variables=?,revision=revision+1 WHERE id=? AND kind=? AND revision=? AND status='active'`, t.Name, t.Description, string(files), string(vars), id, t.Kind, t.Revision)
	}
	if err != nil {
		response.Error(w, errcode.CodeDBError)
		return
	}
	if action == "create" {
		id, err = result.LastInsertId()
	} else {
		count, e := result.RowsAffected()
		err = e
		if count != 1 && err == nil {
			response.ErrorWithMsg(w, errcode.CodeParamInvalid, "模板已被其他人修改或归档，请刷新后重试；本地内容未保存")
			return
		}
	}
	if err != nil {
		response.Error(w, errcode.CodeDBError)
		return
	}
	saved, err := scan(tx.QueryRowContext(r.Context(), `SELECT `+columns+` FROM delivery_template WHERE id=?`, id))
	if err != nil {
		response.Error(w, errcode.CodeDBError)
		return
	}
	if err = persistVersion(r.Context(), tx, saved, middleware.GetUserID(r.Context())); err != nil {
		response.Error(w, errcode.CodeDBError)
		return
	}
	if err = tx.Commit(); err != nil {
		response.Error(w, errcode.CodeDBError)
		return
	}
	response.Success(w, saved)
}
func (h *Handler) Archive(w http.ResponseWriter, r *http.Request) {
	if !h.allowed(w, r, "delete") {
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	rev, _ := strconv.Atoi(r.URL.Query().Get("revision"))
	if id <= 0 || rev <= 0 {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	result, err := h.db.ExecContext(r.Context(), `UPDATE delivery_template SET status='archived' WHERE id=? AND revision=? AND status='active'`, id, rev)
	if err != nil {
		response.Error(w, errcode.CodeDBError)
		return
	}
	count, err := result.RowsAffected()
	if err != nil {
		response.Error(w, errcode.CodeDBError)
		return
	}
	if count != 1 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "模板已变更，请刷新后重试")
		return
	}
	response.Success(w, nil)
}
func (h *Handler) Seed(ctx context.Context) error {
	for _, t := range Defaults() {
		if err := Validate(t); err != nil {
			return err
		}
		files, _ := json.Marshal(t.Files)
		vars, _ := json.Marshal(t.Variables)
		if _, err := h.db.ExecContext(ctx, `INSERT INTO delivery_template(kind,name,description,files,variables,builtin_key,created_by) VALUES (?,?,?,?,?,?,0) ON DUPLICATE KEY UPDATE builtin_key=VALUES(builtin_key)`, t.Kind, t.Name, t.Description, string(files), string(vars), t.BuiltinKey); err != nil {
			return err
		}
		if t.BuiltinKey == "k8s-java17-nacos" {
			if err := h.updateUntouchedNacosTemplate(ctx, t); err != nil {
				return err
			}
		}
	}
	return h.seedVersions(ctx)
}

func persistVersion(ctx context.Context, tx *sql.Tx, t Template, uid int64) error {
	definition, err := Snapshot(t)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(t)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO delivery_template_version(template_id,revision,snapshot,definition,content_hash,created_by) VALUES(?,?,?,?,?,?)`, t.ID, t.Revision, raw, definition, templateversion.Hash(definition), uid)
	return err
}

// Historical versions overwritten before this feature cannot be reconstructed.
// Backfill only the exact current content and preserve every future version.
func (h *Handler) seedVersions(ctx context.Context) error {
	rows, err := h.db.QueryContext(ctx, `SELECT id FROM delivery_template`)
	if err != nil {
		return err
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, id := range ids {
		tx, err := h.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		t, err := scan(tx.QueryRowContext(ctx, `SELECT `+columns+` FROM delivery_template WHERE id=? FOR UPDATE`, id))
		if err == nil {
			var count int
			err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM delivery_template_version WHERE template_id=? AND revision=?`, id, t.Revision).Scan(&count)
			if err == nil && count == 0 {
				err = persistVersion(ctx, tx, t, 0)
			}
		}
		if err != nil {
			tx.Rollback()
			return err
		}
		if err = tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

// The built-in template is copied into each service. Update only the exact
// previous built-in in the library; user-edited templates and service copies
// must keep their own content.
func (h *Handler) updateUntouchedNacosTemplate(ctx context.Context, current Template) error {
	const legacySecret = `            # With Nacos authentication enabled, create this Secret and remove optional: true.
            - name: SPRING_CLOUD_NACOS_USERNAME
              valueFrom:
                secretKeyRef:
                  name: {{.Service}}-nacos-auth
                  key: username
                  optional: true
            - name: SPRING_CLOUD_NACOS_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: {{.Service}}-nacos-auth
                  key: password
                  optional: true
`
	if len(current.Files) == 0 {
		return nil
	}
	legacy := append([]File(nil), current.Files...)
	legacy[0].Content += legacySecret
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var id int64
	var revision int
	var raw []byte
	if err = tx.QueryRowContext(ctx, `SELECT id,revision,files FROM delivery_template WHERE builtin_key=? FOR UPDATE`, current.BuiltinKey).Scan(&id, &revision, &raw); err != nil {
		return err
	}
	var stored []File
	if err = json.Unmarshal(raw, &stored); err != nil {
		return err
	}
	if revision == 1 && len(stored) == len(legacy) {
		same := true
		for i := range legacy {
			if stored[i] != legacy[i] {
				same = false
				break
			}
		}
		if same {
			files, _ := json.Marshal(current.Files)
			if _, err = tx.ExecContext(ctx, `UPDATE delivery_template SET files=?,revision=revision+1 WHERE id=?`, files, id); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}
