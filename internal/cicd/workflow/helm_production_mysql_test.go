package workflow

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ChenJustin666/axiops/internal/cicd/helmconfig"
	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"helm.sh/helm/v3/pkg/release"
)

func testMySQLHelmProduction(t *testing.T, db *sql.DB, source string) {
	ctx := actorContext(7, "helm-release-operator")
	exec := func(q string, args ...interface{}) {
		t.Helper()
		if _, err := db.Exec(q, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec("CREATE TABLE IF NOT EXISTS credential LIKE " + source + ".credential")
	exec(`INSERT INTO project(id,name,code,delivery_type,delivery_status) VALUES(39,'Helm Delivery','helm-delivery','helm','active')`)
	exec(`INSERT INTO app_service(id,project_id,name,code,delivery_type,source_type,source_ref,image_name,current_revision_id) VALUES(31,39,'helm-api','helm-api','helm','manual','{}','registry.example/team/helm-api',311)`)
	definition := testHelmDefinition("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: {{ .Release.Name }}\nspec:\n  replicas: {{ .Values.replicas }}\n  template:\n    spec:\n      containers:\n      - name: api\n        image: \"{{ .Values.image.repository }}:{{ .Values.image.tag }}\"\n        env:\n        - name: HOST\n          value: {{ .Values.host | quote }}\n")
	exec(`INSERT INTO service_revision(id,service_id,revision_number,definition_format,definition,content_hash) VALUES(311,31,1,'helm_chart',?,'helm311')`, definition)
	exec(`INSERT INTO project_environment(id,project_id,name,code,environment_type,cluster_id,namespace,binding_status,is_production) VALUES(41,39,'Helm QA','qa','k8s_namespace',2,'helm-qa','validated',0),(42,39,'Helm Prod','prod','k8s_namespace',2,'helm-prod','validated',1)`)
	exec(`INSERT INTO environment_service_override(project_environment_id,service_id,service_revision_id) VALUES(41,31,311),(42,31,311)`)
	saveSettings := func(eid int64, values string) {
		settings := helmconfig.Default()
		settings.Values = values
		raw, _ := json.Marshal(settings)
		exec(`INSERT INTO environment_resource(environment_id,name,kind,manifest,revision) VALUES(?,?,?,?,1) ON DUPLICATE KEY UPDATE manifest=VALUES(manifest),revision=revision+1`, eid, helmconfig.RowName(31), helmconfig.Kind, raw)
	}
	saveSettings(41, "replicas: 1\nhost: qa-only.example.com\n")
	saveSettings(42, "replicas: 4\nhost: prod-only.example.com\n")
	digest := "sha256:" + strings.Repeat("c", 64)
	exec(`INSERT INTO build_run(id,service_id,project_id,build_number,status,provider,image_url,image_digest,git_commit,git_branch) VALUES(342,31,39,1,'success','kube_job','registry.example/team/helm-api:qa-1',?,'helm-commit','qa')`, digest)
	repo := NewRepository(db)
	w := &Workflow{ProjectID: 39, TargetEnvironmentID: 42, Name: "Helm Prod", Code: "helm-prod", DeliveryType: DeliveryTypeHelm, Nodes: []Node{{Key: "build", Type: NodeBuild, Config: mustJSON(map[string]interface{}{"service_id": 31})}, {Key: "approval", Type: NodeApproval}, {Key: "deploy", Type: NodeHelmDeploy, DependsOn: []string{"build", "approval"}, Config: mustJSON(HelmDeployConfig{ServiceID: 31, ServiceRevisionID: 311, ReleaseName: "helm-api"})}}}
	wid, err := repo.Create(ctx, w)
	if err != nil {
		t.Fatal(err)
	}
	l := &Logic{repo: repo, authorizer: authz.New(policyStore{{Resource: "*", Action: "*", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 39, Environment: "*"}}), helmResolver: NewDefaultHelmResolver(NewSQLHelmSourceStore(db), nil)}
	options, err := l.ProductionOptions(ctx, 7, wid)
	if err != nil || len(options.Services) != 1 || len(options.Services[0].Images) != 1 {
		t.Fatalf("Helm production images missing: %+v %v", options, err)
	}
	p, err := l.ProductionImagePreview(ctx, 7, wid, 342)
	if err != nil {
		t.Fatal(err)
	}
	if p.BlockedReason != "" || !strings.Contains(p.TargetManifest, "prod-only.example.com") || strings.Contains(p.TargetManifest, "qa-only.example.com") || !strings.Contains(p.TargetManifest, digest) {
		t.Fatalf("wrong production render: %+v", p)
	}
	request := ProductionSubmission{RequestKey: "helm-production-selected", Picks: []ProductionPick{{BuildID: 342, ExpectedToken: p.Token}}}
	saveSettings(42, "replicas: 5\nhost: new-prod.example.com\n")
	if _, err := l.SubmitProduction(ctx, 7, wid, request); err == nil {
		t.Fatal("stale values preview accepted")
	}
	p, err = l.ProductionImagePreview(ctx, 7, wid, 342)
	if err != nil {
		t.Fatal(err)
	}
	request.Picks[0].ExpectedToken = p.Token
	id, err := l.SubmitProduction(ctx, 7, wid, request)
	if err != nil {
		t.Fatal(err)
	}
	defer exec("UPDATE workflow_run SET status='failed' WHERE id=?", id)
	if retry, err := l.SubmitProduction(ctx, 7, wid, request); err != nil || retry != id {
		t.Fatal("Helm production retry duplicated submission", err)
	}
	run, err := repo.FindRun(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	var task TaskRun
	for _, item := range run.Tasks {
		if item.NodeType == NodeBuild {
			t.Fatal("image reuse kept build task")
		}
		if item.NodeType == NodeHelmDeploy {
			task = item
		}
	}
	cfg, err := helmDeployConfig(task.NodeSnapshot)
	if err != nil {
		t.Fatal(err)
	}
	target, err := NewDefaultHelmResolver(NewSQLHelmSourceStore(db), nil).RenderHelm(ctx, *run, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if target.Values["host"] != "new-prod.example.com" || target.ImageRef != "registry.example/team/helm-api:qa-1@"+digest {
		t.Fatal("target configuration/image changed")
	}
	store := NewSQLDeploymentSnapshotStore(db)
	t.Run("release ownership also resolves audited rollback snapshots", func(t *testing.T) {
		provider := &OfficialHelmProvider{db: db}
		labels, err := provider.releaseOwnershipLabels(ctx, run.ID, task.NodeKey)
		if err != nil || labels["axiops-service-id"] != "31" || labels["axiops-environment-id"] != "42" {
			t.Fatal("deployment owner not resolved", err)
		}
		snapshot, err := NewHelmDeploymentSnapshot(*run, task, cfg, target, []byte(p.TargetManifest))
		if err != nil {
			t.Fatal(err)
		}
		if err := store.Prepare(ctx, snapshot); err != nil {
			t.Fatal(err)
		}
		defer exec(`DELETE FROM deployment_snapshot WHERE workflow_run_id=? AND node_key=?`, run.ID, task.NodeKey)
		if err := store.MarkHelmApplied(ctx, run.ID, task.NodeKey, target.ReleaseName, 2); err != nil {
			t.Fatal(err)
		}
		var sid int64
		if err := db.QueryRow(`SELECT id FROM deployment_snapshot WHERE workflow_run_id=? AND node_key=?`, run.ID, task.NodeKey).Scan(&sid); err != nil {
			t.Fatal(err)
		}
		legacy := &release.Release{Name: target.ReleaseName, Version: 2}
		if err := provider.validateReleaseOwnership(ctx, legacy, target.ClusterID, target.Namespace, labels); err != nil {
			t.Fatal("registered legacy revision rejected", err)
		}
		legacy.Labels = map[string]string{"axiops-service-id": "another-service"}
		if err := provider.validateReleaseOwnership(ctx, legacy, target.ClusterID, target.Namespace, labels); err == nil {
			t.Fatal("foreign Release accepted")
		}
		rollbackNode := mustJSON(Node{Key: task.NodeKey, Type: NodeHelmRollback, Config: mustJSON(HelmRollbackConfig{SnapshotID: sid})})
		exec(`UPDATE workflow_task_run SET node_type=?,node_snapshot=? WHERE id=?`, NodeHelmRollback, rollbackNode, task.ID)
		defer exec(`UPDATE workflow_task_run SET node_type=?,node_snapshot=? WHERE id=?`, NodeHelmDeploy, task.NodeSnapshot, task.ID)
		actual, err := provider.releaseOwnershipLabels(ctx, run.ID, task.NodeKey)
		if err != nil || !helmOwnerLabelsMatch(actual, labels) {
			t.Fatal("rollback owner not resolved from its snapshot", err)
		}
		exec(`UPDATE deployment_snapshot SET environment_id=41 WHERE id=?`, sid)
		if _, err := provider.releaseOwnershipLabels(ctx, run.ID, task.NodeKey); err == nil {
			t.Fatal("rollback adopted snapshot from another environment")
		}
	})
	if err := store.ValidateHelmProductionExecution(ctx, *run, task, cfg, target); err != nil {
		t.Fatal(err)
	}
	exec(`UPDATE build_run SET image_digest=? WHERE id=342`, "sha256:"+strings.Repeat("d", 64))
	if err := store.ValidateHelmProductionExecution(ctx, *run, task, cfg, target); err == nil {
		t.Fatal("worker accepted altered artifact")
	}
	exec(`UPDATE build_run SET image_digest=? WHERE id=342`, digest)
	exec(`UPDATE project_environment SET namespace='wrong-prod' WHERE id=42`)
	if _, err := NewDefaultHelmResolver(NewSQLHelmSourceStore(db), nil).RenderHelm(ctx, *run, cfg); err == nil {
		t.Fatal("queued Helm release followed changed namespace")
	}
	exec(`UPDATE project_environment SET namespace='helm-prod' WHERE id=42`)
	exec("UPDATE workflow_run SET status='failed' WHERE id=?", id)
	bid, code, msg := l.RunWithValidation(ctx, 7, wid, &RunRequest{EnvironmentID: 42, ServiceID: 31, ServiceRevisionID: 311, UseEnvironmentConfig: true, ExecutionMode: "build", Branch: "main"})
	if code != 0 {
		t.Fatalf("explicit Helm production build failed: %d %s", code, msg)
	}
	defer exec("UPDATE workflow_run SET status='failed' WHERE id=?", bid)
	buildRun, err := repo.FindRun(context.Background(), bid)
	if err != nil {
		t.Fatal(err)
	}
	for i, item := range buildRun.Tasks {
		if item.NodeType == NodeBuild {
			buildRun.Tasks[i].ProviderRunID = "build:342"
			buildRun.Tasks[i].Status = RunSucceeded
		}
		if item.NodeType == NodeHelmDeploy {
			task = item
		}
	}
	cfg, err = helmDeployConfig(task.NodeSnapshot)
	if err != nil {
		t.Fatal(err)
	}
	target, err = NewDefaultHelmResolver(NewSQLHelmSourceStore(db), nil).RenderHelm(ctx, *buildRun, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ValidateHelmProductionExecution(ctx, *buildRun, task, cfg, target); err != nil {
		t.Fatal("worker refused own successful Helm build", err)
	}
}
