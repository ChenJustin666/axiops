package projectenv

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/ChenJustin666/axiops/internal/cicd/serviceassets"
	"github.com/ChenJustin666/axiops/internal/cicd/testschema"
	"github.com/ChenJustin666/axiops/internal/cicd/workflow"
	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/config"
	"github.com/ChenJustin666/axiops/internal/shared/middleware"
	"github.com/zeromicro/go-zero/core/conf"
)

func TestMySQLServiceEnvironmentConfig(t *testing.T) {
	path := os.Getenv("AXIOPS_TEST_CONFIG")
	if path == "" {
		t.Skip("MySQL integration test not configured")
	}
	var cfg config.PlatformAPIConfig
	if conf.Load(path, &cfg) != nil {
		t.Fatal("could not load test config")
	}
	dsn, err := mysql.ParseDSN(cfg.MySQL.DSN())
	if err != nil {
		t.Fatal("invalid DSN")
	}
	admin, err := sql.Open("mysql", dsn.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	source := "`" + strings.ReplaceAll(dsn.DBName, "`", "``") + "`"
	schema := fmt.Sprintf("axiops_envconfig_test_%d", time.Now().UnixNano())
	if _, err = admin.Exec("CREATE DATABASE `" + schema + "`"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := admin.Exec("DROP DATABASE `" + schema + "`"); err != nil {
			t.Error(err)
		}
	}()
	dsn.DBName = schema
	dsn.ParseTime = true
	db, err := sql.Open("mysql", dsn.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	exec := func(query string, args ...interface{}) {
		t.Helper()
		if _, err := db.Exec(query, args...); err != nil {
			t.Fatal(err)
		}
	}
	for _, table := range []string{"app_service", "service_revision", "service_revision_sequence", "project", "cluster", "project_environment", "environment_service_override", "environment_resource", "variable_set", "variable_item", "credential", "deployment_snapshot"} {
		exec("CREATE TABLE `" + table + "` LIKE " + source + ".`" + table + "`")
	}
	testschema.EnvironmentYAML(t, db)
	exec(`INSERT INTO project(id,name,code,delivery_type,delivery_status) VALUES(19,'Test','test','k8s_yaml','active'),(20,'Other','other','k8s_yaml','active')`)
	exec(`INSERT INTO app_service(id,project_id,name,code,delivery_type,source_type,source_ref,image_name,current_revision_id) VALUES(1,19,'api','api','k8s_yaml','manual','{}','registry/team/api',11)`)
	definition := `# AxiOps-Variables: [{"key":"port","value":"8080"},{"key":"replicas","value":"1"}]
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.Service}}
spec:
  replicas: {{.replicas}}
  selector:
    matchLabels: {app: api}
  template:
    metadata:
      labels: {app: api}
    spec:
      containers:
      - name: api
        image: {{.Image}}
        ports:
        - containerPort: {{.port}}
`
	exec(`INSERT INTO service_revision(id,service_id,revision_number,definition_format,definition,content_hash) VALUES(10,1,1,'k8s_yaml',?,'test10'),(11,1,2,'k8s_yaml',?,'test11')`, definition, definition)
	exec(`INSERT INTO project_environment(id,project_id,name,code,environment_type,cluster_id,namespace,binding_status) VALUES(1,19,'Test','test','k8s_namespace',2,'api-test','validated'),(2,19,'Prod','prod','k8s_namespace',2,'api-prod','validated'),(3,20,'Other','other','k8s_namespace',2,'other','validated')`)
	ctx := context.Background()
	repo := NewRepository(db)
	get := func(eid int64) *ServiceConfig {
		t.Helper()
		c, err := readServiceConfig(ctx, db, eid, 1, false)
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	save := func(req SaveConfigRequest, preview bool) error {
		return repo.saveServiceConfig(ctx, 7, 19, 1, &req, preview)
	}
	target := func(eid int64) ConfigTarget { return ConfigTarget{EnvironmentID: eid, Token: get(eid).Token} }
	str := func(v string) *string { return &v }
	// New services may be joined before their first build supplies an image.
	exec(`UPDATE app_service SET image_name='' WHERE id=1`)
	// Reads do not create bindings or variable sets, preview must roll back.
	c := get(1)
	if c.Bound || c.RevisionID != 11 {
		t.Fatal("unexpected initial config")
	}
	req := SaveConfigRequest{Targets: []ConfigTarget{target(1), target(2)}, RevisionID: 10}
	if err = save(req, true); err != nil {
		t.Fatal(err)
	}
	if get(1).Bound || get(2).Bound {
		t.Fatal("preview persisted changes")
	}
	if err = save(req, false); err != nil {
		t.Fatal(err)
	}
	if get(1).RevisionID != 10 {
		t.Fatal("binding was not pinned")
	}
	var storedImage string
	if err = db.QueryRow(`SELECT image_name FROM app_service WHERE id=1`).Scan(&storedImage); err != nil || storedImage != "" || get(1).ImageRef != "" {
		t.Fatal("configuration validation persisted a temporary image")
	}
	exec(`UPDATE app_service SET image_name='registry/team/api' WHERE id=1`)
	if err = save(SaveConfigRequest{Targets: []ConfigTarget{target(2)}, Variables: map[string]*string{"replicas": str("3"), "database": str("prod-db")}}, false); err != nil {
		t.Fatal(err)
	}
	// Only selected port propagates; production replica count and DB remain.
	if err = save(SaveConfigRequest{Targets: []ConfigTarget{target(1), target(2)}, Variables: map[string]*string{"port": str("9000")}}, false); err != nil {
		t.Fatal(err)
	}
	values := map[string]string{}
	for _, v := range get(2).Variables {
		values[v.Key] = v.Value
	}
	if values["port"] != "9000" || values["replicas"] != "3" || values["database"] != "prod-db" {
		t.Fatalf("unexpected overrides: %+v", values)
	}
	// A stale target rejects the whole batch, including the earlier valid target.
	stale := target(2)
	if err = save(SaveConfigRequest{Targets: []ConfigTarget{target(2)}, Variables: map[string]*string{"port": str("9001")}}, false); err != nil {
		t.Fatal(err)
	}
	before := get(1).Token
	if err = save(SaveConfigRequest{Targets: []ConfigTarget{target(1), stale}, Variables: map[string]*string{"port": str("9999")}}, false); err == nil {
		t.Fatal("stale token accepted")
	}
	if get(1).Token != before {
		t.Fatal("partial batch committed")
	}
	if _, err = readServiceConfig(ctx, db, 3, 1, false); err == nil {
		t.Fatal("cross-project config visible")
	}
	// Missing required values invalidate the entire change.
	missing := strings.ReplaceAll(definition, "{{.port}}", "{{.missing}}")
	exec(`INSERT INTO service_revision(id,service_id,revision_number,definition_format,definition,content_hash) VALUES(12,1,3,'k8s_yaml',?,'test12')`, missing)
	if err = save(SaveConfigRequest{Targets: []ConfigTarget{target(1)}, RevisionID: 12}, false); err == nil {
		t.Fatal("missing variable accepted")
	}
	if get(1).RevisionID != 10 {
		t.Fatal("invalid revision persisted")
	}
	// Frozen run follows the environment binding and survives later config edits.
	raw, _ := json.Marshal(map[string]int64{"service_id": 1, "service_revision_id": 11})
	nodes, err := workflow.NewRepository(db).FreezeDeploymentConfigs(ctx, 2, []workflow.Node{{Type: workflow.NodeYAMLDeploy, Config: raw}}, false)
	if err != nil {
		t.Fatal(err)
	}
	var frozen workflow.YAMLDeployConfig
	if err = json.Unmarshal(nodes[0].Config, &frozen); err != nil {
		t.Fatal(err)
	}
	if frozen.ServiceRevisionID != 10 || frozen.FrozenYAML == nil || frozen.FrozenYAML.Variables["port"] != "9001" {
		t.Fatal("run did not capture environment config")
	}
	if err = save(SaveConfigRequest{Targets: []ConfigTarget{target(2)}, Variables: map[string]*string{"port": str("9998")}}, false); err != nil {
		t.Fatal(err)
	}
	rendered, err := workflow.NewDefaultYAMLResolver(workflow.NewSQLYAMLSourceStore(db)).RenderYAML(ctx, workflow.WorkflowRun{ProjectID: 19, EnvironmentID: 2}, frozen)
	if err != nil || !strings.Contains(string(rendered.Manifest), "9001") {
		t.Fatalf("frozen render changed: %s, %v", rendered.Manifest, err)
	}
	// Restoring inheritance removes just that local key.
	if err = save(SaveConfigRequest{Targets: []ConfigTarget{target(2)}, Variables: map[string]*string{"port": nil}}, false); err != nil {
		t.Fatal(err)
	}
	for _, v := range get(2).Variables {
		if v.Key == "port" && v.Origin == "override" {
			t.Fatal("override not removed")
		}
	}
	// Two concurrent clients with the same preview can never both overwrite it.
	shared := target(2)
	var group sync.WaitGroup
	outcomes := make(chan error, 2)
	for _, port := range []string{"7001", "7002"} {
		group.Add(1)
		go func(port string) {
			defer group.Done()
			outcomes <- save(SaveConfigRequest{Targets: []ConfigTarget{shared}, Variables: map[string]*string{"port": str(port)}}, false)
		}(port)
	}
	group.Wait()
	close(outcomes)
	passed := 0
	for err := range outcomes {
		if err == nil {
			passed++
		}
	}
	if passed != 1 {
		t.Fatalf("concurrent successes=%d", passed)
	}
	// Exercise the real HTTP handlers against the disposable schema. Preview
	// returns the resulting config but does not persist it or submit a workflow.
	handler := NewHandler(&Logic{repo: repo, authorizer: authz.New(policyStore{{Resource: "*", Action: "*", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 19, Environment: "*"}})})
	t.Run("native manual first save creates revision and environment atomically", func(t *testing.T) {
		exec(`INSERT INTO app_service(id,project_id,name,code,delivery_type,source_type,current_revision_id) VALUES(80,19,'native','native','k8s_yaml','manual',0)`)
		read := func(eid int64) *ServiceConfig {
			c, e := readServiceConfig(ctx, db, eid, 80, false)
			if e != nil {
				t.Fatal(e)
			}
			return c
		}
		initial := read(1)
		if initial.Definition != "" || initial.Bound || initial.Format != "k8s_yaml" {
			t.Fatal("manual service was not blank")
		}
		native := "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: native-config\ndata:\n  application.yaml: |\n    profile: test\n"
		req := SaveConfigRequest{Targets: []ConfigTarget{{EnvironmentID: 1, Token: initial.Token}}, Definition: &native}
		if e := repo.saveServiceConfig(ctx, 7, 19, 80, &req, true); e != nil {
			t.Fatal(e)
		}
		if read(1).Token != initial.Token || read(1).RevisionID != 0 {
			t.Fatal("preview persisted initial version")
		}
		invalid := "apiVersion: v1\nkind: ConfigMap\nmetadata: ["
		bad := SaveConfigRequest{Targets: req.Targets, Definition: &invalid}
		if e := repo.saveServiceConfig(ctx, 7, 19, 80, &bad, false); e == nil || read(1).RevisionID != 0 {
			t.Fatal("invalid YAML left revision")
		}
		req.Result = nil
		if e := repo.saveServiceConfig(ctx, 7, 19, 80, &req, false); e != nil {
			t.Fatal(e)
		}
		if !read(1).Bound || read(1).Definition != native || read(2).Bound {
			t.Fatal("first save crossed environments")
		}
		// A manually pasted manifest may opt into system placeholders without
		// template parameter metadata or automatic user variables.
		systemYAML := strings.ReplaceAll(definition[strings.Index(definition, "apiVersion:"):], "{{.replicas}}", "1")
		systemYAML = strings.ReplaceAll(systemYAML, "{{.port}}", "8080")
		request := SaveConfigRequest{Targets: []ConfigTarget{{EnvironmentID: 1, Token: read(1).Token}}, Definition: &systemYAML}
		if e := repo.saveServiceConfig(ctx, 7, 19, 80, &request, false); e != nil {
			t.Fatal(e)
		}
		if len(request.Result) != 1 || strings.Contains(request.Result[0].Manifest, "{{.") || !strings.Contains(request.Result[0].Manifest, "name: native") || len(read(1).Variables) != 0 {
			t.Fatal("manual system placeholders did not resolve independently")
		}
	})
	t.Run("configured template creation previews rolls back and validates target", func(t *testing.T) {
		call := func(name, yaml string, preview bool) struct {
			Code int `json:"code"`
			Data struct {
				ServiceID int64            `json:"service_id"`
				Saved     bool             `json:"saved"`
				Configs   []*ServiceConfig `json:"configs"`
			} `json:"data"`
		} {
			body, _ := json.Marshal(ConfiguredServiceRequest{Name: name, Code: name, Definition: yaml, Variables: map[string]*string{"port": str("9099"), "replicas": str("2")}})
			url := "/environments/1/services"
			if preview {
				url += "?preview=1"
			}
			r := httptest.NewRequest("POST", url, strings.NewReader(string(body)))
			r.SetPathValue("id", "1")
			r = r.WithContext(context.WithValue(ctx, middleware.ContextKeyUserID, int64(7)))
			w := httptest.NewRecorder()
			handler.CreateConfiguredService(w, r)
			var result struct {
				Code int `json:"code"`
				Data struct {
					ServiceID int64            `json:"service_id"`
					Saved     bool             `json:"saved"`
					Configs   []*ServiceConfig `json:"configs"`
				} `json:"data"`
			}
			if e := json.Unmarshal(w.Body.Bytes(), &result); e != nil {
				t.Fatal(e)
			}
			if result.Code != 0 {
				t.Log(w.Body.String())
			}
			return result
		}
		count := func() int {
			var n int
			if e := db.QueryRow(`SELECT COUNT(*) FROM app_service WHERE code='created-template'`).Scan(&n); e != nil {
				t.Fatal(e)
			}
			return n
		}
		before := get(2).Token
		preview := call("created-template", definition, true)
		if preview.Code != 0 || preview.Data.Saved || count() != 0 || len(preview.Data.Configs) != 1 || !strings.Contains(preview.Data.Configs[0].Manifest, "containerPort: 9099") {
			t.Fatalf("atomic preview failed: %+v", preview)
		}
		wrong := strings.Replace(definition, "  name: {{.Service}}", "  name: {{.Service}}\n  namespace: api-prod", 1)
		if got := call("created-template", wrong, false); got.Code == 0 || count() != 0 {
			t.Fatalf("invalid target left service: %+v", got)
		}
		got := call("created-template", definition, false)
		if got.Code != 0 || !got.Data.Saved || count() != 1 || get(2).Token != before {
			t.Fatalf("atomic create failed: %+v", got)
		}
		if call("created-template", definition, false).Code == 0 || count() != 1 {
			t.Fatal("duplicate service created")
		}
	})
	t.Run("configured preview renders all optional assets as native YAML without saving", func(t *testing.T) {
		assets := serviceassets.Empty()
		assets.Routes = []serviceassets.Route{{Name: "api-route", Host: "api.test.example", Path: "/", Service: "preview-assets", Port: 9099}}
		assets.Volumes = []serviceassets.Volume{{Name: "data", Capacity: "1Gi", AccessMode: "ReadWriteOnce", MountPath: "/data"}}
		assets.Files = []serviceassets.File{{Name: "config", Filename: "application.yaml", Content: "profile: test", MountPath: "/app/application.yaml"}}
		assets.Secrets = []serviceassets.Secret{{Name: "auth", ExistingSecret: "test-auth", Key: "password", EnvName: "APP_PASSWORD"}}
		body, _ := json.Marshal(ConfiguredServiceRequest{
			Name: "preview-assets", Code: "preview-assets",
			Definition: definition + "\n---\napiVersion: v1\nkind: Service\nmetadata:\n  name: {{.Service}}\nspec:\n  ports:\n  - port: {{.port}}\n    targetPort: {{.port}}\n",
			Variables:  map[string]*string{"port": str("9099"), "replicas": str("2"), "healthCheckType": str("tcp")}, Assets: &assets,
		})
		beforeProd := get(2).Token
		r := httptest.NewRequest("POST", "/environments/1/services?preview=1", strings.NewReader(string(body)))
		r.SetPathValue("id", "1")
		r = r.WithContext(context.WithValue(ctx, middleware.ContextKeyUserID, int64(7)))
		w := httptest.NewRecorder()
		handler.CreateConfiguredService(w, r)
		var result struct {
			Code int `json:"code"`
			Data struct {
				Saved   bool             `json:"saved"`
				Configs []*ServiceConfig `json:"configs"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || result.Code != 0 || result.Data.Saved || len(result.Data.Configs) != 1 {
			t.Fatalf("optional assets preview failed: %s", w.Body.String())
		}
		manifest := result.Data.Configs[0].Manifest
		if strings.Contains(manifest, "{{.") || strings.Contains(manifest, "AxiOps-Variables") {
			t.Fatal("preview returned template source instead of native YAML")
		}
		objects, err := workflow.ValidateYAMLManifest([]byte(manifest), "api-test")
		if err != nil {
			t.Fatal(err)
		}
		kinds := map[string]int{}
		for _, object := range objects {
			kinds[object.GetKind()]++
		}
		for _, kind := range []string{"Deployment", "Service", "Ingress", "PersistentVolumeClaim", "ConfigMap"} {
			if kinds[kind] != 1 {
				t.Fatalf("missing or duplicate %s in preview", kind)
			}
		}
		for _, fragment := range []string{"api.test.example", "profile: test", "claimName:", "secretKeyRef:", "name: test-auth", "tcpSocket:", "containerPort: 9099", "replicas: 2"} {
			if !strings.Contains(manifest, fragment) {
				t.Fatalf("missing preview detail %q", fragment)
			}
		}
		if kinds["Secret"] != 0 {
			t.Fatal("existing Secret references must not create a fake Secret or expose its body")
		}
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM app_service WHERE code='preview-assets'`).Scan(&count); err != nil || count != 0 || get(2).Token != beforeProd {
			t.Fatal("preview persisted a service or changed production")
		}
	})
	before = get(2).Token
	payload, _ := json.Marshal(SaveConfigRequest{Targets: []ConfigTarget{target(2)}, Variables: map[string]*string{"port": str("8001")}})
	request := httptest.NewRequest("POST", "/services/1/environment-config?preview=1", strings.NewReader(string(payload)))
	request.SetPathValue("service_id", "1")
	request = request.WithContext(context.WithValue(request.Context(), middleware.ContextKeyUserID, int64(7)))
	w := httptest.NewRecorder()
	handler.SaveServiceConfig(w, request)
	var response struct {
		Code int `json:"code"`
		Data struct {
			Saved     bool            `json:"saved"`
			Validated bool            `json:"validated"`
			Configs   []ServiceConfig `json:"configs"`
		} `json:"data"`
	}
	if json.Unmarshal(w.Body.Bytes(), &response) != nil || response.Code != 0 || response.Data.Saved || !response.Data.Validated || len(response.Data.Configs) != 1 {
		t.Fatalf("preview response: %s", w.Body.String())
	}
	if get(2).Token != before {
		t.Fatal("HTTP preview persisted configuration")
	}
	request = httptest.NewRequest("GET", "/environments/2/services/1/config", nil)
	request.SetPathValue("id", "2")
	request.SetPathValue("service_id", "1")
	request = request.WithContext(context.WithValue(request.Context(), middleware.ContextKeyUserID, int64(7)))
	w = httptest.NewRecorder()
	handler.GetServiceConfig(w, request)
	var read struct {
		Code int           `json:"code"`
		Data ServiceConfig `json:"data"`
	}
	if json.Unmarshal(w.Body.Bytes(), &read) != nil || read.Code != 0 || read.Data.Token != before {
		t.Fatalf("read response: %s", w.Body.String())
	}
	// Match the frontend's confirmation contract against a real committed save.
	request = httptest.NewRequest("POST", "/services/1/environment-config", strings.NewReader(string(payload)))
	request.SetPathValue("service_id", "1")
	request = request.WithContext(context.WithValue(request.Context(), middleware.ContextKeyUserID, int64(7)))
	w = httptest.NewRecorder()
	handler.SaveServiceConfig(w, request)
	if json.Unmarshal(w.Body.Bytes(), &response) != nil || response.Code != 0 || !response.Data.Saved || !response.Data.Validated || len(response.Data.Configs) != 1 {
		t.Fatalf("save confirmation response: %s", w.Body.String())
	}
	saved := response.Data.Configs[0]
	if saved.EnvironmentID != 2 || saved.ServiceID != 1 || !saved.Bound || saved.RevisionID <= 0 || len(saved.Token) != 64 || saved.Token != get(2).Token || saved.Token == before {
		t.Fatal("save confirmation does not describe the committed target")
	}
	for _, variable := range get(2).Variables {
		if variable.Key == "port" && variable.Value != "8001" {
			t.Fatal("HTTP save did not persist the requested value")
		}
	}
	// Save shared variables through the application logic, then verify the same
	// resolver used by deployment sees inheritance, overrides and frozen inputs.
	exec(`INSERT INTO variable_set(project_id,environment_id,service_id,name,scope,revision,created_by) VALUES(19,1,0,'default','environment',1,7)`)
	global, err := repo.FindVariableSetByEnvironment(ctx, 1)
	if err != nil || global == nil {
		t.Fatal("shared variable set missing")
	}
	global, code := handler.logic.UpdateVariableSet(ctx, 7, global.ID, &UpdateVariableSetRequest{Revision: global.Revision, Items: []VariableItemInput{{VariableKey: "port", Value: "7777"}}})
	if code != 0 || global == nil {
		t.Fatalf("shared variable save code=%d", code)
	}
	if err = save(SaveConfigRequest{Targets: []ConfigTarget{target(1)}, Variables: map[string]*string{"port": nil}}, false); err != nil {
		t.Fatal(err)
	}
	store := workflow.NewSQLYAMLSourceStore(db)
	sharedSource, err := store.LoadYAMLSource(ctx, 1, 1, get(1).RevisionID)
	if err != nil || sharedSource.Variables["port"] != "7777" {
		t.Fatal("shared variable not inherited by deployment")
	}
	other, err := store.LoadYAMLSource(ctx, 2, 1, get(2).RevisionID)
	if err != nil || other.Variables["port"] != "8001" {
		t.Fatal("shared change crossed environment boundary")
	}
	frozenNodes, err := workflow.NewRepository(db).FreezeDeploymentConfigs(ctx, 1, []workflow.Node{{Type: workflow.NodeYAMLDeploy, Config: raw}}, false)
	if err != nil {
		t.Fatal(err)
	}
	var captured workflow.YAMLDeployConfig
	if err = json.Unmarshal(frozenNodes[0].Config, &captured); err != nil {
		t.Fatal(err)
	}
	oldToken := get(1).Token
	global, code = handler.logic.UpdateVariableSet(ctx, 7, global.ID, &UpdateVariableSetRequest{Revision: global.Revision, Items: []VariableItemInput{{VariableKey: "port", Value: "8888"}}})
	if code != 0 || get(1).Token == oldToken {
		t.Fatal("shared change did not invalidate configuration token")
	}
	resolver := workflow.NewDefaultYAMLResolver(store)
	currentManifest, err := resolver.RenderYAML(ctx, workflow.WorkflowRun{ProjectID: 19, EnvironmentID: 1}, workflow.YAMLDeployConfig{ServiceID: 1, ServiceRevisionID: get(1).RevisionID})
	if err != nil || !strings.Contains(string(currentManifest.Manifest), "containerPort: 8888") {
		t.Fatal("next deployment does not receive shared change")
	}
	frozenManifest, err := resolver.RenderYAML(ctx, workflow.WorkflowRun{ProjectID: 19, EnvironmentID: 1}, captured)
	if err != nil || !strings.Contains(string(frozenManifest.Manifest), "containerPort: 7777") {
		t.Fatal("shared update changed an already submitted run")
	}
	if err = save(SaveConfigRequest{Targets: []ConfigTarget{target(1)}, Variables: map[string]*string{"port": str("9999")}}, false); err != nil {
		t.Fatal(err)
	}
	overridden, err := store.LoadYAMLSource(ctx, 1, 1, get(1).RevisionID)
	if err != nil || overridden.Variables["port"] != "9999" {
		t.Fatal("service override did not win over shared variable")
	}
	t.Run("runtime resources HTTP persistence and frozen environment isolation", func(t *testing.T) {
		testRuntimeResourcesHTTP(t, db, handler)
	})
	t.Run("whole YAML and resources save atomically only in selected environment", func(t *testing.T) {
		before, prod := get(1), get(2)
		definition := strings.Replace(before.Definition, "        ports:", "        env:\n        - name: LOG_LEVEL\n          value: debug\n        ports:", 1) + "\n---\n# AxiOps-File: configmap.yaml\napiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: api-config\ndata:\n  mode: test\n"
		request := SaveConfigRequest{Targets: []ConfigTarget{target(1)}, Definition: &definition, Variables: map[string]*string{"port": str("9080")}, Assets: &serviceassets.Settings{Resources: &serviceassets.RuntimeResources{MemoryLimit: "2Gi"}}}
		if err := repo.saveServiceConfig(ctx, 7, 19, 1, &request, true); err != nil {
			t.Fatal(err)
		}
		if get(1).Token != before.Token || request.Result[0].Manifest == "" || !strings.Contains(request.Result[0].Manifest, "2Gi") || !strings.Contains(request.Result[0].Manifest, "9080") {
			t.Fatal("preview did not include every draft input or persisted")
		}
		request.Result = nil
		if err := repo.saveServiceConfig(ctx, 7, 19, 1, &request, false); err != nil {
			t.Fatal(err)
		}
		if get(1).EnvironmentYAMLRevisionID == 0 || get(1).Definition != definition || get(2).Token != prod.Token {
			t.Fatal("environment YAML not isolated")
		}
		var public int64
		if err := db.QueryRow(`SELECT current_revision_id FROM app_service WHERE id=1`).Scan(&public); err != nil || public != 11 {
			t.Fatal("public definition changed")
		}
		frozen, e := workflow.NewRepository(db).FreezeDeploymentConfigs(ctx, 1, []workflow.Node{{Type: workflow.NodeYAMLDeploy, Config: json.RawMessage(`{"service_id":1,"service_revision_id":10}`)}}, false)
		if e != nil {
			t.Fatal(e)
		}
		var cfg workflow.YAMLDeployConfig
		json.Unmarshal(frozen[0].Config, &cfg)
		if !strings.Contains(string(cfg.FrozenYAML.Manifest), "LOG_LEVEL") {
			t.Fatal("deployment lost environment YAML")
		}
		good := get(1)
		bad := "apiVersion: v1\nkind: Service\nmetadata:\n  name: api\n  namespace: wrong-environment"
		if e := save(SaveConfigRequest{Targets: []ConfigTarget{target(1)}, Definition: &bad, Variables: map[string]*string{"port": str("9090")}}, false); e == nil || get(1).Token != good.Token {
			t.Fatal("invalid definition did not atomically roll back")
		}
		if e := save(SaveConfigRequest{Targets: []ConfigTarget{target(1)}, ResetDefinition: true}, false); e != nil {
			t.Fatal(e)
		}
		if get(1).EnvironmentYAMLRevisionID != 0 || get(1).Definition != before.Definition || get(2).Token != prod.Token {
			t.Fatal("restore did not preserve other environment")
		}
		if !strings.Contains(string(cfg.FrozenYAML.Manifest), "LOG_LEVEL") {
			t.Fatal("already queued snapshot changed")
		}
	})
	t.Run("source guarded YAML synchronization preserves production inputs and rejects stale previews", func(t *testing.T) {
		beforeSource, beforeTarget := get(1), get(2)
		beforeAssets, _ := json.Marshal(beforeTarget.Assets)
		yaml := beforeSource.Definition + "\n---\napiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: promoted-config\ndata:\n  mode: production\n"
		request := SaveConfigRequest{Source: &ConfigTarget{EnvironmentID: 1, Token: beforeSource.Token}, Targets: []ConfigTarget{target(2)}, Definition: &yaml, Variables: map[string]*string{"replicas": str("4")}}
		if !validConfigRequest(&request) {
			t.Fatal("guarded YAML request rejected")
		}
		if e := repo.saveServiceConfig(ctx, 7, 19, 1, &request, true); e != nil {
			t.Fatal(e)
		}
		if get(2).Token != beforeTarget.Token || len(request.Result) != 1 || !strings.Contains(request.Result[0].Manifest, "6Gi") || !strings.Contains(request.Result[0].Manifest, "containerPort: 8001") {
			t.Fatal("preview did not preserve target resource and variable inputs")
		}
		if e := save(SaveConfigRequest{Targets: []ConfigTarget{target(1)}, Variables: map[string]*string{"port": str("9099")}}, false); e != nil {
			t.Fatal(e)
		}
		request.Result = nil
		if e := repo.saveServiceConfig(ctx, 7, 19, 1, &request, false); e == nil || get(2).Token != beforeTarget.Token {
			t.Fatal("stale source modified production")
		}
		request.Source.Token = get(1).Token
		request.Result = nil
		wrongNamespace := strings.Replace(yaml, "  name: promoted-config", "  name: promoted-config\n  namespace: api-test", 1)
		request.Definition = &wrongNamespace
		if e := repo.saveServiceConfig(ctx, 7, 19, 1, &request, false); e == nil || get(2).Token != beforeTarget.Token {
			t.Fatal("source namespace crossed into production")
		}
		request.Definition = &yaml
		request.Result = nil
		stableSource := get(1).Token
		if e := repo.saveServiceConfig(ctx, 7, 19, 1, &request, false); e != nil {
			t.Fatal(e)
		}
		after := get(2)
		afterAssets, _ := json.Marshal(after.Assets)
		if string(afterAssets) != string(beforeAssets) || after.Definition != yaml || get(1).Token != stableSource {
			t.Fatal("sync changed target assets or source config")
		}
	})
	t.Run("first join retains target environment defaults without copying test config", func(t *testing.T) {
		exec(`INSERT INTO project_environment(id,project_id,name,code,environment_type,cluster_id,namespace,binding_status) VALUES(4,19,'Stage','stage','k8s_namespace',2,'api-stage','validated')`)
		exec(`INSERT INTO variable_set(project_id,environment_id,service_id,name,scope,revision,created_by) VALUES(19,4,0,'default','environment',1,7)`)
		exec(`INSERT INTO variable_item(variable_set_id,variable_key,value_text,credential_id,is_secret,created_by) SELECT id,'port','7779',0,0,7 FROM variable_set WHERE environment_id=4`)
		before := get(4)
		request := SaveConfigRequest{Targets: []ConfigTarget{target(4)}, RevisionID: 11}
		if e := repo.saveServiceConfig(ctx, 7, 19, 1, &request, true); e != nil {
			t.Fatal(e)
		}
		if get(4).Bound || get(4).Token != before.Token || !strings.Contains(request.Result[0].Manifest, "containerPort: 7779") {
			t.Fatal("join preview wrote a binding or used test variables")
		}
		request.Result = nil
		if e := repo.saveServiceConfig(ctx, 7, 19, 1, &request, false); e != nil {
			t.Fatal(e)
		}
		if !get(4).Bound || get(4).RevisionID != 11 {
			t.Fatal("join not persisted")
		}
	})
	t.Run("route and generated probes follow port edits atomically", func(t *testing.T) { testRoutePortUpdate(t, db) })
	t.Run("Helm native values isolation preview and atomic save", func(t *testing.T) { testHelmNativeConfig(t, db) })
}

func testRoutePortUpdate(t *testing.T, db *sql.DB) {
	ctx := context.Background()
	exec := func(query string, args ...interface{}) {
		t.Helper()
		if _, e := db.Exec(query, args...); e != nil {
			t.Fatal(e)
		}
	}
	definition := `# AxiOps-Variables: [{"key":"port","value":"8080"},{"key":"healthCheckType","value":"tcp"},{"key":"healthPort","value":"0"}]
apiVersion: apps/v1
kind: Deployment
metadata: {name: port-api}
spec:
  selector:
    matchLabels: {app: port-api}
  template:
    metadata:
      labels: {app: port-api}
    spec:
      containers:
      - name: port-api
        image: registry/api:v1
        ports: [{name: http, containerPort: {{.port}}}]
---
apiVersion: v1
kind: Service
metadata: {name: port-api}
spec:
  selector: {app: port-api}
  ports: [{name: http, port: {{.port}}, targetPort: http}]
`
	exec(`INSERT INTO app_service(id,project_id,name,code,delivery_type,source_type,source_ref,image_name,current_revision_id) VALUES(901,19,'port-api','port-api','k8s_yaml','manual','{}','registry/api',901)`)
	exec(`INSERT INTO service_revision(id,service_id,revision_number,definition_format,definition,content_hash) VALUES(901,901,1,'k8s_yaml',?,'port-test')`, definition)
	assets := serviceassets.Empty()
	assets.Routes = []serviceassets.Route{{Name: "route", Host: "api.example.com", Path: "/", Service: "port-api", Port: 8080}}
	raw, _ := json.Marshal(assets)
	for _, eid := range []int64{1, 2} {
		exec(`INSERT INTO environment_service_override(project_environment_id,service_id,service_revision_id) VALUES(?,901,901)`, eid)
		exec(`INSERT INTO environment_resource(environment_id,name,kind,manifest) VALUES(?,?,?,?)`, eid, serviceassets.RowName(901), serviceassets.ResourceKind, string(raw))
		exec(`INSERT INTO variable_set(project_id,environment_id,service_id,name,scope,created_by) VALUES(19,?,901,'default','environment',7)`, eid)
		exec(`INSERT INTO variable_item(variable_set_id,variable_key,value_text) SELECT id,'port','8080' FROM variable_set WHERE environment_id=? AND service_id=901`, eid)
	}
	read := func(eid int64) *ServiceConfig {
		t.Helper()
		c, e := readServiceConfig(ctx, db, eid, 901, false)
		if e != nil {
			t.Fatal(e)
		}
		return c
	}
	port := "28080"
	request := SaveConfigRequest{Targets: []ConfigTarget{{EnvironmentID: 1, Token: read(1).Token}}, Variables: map[string]*string{"port": &port}, Assets: &assets}
	repo := NewRepository(db)
	before, other := read(1).Token, read(2).Token
	if e := repo.saveServiceConfig(ctx, 7, 19, 901, &request, true); e != nil {
		t.Fatal(e)
	}
	preview := request.Result[0]
	if len(preview.Adjustments) != 1 || preview.Assets.Routes[0].Port != 28080 || !strings.Contains(preview.Manifest, "port: 28080") || !strings.Contains(preview.Manifest, "number: 28080") {
		t.Fatalf("port preview incomplete: %+v", preview)
	}
	if read(1).Token != before || read(2).Token != other || assets.Routes[0].Port != 8080 {
		t.Fatal("preview persisted or mutated request")
	}
	request.Result = nil
	if e := repo.saveServiceConfig(ctx, 7, 19, 901, &request, false); e != nil {
		t.Fatal(e)
	}
	if read(1).Assets.Routes[0].Port != 28080 || read(2).Token != other {
		t.Fatal("port save crossed environments")
	}
	// A Service may expose 80 while the container and probes remain on 28080.
	definition = strings.Replace(definition, "name: http, port: {{.port}}", "name: http, port: 80", 1)
	request = SaveConfigRequest{Targets: []ConfigTarget{{EnvironmentID: 1, Token: read(1).Token}}, Definition: &definition}
	if e := repo.saveServiceConfig(ctx, 7, 19, 901, &request, false); e != nil {
		t.Fatal(e)
	}
	if read(1).Assets.Routes[0].Port != 80 || !strings.Contains(request.Result[0].Manifest, "number: 80") || !strings.Contains(request.Result[0].Manifest, "port: 28080") {
		t.Fatal("Service and container port confused")
	}
	invalid := read(1).Assets
	invalid.Routes[0].Port = 1234
	request = SaveConfigRequest{Targets: []ConfigTarget{{EnvironmentID: 1, Token: read(1).Token}}, Assets: &invalid}
	before = read(1).Token
	if e := repo.saveServiceConfig(ctx, 7, 19, 901, &request, false); e == nil || read(1).Token != before {
		t.Fatal("explicit invalid backend accepted or partially saved")
	}
}

func testRuntimeResourcesHTTP(t *testing.T, db *sql.DB, handler *Handler) {
	ctx := context.Background()
	read := func(eid int64) *ServiceConfig {
		t.Helper()
		c, e := readServiceConfig(ctx, db, eid, 1, false)
		if e != nil {
			t.Fatal(e)
		}
		return c
	}
	type result struct {
		Code int `json:"code"`
		Data struct {
			Saved     bool          `json:"saved"`
			Validated bool          `json:"validated"`
			Manifest  string        `json:"manifest"`
			Config    ServiceConfig `json:"config"`
		} `json:"data"`
	}
	save := func(eid int64, token string, resources *serviceassets.RuntimeResources, preview bool) result {
		t.Helper()
		assets := serviceassets.Empty()
		assets.Resources = resources
		body, _ := json.Marshal(map[string]interface{}{"token": token, "assets": assets})
		suffix := ""
		if preview {
			suffix = "?preview=1"
		}
		request := httptest.NewRequest("PUT", "/environments/"+fmt.Sprint(eid)+"/services/1/assets"+suffix, strings.NewReader(string(body)))
		request.SetPathValue("id", fmt.Sprint(eid))
		request.SetPathValue("service_id", "1")
		request = request.WithContext(context.WithValue(request.Context(), middleware.ContextKeyUserID, int64(7)))
		w := httptest.NewRecorder()
		handler.SaveServiceAssets(w, request)
		var got result
		if e := json.Unmarshal(w.Body.Bytes(), &got); e != nil {
			t.Fatal(e)
		}
		return got
	}
	for _, item := range []struct {
		eid    int64
		memory string
	}{{1, "1Gi"}, {2, "4Gi"}} {
		before := read(item.eid).Token
		resources := &serviceassets.RuntimeResources{CPURequest: "250m", CPULimit: "1", MemoryRequest: "512Mi", MemoryLimit: item.memory}
		got := save(item.eid, before, resources, true)
		if got.Code != 0 || got.Data.Saved || !got.Data.Validated || !strings.Contains(got.Data.Manifest, "memory: "+item.memory) || read(item.eid).Token != before {
			t.Fatalf("preview failed: %+v", got)
		}
		got = save(item.eid, before, resources, false)
		if got.Code != 0 || !got.Data.Saved || read(item.eid).Assets.Resources.MemoryLimit != item.memory {
			t.Fatalf("save failed: %+v", got)
		}
		if save(item.eid, before, resources, false).Code == 0 {
			t.Fatal("stale resource token overwrote configuration")
		}
	}
	if read(1).Assets.Resources.MemoryLimit != "1Gi" || read(2).Assets.Resources.MemoryLimit != "4Gi" {
		t.Fatal("resources crossed environments")
	}
	nodes, e := workflow.NewRepository(db).FreezeDeploymentConfigs(ctx, 2, []workflow.Node{{Type: workflow.NodeYAMLDeploy, Config: json.RawMessage(`{"service_id":1,"service_revision_id":11}`)}}, false)
	if e != nil {
		t.Fatal(e)
	}
	var frozen workflow.YAMLDeployConfig
	if e = json.Unmarshal(nodes[0].Config, &frozen); e != nil {
		t.Fatal(e)
	}
	if got := save(2, read(2).Token, &serviceassets.RuntimeResources{MemoryLimit: "6Gi"}, false); got.Code != 0 {
		t.Fatalf("update failed: %+v", got)
	}
	rendered, e := workflow.NewDefaultYAMLResolver(workflow.NewSQLYAMLSourceStore(db)).RenderYAML(ctx, workflow.WorkflowRun{ProjectID: 19, EnvironmentID: 2}, frozen)
	if e != nil || !strings.Contains(string(rendered.Manifest), "memory: 4Gi") || rendered.Namespace != "api-prod" {
		t.Fatal("queued deployment resource snapshot changed")
	}
	for _, invalid := range []*serviceassets.RuntimeResources{{MemoryLimit: "1g"}, {MemoryLimit: "1"}, {MemoryRequest: "2Gi", MemoryLimit: "1Gi"}, {CPURequest: "2", CPULimit: "1"}} {
		token := read(1).Token
		if save(1, token, invalid, false).Code == 0 || read(1).Token != token {
			t.Fatal("invalid resources were committed")
		}
	}
	if got := save(1, read(1).Token, nil, false); got.Code != 0 || read(1).Assets.Resources != nil || read(2).Assets.Resources.MemoryLimit != "6Gi" {
		t.Fatal("restoring inheritance changed other environment")
	}
}
