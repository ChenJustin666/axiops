package projectenv

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/ChenJustin666/axiops/internal/cicd/appsvc"
	"github.com/ChenJustin666/axiops/internal/cicd/helmconfig"
	"github.com/ChenJustin666/axiops/internal/cicd/templatelib"
	"github.com/ChenJustin666/axiops/internal/cicd/templateversion"
	"github.com/ChenJustin666/axiops/internal/cicd/testschema"
	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
)

type helmTemplateTestAccess struct {
	denyProject, denyEnvironment int64
	denyService                  bool
}

func (helmTemplateTestAccess) Scopes(context.Context, int64, string, string) (authz.ScopeSet, error) {
	return authz.ScopeSet{AllowAll: true}, nil
}
func (a helmTemplateTestAccess) Authorize(_ context.Context, _ int64, r authz.Request) (bool, error) {
	return !(a.denyProject > 0 && r.ProjectID == a.denyProject || a.denyEnvironment > 0 && r.EnvironmentID == a.denyEnvironment || a.denyService && r.Resource == "service"), nil
}
func (helmTemplateTestAccess) Policies(context.Context, int64) ([]authz.Policy, error) {
	return nil, nil
}

type helmMultiFieldDecryptor struct{}

func (helmMultiFieldDecryptor) Decrypt(string) (string, error) {
	return `{"password":"do-not-expose-password","SK":"do-not-expose-sk"}`, nil
}

