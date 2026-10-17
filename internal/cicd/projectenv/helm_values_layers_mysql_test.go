package projectenv

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ChenJustin666/axiops/internal/cicd/helmconfig"
	"github.com/ChenJustin666/axiops/internal/cicd/workflow"
)

func testHelmValuesLayers(t *testing.T, db *sql.DB) {
	ctx := context.Background()
	repo := NewRepository(db)
	get := func(eid int64) *ServiceConfig {
		t.Helper()
		c, e := readServiceConfig(ctx, db, eid, 21, false)
		if e != nil {
			t.Fatal(e)
		}
		return c
	}
	logic := NewLogic(db, helmTemplateTestAccess{})
	h := NewHandler(logic)
	call := func(method, url, body string) struct {
		Code int `json:"code"`
		Data struct {
			Values   string          `json:"values"`
			Revision int64           `json:"revision"`
			Saved    bool            `json:"saved"`
			Services []sharedService `json:"services"`
		} `json:"data"`
	} {
		t.Helper()
		r := httptest.NewRequest(method, url, strings.NewReader(body))
		r.SetPathValue("id", "31")
		w := httptest.NewRecorder()
		h.HelmSharedValues(w, r)
		var result struct {
			Code int `json:"code"`
			Data struct {
				Values   string          `json:"values"`
				Revision int64           `json:"revision"`
				Saved    bool            `json:"saved"`
				Services []sharedService `json:"services"`
			} `json:"data"`
		}
		if e := json.Unmarshal(w.Body.Bytes(), &result); e != nil {
			t.Fatal(e)
		}
		return result
	}
	current := call("GET", "/", "")
	if current.Code != 0 || current.Data.Revision != 0 || len(current.Data.Services) != 1 {
		t.Fatalf("read shared code=%d", current.Code)
	}
	tokens := map[int64]string{21: get(31).Token}
	request := sharedValuesRequest{Values: "host: shared.example.com\nregion: qa-region\n", Revision: 0, Tokens: tokens}
	raw, _ := json.Marshal(request)
	before := get(31).Token
	preview := call("POST", "/?preview=1", string(raw))
	if preview.Code != 0 || preview.Data.Saved || get(31).Token != before || len(preview.Data.Services) != 1 || !preview.Data.Services[0].Changed {
		t.Fatal("shared preview persisted or lacked impact")
	}
	saved := call("POST", "/", string(raw))
	if saved.Code != 0 || !saved.Data.Saved || get(31).Token == before || strings.Contains(get(32).Helm.SharedValues, "qa-region") {
		t.Fatal("shared save crossed environment")
	}
	if call("POST", "/", string(raw)).Code == 0 {
		t.Fatal("stale shared save accepted")
	}
	tx, e := db.BeginTx(ctx, nil)
	if e != nil {
		t.Fatal(e)
	}
	_, final, e := workflow.RenderHelmServicePreview(ctx, tx, nil, 29, 31, 21, get(31).RevisionID)
	_ = tx.Rollback()
	if e != nil || !strings.Contains(final, "changed-qa.example.com") || !strings.Contains(final, "qa-region") {
		t.Fatal("shared precedence or final values wrong", e)
	}
	nodes, e := workflow.NewRepository(db).FreezeDeploymentConfigs(ctx, 31, []workflow.Node{{Type: workflow.NodeHelmDeploy, Config: json.RawMessage(`{"service_id":21,"release_name":"helm-api"}`)}}, false)
	if e != nil {
		t.Fatal(e)
	}
	var frozen workflow.HelmDeployConfig
	_ = json.Unmarshal(nodes[0].Config, &frozen)
	current = call("GET", "/", "")
	request.Revision = current.Data.Revision
	request.Tokens = map[int64]string{21: get(31).Token}
	request.Values = "region: new-region\n"
	raw, _ = json.Marshal(request)
	if call("POST", "/", string(raw)).Code != 0 {
		t.Fatal("shared second update failed")
	}
	target, e := workflow.NewDefaultHelmResolver(workflow.NewSQLHelmSourceStore(db), nil).RenderHelmConfig(ctx, workflow.WorkflowRun{ProjectID: 29, EnvironmentID: 31}, frozen)
	if e != nil || target.Values["region"] != "qa-region" {
		t.Fatal("queued shared snapshot was not frozen", e)
	}
	resources, e := repo.ListResources(ctx, 31)
	if e != nil || len(resources) != 0 {
		t.Fatal("shared internal resource exposed", e)
	}
	// A client must never inject environment-owned shared content through a service save.
	c := get(31)
	settings := *c.Helm
	settings.SharedValues = "region: forged\n"
	settings.SharedRevision = 999
	req := SaveConfigRequest{Targets: []ConfigTarget{{EnvironmentID: 31, Token: c.Token}}, RevisionID: c.RevisionID, Helm: &settings}
	if e = repo.saveServiceConfig(ctx, 7, 29, 21, &req, false); e != nil {
		t.Fatal(e)
	}
	if strings.Contains(get(31).Helm.SharedValues, "forged") {
		t.Fatal("service save forged shared values")
	}
	var stored string
	if e = db.QueryRow(`SELECT manifest FROM environment_resource WHERE environment_id=31 AND name=? AND kind=?`, helmconfig.RowName(21), helmconfig.Kind).Scan(&stored); e != nil {
		t.Fatal(e)
	}
	var persisted helmconfig.Settings
	_ = json.Unmarshal([]byte(stored), &persisted)
	if persisted.SharedValues != "" || persisted.SharedRevision != 0 {
		t.Fatal("shared values duplicated into service row")
	}
	t.Run("Git values provenance is verified", func(t *testing.T) {
		provider := &fakeValuesCodeSource{projectID: 29, commit: "commit-a", raw: "region: from-git\n"}
		logic.WithCodeSources(provider)
		settings.Source = &helmconfig.ValuesSource{CodeSourceID: 9, RepoPath: "team/config", Ref: "main", Path: "values-qa.yaml", CommitID: "commit-a", Base: provider.raw}
		request := SaveConfigRequest{Helm: &settings, Targets: []ConfigTarget{{EnvironmentID: 31, Token: get(31).Token}}}
		if e := logic.validateValuesSource(ctx, 7, 29, 21, &request); e != nil {
			t.Fatal(e)
		}
		settings.Source.Base = "forged: true\n"
		if e := logic.validateValuesSource(ctx, 7, 29, 21, &request); e == nil {
			t.Fatal("forged import baseline accepted")
		}
		settings.Source.Base = provider.raw
		provider.commit = "commit-b"
		if e := logic.validateValuesSource(ctx, 7, 29, 21, &request); e == nil {
			t.Fatal("branch movement after preview accepted")
		}
		settings.Source = nil
	})
	t.Run("shared invalid update rolls back", func(t *testing.T) {
		current = call("GET", "/", "")
		before := get(31).Token
		request = sharedValuesRequest{Values: "image: invalid-scalar\n", Revision: current.Data.Revision, Tokens: map[int64]string{21: before}}
		raw, _ = json.Marshal(request)
		if call("POST", "/", string(raw)).Code == 0 || get(31).Token != before {
			t.Fatal("invalid shared values persisted")
		}
	})
	t.Run("new environment is all or nothing", func(t *testing.T) {
		makeEnv := func(code string, preview bool) *ProjectEnvironment {
			return &ProjectEnvironment{ProjectID: 29, Name: code, Code: code, EnvironmentType: "k8s_namespace", ClusterID: 2, Namespace: "helm-" + code, BindingStatus: "validated", Status: "active", CreatedBy: 7, preview: preview, initialHelmServices: []InitialHelmService{{ServiceID: 21, RevisionID: 211, Helm: helmconfig.Default()}}}
		}
		env := makeEnv("review", true)
		if _, e := repo.Create(ctx, env); e != nil {
			t.Fatal(e)
		}
		var count int
		if e := db.QueryRow(`SELECT COUNT(*) FROM project_environment WHERE project_id=29 AND code='review'`).Scan(&count); e != nil || count != 0 {
			t.Fatal("preview created environment", e)
		}
		if len(env.initialConfigs) != 1 || env.initialConfigs[0].FinalValues == "" {
			t.Fatal("environment preview omitted final values")
		}
		env = makeEnv("atomic", false)
		env.initialHelmServices = append(env.initialHelmServices, InitialHelmService{ServiceID: 999999, RevisionID: 211, Helm: helmconfig.Default()})
		if _, e := repo.Create(ctx, env); e == nil {
			t.Fatal("invalid second service accepted")
		}
		if e := db.QueryRow(`SELECT COUNT(*) FROM project_environment WHERE project_id=29 AND code='atomic'`).Scan(&count); e != nil || count != 0 {
			t.Fatal("invalid second service left a partial environment", e)
		}
		env = makeEnv("created", false)
		id, e := repo.Create(ctx, env)
		if e != nil {
			t.Fatal(e)
		}
		if !get(id).Bound || get(id).Helm.Values != "{}\n" {
			t.Fatal("creation did not bind service or fixed all chart defaults")
		}
	})
	// Restore shared input for tests that follow; no business schema is used.
	if _, e = db.Exec(`DELETE FROM environment_resource WHERE environment_id=31 AND kind=?`, helmconfig.SharedKind); e != nil {
		t.Fatal(e)
	}
}
