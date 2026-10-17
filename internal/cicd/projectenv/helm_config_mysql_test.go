package projectenv

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ChenJustin666/axiops/internal/cicd/helmconfig"
	"github.com/ChenJustin666/axiops/internal/cicd/helmimport"
	"github.com/ChenJustin666/axiops/internal/cicd/workflow"
)

func testHelmNativeConfig(t *testing.T, db *sql.DB) {
	ctx := context.Background()
	exec := func(q string, args ...interface{}) {
		t.Helper()
		if _, err := db.Exec(q, args...); err != nil {
			t.Fatal(err)
		}
	}
	definition := helmimport.Definition{SchemaVersion: 1}
	for path, content := range map[string]string{
		"Chart.yaml":                "apiVersion: v2\nname: helm-api\nversion: 1.0.0\ntype: application\n",
		"values.yaml":               "replicas: 1\nimage:\n  repository: registry.example.com/team/api\n  tag: latest\nhost: default.invalid\n",
		"templates/deployment.yaml": "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: {{ .Release.Name }}\nspec:\n  replicas: {{ .Values.replicas }}\n  template:\n    spec:\n      containers:\n      - name: api\n        image: \"{{ .Values.image.repository }}:{{ .Values.image.tag }}\"\n        env:\n        - name: HOST\n          value: {{ .Values.host | quote }}\n",
	} {
		sum := sha256.Sum256([]byte(content))
		definition.Files = append(definition.Files, helmimport.ChartFile{Path: path, Content: content, SHA256: hex.EncodeToString(sum[:])})
	}
	raw, _ := json.Marshal(definition)
	exec(`INSERT INTO project(id,name,code,delivery_type,delivery_status) VALUES(29,'Helm','helm','helm','active')`)
	exec(`INSERT INTO app_service(id,project_id,name,code,delivery_type,source_type,source_ref,image_name,current_revision_id) VALUES(21,29,'helm-api','helm-api','helm','manual','{}','registry.example.com/team/api',211)`)
	exec(`INSERT INTO service_revision(id,service_id,revision_number,definition_format,definition,content_hash) VALUES(211,21,1,'helm_chart',?,'helm211')`, raw)
	exec(`INSERT INTO project_environment(id,project_id,name,code,environment_type,cluster_id,namespace,binding_status,is_production) VALUES(31,29,'QA','qa','k8s_namespace',2,'helm-qa','validated',0),(32,29,'Prod','prod','k8s_namespace',2,'helm-prod','validated',1)`)
	repo := NewRepository(db)
	get := func(eid int64) *ServiceConfig {
		t.Helper()
		c, err := readServiceConfig(ctx, db, eid, 21, false)
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	settings := helmconfig.Default()
	settings.Values = "replicas: 2\nhost: qa.example.com\n"
	request := SaveConfigRequest{Targets: []ConfigTarget{{EnvironmentID: 31, Token: get(31).Token}}, RevisionID: 211, Helm: &settings}
	old := get(31).Token
	if err := repo.saveServiceConfig(ctx, 7, 29, 21, &request, true); err != nil {
		t.Fatal(err)
	}
	if get(31).Bound || get(31).Token != old || !strings.Contains(request.Result[0].Manifest, "qa.example.com") {
		t.Fatal("preview did not render, or persisted configuration")
	}
	if !strings.Contains(request.Result[0].FinalValues, "replicas: 2") || !strings.Contains(request.Result[0].FinalValues, "qa.example.com") {
		t.Fatal("final values preview missing")
	}
	if err := repo.saveServiceConfig(ctx, 7, 29, 21, &request, false); err != nil {
		t.Fatal(err)
	}
	if get(31).Token == old || get(31).Helm.Values != settings.Values || get(32).Bound {
		t.Fatal("Helm save crossed environment or failed to change token")
	}
	if err := repo.saveServiceConfig(ctx, 7, 29, 21, &request, false); err == nil {
		t.Fatal("stale token saved")
	}
	settings.Values = "replicas: 4\nhost: prod.example.com\n"
	request = SaveConfigRequest{Targets: []ConfigTarget{{EnvironmentID: 32, Token: get(32).Token}}, RevisionID: 211, Helm: &settings}
	if err := repo.saveServiceConfig(ctx, 7, 29, 21, &request, false); err != nil {
		t.Fatal(err)
	}
	prodBefore := get(32).Token
	qaBefore := get(31).Token
	bad := helmconfig.Default()
	bad.Values = "replicas: [invalid\n"
	request = SaveConfigRequest{Targets: []ConfigTarget{{EnvironmentID: 31, Token: qaBefore}, {EnvironmentID: 32, Token: prodBefore}}, Helm: &bad}
	if err := repo.saveServiceConfig(ctx, 7, 29, 21, &request, false); err == nil || get(31).Token != qaBefore || get(32).Token != prodBefore {
		t.Fatal("invalid multi-target save partially persisted")
	}
	nodes, err := workflow.NewRepository(db).FreezeDeploymentConfigs(ctx, 31, []workflow.Node{{Type: workflow.NodeHelmDeploy, Config: json.RawMessage(`{"service_id":21,"service_revision_id":211,"release_name":"helm-api","image_ref":"forged"}`)}}, false)
	if err != nil {
		t.Fatal(err)
	}
	var cfg workflow.HelmDeployConfig
	_ = json.Unmarshal(nodes[0].Config, &cfg)
	settings.Values = "replicas: 3\nhost: changed-qa.example.com\n"
	request = SaveConfigRequest{Targets: []ConfigTarget{{EnvironmentID: 31, Token: get(31).Token}}, Helm: &settings}
	if err := repo.saveServiceConfig(ctx, 7, 29, 21, &request, false); err != nil {
		t.Fatal(err)
	}
	target, err := workflow.NewDefaultHelmResolver(workflow.NewSQLHelmSourceStore(db), nil).RenderHelmConfig(ctx, workflow.WorkflowRun{ProjectID: 29, EnvironmentID: 31}, cfg)
	if err != nil || target.Values["host"] != "qa.example.com" || cfg.ImageRef != "" || !strings.Contains(get(32).Helm.Values, "prod.example.com") {
		t.Fatal("queued snapshot changed or forged image survived", err)
	}
	resources, err := repo.ListResources(ctx, 31)
	if err != nil || len(resources) != 0 {
		t.Fatal("internal Helm values exposed as runtime resource", err)
	}
	var rid int64
	if err := db.QueryRow(`SELECT id FROM environment_resource WHERE environment_id=31 AND kind=?`, helmconfig.Kind).Scan(&rid); err != nil {
		t.Fatal(err)
	}
	if resource, err := repo.FindResource(ctx, 31, rid); err != nil || resource != nil {
		t.Fatal("generic API exposed internal Helm settings")
	}
	if deleted, err := repo.DeleteResource(ctx, 31, rid); err != nil || deleted {
		t.Fatal("generic delete removed Helm values")
	}
	t.Run("cloned environment requires explicit Helm configuration", func(t *testing.T) {
		cloned, err := repo.Clone(ctx, &ProjectEnvironment{ID: 31}, &ProjectEnvironment{
			ProjectID: 29, Name: "Staging", Code: "staging", EnvironmentType: "k8s_namespace",
			ClusterID: 2, Namespace: "helm-staging", BindingStatus: "validated",
		}, 7)
		if err != nil {
			t.Fatal(err)
		}
		c := get(cloned)
		if c.Bound || c.Helm == nil || strings.Contains(c.Helm.Values, "qa.example.com") {
			t.Fatal("clone silently bound Helm service or copied source values")
		}
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM environment_resource WHERE environment_id=? AND kind=?`, cloned, helmconfig.Kind).Scan(&count); err != nil || count != 0 {
			t.Fatal("clone copied internal Helm settings", err)
		}
	})
	t.Run("shared values and atomic environment creation", func(t *testing.T) { testHelmValuesLayers(t, db) })
	t.Run("shared template versions and credentials", func(t *testing.T) { testHelmTemplateUpgrade(t, db) })
}