func testHelmTemplateUpgrade(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	testschema.TemplateVersions(t, db)
	h := templatelib.NewHandler(db, helmTemplateTestAccess{})
	if err := h.Seed(ctx); err != nil {
		t.Fatal(err)
	}
	var template templatelib.Template
	for _, item := range templatelib.Defaults() {
		if item.BuiltinKey == "helm-service" {
			template = item
		}
	}
	if err := db.QueryRow(`SELECT id,revision FROM delivery_template WHERE builtin_key='helm-service'`).Scan(&template.ID, &template.Revision); err != nil {
		t.Fatal(err)
	}
	initial, err := templateversion.Load(ctx, db, templateversion.Ref{ID: template.ID, Revision: 1}, true)
	if err != nil {
		t.Fatal(err)
	}
	publish := func(item templatelib.Template) templatelib.Template {
		t.Helper()
		raw, _ := json.Marshal(item)
		req := httptest.NewRequest("PUT", "/", bytes.NewReader(raw))
		req.SetPathValue("id", strconv.FormatInt(item.ID, 10))
		w := httptest.NewRecorder()
		h.Save(w, req)
		if w.Code != 200 {
			t.Fatalf("template save: %s", w.Body.String())
		}
		var response struct {
			Data templatelib.Template `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		return response.Data
	}
	template.Files = append(template.Files, templatelib.File{Path: "templates/config.yaml", Content: "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: {{ .Release.Name }}-runtime\ndata:\n  host: {{ default \"\" .Values.host | quote }}\n  passwordB64: {{ dig \"runtime\" \"password\" \"\" .Values | b64enc | quote }}\n"})
	// dig takes native maps; omit a value-independent transformation in the
	// initial template and add secret transform only in the credential phase.
	template.Files[len(template.Files)-1].Content = "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: {{ .Release.Name }}-runtime\ndata:\n  host: {{ default \"\" .Values.host | quote }}\n  passwordB64: {{ default \"\" (index (default dict .Values.runtime) \"password\") | b64enc | quote }}\n"
	template = publish(template)
	if template.Revision != 2 {
		t.Fatal("new template version missing")
	}
	logic := appsvc.NewLogic(db, helmTemplateTestAccess{})
	if _, code := logic.CreateRevision(ctx, 7, 21, &appsvc.CreateServiceRevisionRequest{SourceCommit: templateversion.Ref{ID: template.ID, Revision: 2}.SourceCommit(), Definition: initial.Definition}); code != errcode.CodeParamInvalid {
		t.Fatal("client forged template provenance")
	}
	if _, code := logic.CreateRevision(ctx, 7, 21, &appsvc.CreateServiceRevisionRequest{Template: &templateversion.Ref{ID: template.ID, Revision: 2}, Definition: initial.Definition}); code != errcode.CodeServiceDefinitionInvalid {
		t.Fatal("modified Chart claimed stored template version")
	}
	old, err := templateversion.Load(ctx, db, templateversion.Ref{ID: template.ID, Revision: 1}, true)
	if err != nil || old.ContentHash != initial.ContentHash || old.Definition != initial.Definition {
		t.Fatal("template history mutated", err)
	}
	// Optimistic template save must not overwrite the newer version.
	stale := template
	stale.Revision = 1
	raw, _ := json.Marshal(stale)
	req := httptest.NewRequest("PUT", "/", bytes.NewReader(raw))
	req.SetPathValue("id", strconv.FormatInt(template.ID, 10))
	w := httptest.NewRecorder()
	h.Save(w, req)
	if w.Code == 200 {
		t.Fatal("stale template save succeeded")
	}
	if err := h.Seed(ctx); err != nil {
		t.Fatal(err)
	}
	var currentRevision int
	if err := db.QueryRow(`SELECT revision FROM delivery_template WHERE id=?`, template.ID).Scan(&currentRevision); err != nil || currentRevision != 2 {
		t.Fatal("seed overwrote user template", err)
	}
	repo := NewRepository(db)
	repo.decryptor = helmMultiFieldDecryptor{}
	get := func(eid int64) *ServiceConfig {
		t.Helper()
		c, err := readServiceConfig(ctx, db, eid, 21, false)
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	count := func() int {
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM service_revision WHERE service_id=21`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	settings := *get(31).Helm
	settings.Values = "replicaCount: 3\nhost: qa.example.com\n"
	prodBefore := get(32)
	beforeCount := count()
	before := get(31)
	upgrade := SaveConfigRequest{Template: &templateversion.Ref{ID: template.ID, Revision: 2}, Helm: &settings, Targets: []ConfigTarget{{31, before.Token}}}
	if err := repo.saveServiceConfig(ctx, 7, 29, 21, &upgrade, true); err != nil {
		t.Fatal("upgrade preview", err)
	}
	if get(31).Token != before.Token || count() != beforeCount || !strings.Contains(upgrade.Result[len(upgrade.Result)-1].Manifest, "qa.example.com") {
		t.Fatal("preview persisted or failed to render environment values")
	}
	upgrade.Result = nil
	if err := repo.saveServiceConfig(ctx, 7, 29, 21, &upgrade, false); err != nil {
		t.Fatal("upgrade save", err)
	}
	after := get(31)
	if after.RevisionID == 211 || count() != beforeCount+1 || get(32).RevisionID != prodBefore.RevisionID || get(32).Helm.Values != prodBefore.Helm.Values {
		t.Fatal("template upgrade crossed environment or lost revision")
	}
	var source string
	var number int64
	if err := db.QueryRow(`SELECT source_commit,revision_number FROM service_revision WHERE id=?`, after.RevisionID).Scan(&source, &number); err != nil || source != upgrade.Template.SourceCommit() || number != 2 {
		t.Fatal("invalid provenance or numbering", err)
	}
	if err := repo.saveServiceConfig(ctx, 7, 29, 21, &upgrade, false); err == nil {
		t.Fatal("stale environment token accepted")
	}
	upgrade.Targets[0].Token = after.Token
	if err := repo.saveServiceConfig(ctx, 7, 29, 21, &upgrade, false); err != nil || count() != beforeCount+1 {
		t.Fatal("identical template created duplicate revision", err)
	}
	// A new schema incompatible with current values rolls back the service
	// revision, environment binding and current revision pointer together.
	template.Files = append(template.Files, templatelib.File{Path: "values.schema.json", Content: `{"type":"object","properties":{"replicaCount":{"type":"integer","maximum":1}}}`})
	template = publish(template)
	before = get(31)
	beforeCount = count()
	upgrade.Template.Revision = 3
	upgrade.Targets[0].Token = before.Token
	if err := repo.saveServiceConfig(ctx, 7, 29, 21, &upgrade, false); err == nil || get(31).Token != before.Token || count() != beforeCount {
		t.Fatal("incompatible schema partially persisted")
	}
	// Multi-field credential bindings remain local and public transforms redact.
	if _, err := db.Exec(`INSERT INTO credential(id,name,type,description,data,project_id,created_by) VALUES(901,'Helm runtime','secret','','encrypted',29,7)`); err != nil {
		t.Fatal(err)
	}
	bind := SaveConfigRequest{Helm: &settings, Targets: []ConfigTarget{{31, get(31).Token}}, HelmCredentials: map[string]*helmconfig.CredentialBinding{"runtime.password": {CredentialID: 901, Key: "password"}, "runtime.sk": {CredentialID: 901, Key: "SK"}}}
	bind.validateHelmCredentials = func(context.Context, *sql.Tx, int64, map[string]*helmconfig.CredentialBinding) error { return nil }
	if err := repo.saveServiceConfig(ctx, 7, 29, 21, &bind, false); err != nil {
		t.Fatal("credential binding", err)
	}
	manifest := bind.Result[len(bind.Result)-1].Manifest
	if strings.Contains(bind.Result[len(bind.Result)-1].FinalValues, "do-not-expose") {
		t.Fatal("credential leaked in final values")
	}
	if strings.Contains(manifest, "do-not-expose") || strings.Contains(manifest, base64.StdEncoding.EncodeToString([]byte("do-not-expose-password"))) {
		t.Fatal("credential leaked in rendered preview")
	}
	if get(31).Helm.CredentialKeys["runtime.sk"] != "SK" || len(get(32).Helm.CredentialKeys) != 0 {
		t.Fatal("credential field references lost or crossed environments")
	}
	// Ordinary values saves and forged hidden key mappings cannot change the
	// selected credential fields without explicit authorization.
	settings.CredentialKeys = map[string]string{"runtime.password": "SK"}
	preserve := SaveConfigRequest{Helm: &settings, Targets: []ConfigTarget{{31, get(31).Token}}}
	if err := repo.saveServiceConfig(ctx, 7, 29, 21, &preserve, false); err != nil || get(31).Helm.CredentialKeys["runtime.password"] != "password" {
		t.Fatal("hidden credential mapping changed", err)
	}
	denied := SaveConfigRequest{Helm: &settings, Targets: []ConfigTarget{{31, get(31).Token}}, HelmCredentials: map[string]*helmconfig.CredentialBinding{"runtime.sk": {CredentialID: 901, Key: "SK"}}}
	before = get(31)
	if err := repo.saveServiceConfig(ctx, 7, 29, 21, &denied, false); err == nil || get(31).Token != before.Token {
		t.Fatal("unauthorized credential write persisted")
	}
	// The shared library never reveals consumer identities in denied projects
	// or environments, even when the caller can read the shared Chart itself.
	for _, fixture := range []struct {
		sid, rid, pid, eid int64
		name               string
	}{{22, 2222, 29, 32, "hidden-prod-consumer"}, {23, 2323, 19, 1, "hidden-project-consumer"}} {
		if _, err := db.Exec(`INSERT INTO app_service(id,project_id,name,code,delivery_type,source_type,source_ref,image_name,current_revision_id) VALUES(?,?,?,?,'helm','template','{}','',?)`, fixture.sid, fixture.pid, fixture.name, fixture.name, fixture.rid); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO service_revision(id,service_id,revision_number,definition_format,source_commit,definition,content_hash) VALUES(?,?,1,'helm_chart',?,?,'fixture')`, fixture.rid, fixture.sid, templateversion.Ref{ID: template.ID, Revision: 2}.SourceCommit(), initial.Definition); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO environment_service_override(project_environment_id,service_id,service_revision_id,image_ref,replicas,revision,updated_by) VALUES(?,?,?,'',NULL,1,7)`, fixture.eid, fixture.sid, fixture.rid); err != nil {
			t.Fatal(err)
		}
	}
	filtered := templatelib.NewHandler(db, helmTemplateTestAccess{denyProject: 19, denyEnvironment: 32})
	req = httptest.NewRequest("GET", "/", nil)
	req.SetPathValue("id", strconv.FormatInt(template.ID, 10))
	w = httptest.NewRecorder()
	filtered.Usages(w, req)
	if w.Code != 200 || strings.Contains(w.Body.String(), "hidden-project-consumer") || strings.Contains(w.Body.String(), `"environment_id":32`) {
		t.Fatal("consumer scope leaked", w.Body.String())
	}
	req = httptest.NewRequest("GET", "/?revision_id="+strconv.FormatInt(get(31).RevisionID, 10), nil)
	req.SetPathValue("id", "21")
	w = httptest.NewRecorder()
	h.ServiceTemplate(w, req)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"mode":"template"`) || !strings.Contains(w.Body.String(), `"revision":2`) {
		t.Fatal("environment source status incorrect", w.Body.String())
	}
	w = httptest.NewRecorder()
	templatelib.NewHandler(db, helmTemplateTestAccess{denyService: true}).ServiceTemplate(w, req)
	if w.Code != 403 {
		t.Fatal("denied service source exposed")
	}
	custom, code := logic.CreateRevision(ctx, 7, 21, &appsvc.CreateServiceRevisionRequest{SourceCommit: "manual-chart-edit", Definition: initial.Definition})
	if code != errcode.CodeSuccess {
		t.Fatal("custom revision failed", code)
	}
	req = httptest.NewRequest("GET", "/?revision_id="+strconv.FormatInt(custom.ID, 10), nil)
	req.SetPathValue("id", "21")
	w = httptest.NewRecorder()
	h.ServiceTemplate(w, req)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"mode":"custom"`) {
		t.Fatal("custom Chart treated as auto-updateable", w.Body.String())
	}
	// Archiving stops new use; already pinned service definitions remain readable.
	req = httptest.NewRequest("DELETE", "/?revision=3", nil)
	req.SetPathValue("id", strconv.FormatInt(template.ID, 10))
	w = httptest.NewRecorder()
	h.Archive(w, req)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	upgrade.Template.Revision = 2
	upgrade.Targets[0].Token = get(31).Token
	if err := repo.saveServiceConfig(ctx, 7, 29, 21, &upgrade, false); err == nil {
		t.Fatal("archived template accepted for new binding")
	}
	if _, err := templateversion.Load(ctx, db, templateversion.Ref{ID: template.ID, Revision: 2}, false); err != nil {
		t.Fatal("archive removed history", err)
	}
}
