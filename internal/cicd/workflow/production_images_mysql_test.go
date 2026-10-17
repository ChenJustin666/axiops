package workflow

import (
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
)

// Uses the existing disposable MySQL schema; no cluster/provider writes.
func testMySQLProductionImageSelection(t *testing.T, db *sql.DB, l *Logic, wid int64) {
	ctx := actorContext(7, "release-operator")
	exec := func(q string, args ...interface{}) {
		t.Helper()
		if _, err := db.Exec(q, args...); err != nil {
			t.Fatal(err)
		}
	}
	repo := NewRepository(db)
	exec("UPDATE project_environment SET status='disabled' WHERE id=1")
	defer exec("UPDATE project_environment SET status='active' WHERE id=1")
	options, err := l.ProductionOptions(ctx, 7, wid)
	if err != nil || len(options.Services) != 1 || len(options.Services[0].Images) == 0 || options.Policy.SourceEnvironmentID != 0 {
		t.Fatalf("successful builds must work without source policy: %+v %v", options, err)
	}
	p, err := l.ProductionImagePreview(ctx, 7, wid, 42)
	if err != nil || p.BlockedReason != "" || !strings.Contains(p.TargetManifest, "new-prod-db") || strings.Contains(p.TargetManifest, "test-db") || p.Candidate.Method != "build" {
		t.Fatalf("preview %+v %v", p, err)
	}
	exec("UPDATE environment_service_override SET project_environment_id=999 WHERE project_environment_id=2 AND service_id=1")
	if _, err = l.ProductionImagePreview(ctx, 7, wid, 42); err == nil {
		t.Fatal("production image used unsaved base config")
	}
	if _, code, _ := l.RunWithValidation(ctx, 7, wid, &RunRequest{EnvironmentID: 2, ServiceID: 1, ServiceRevisionID: 11, UseEnvironmentConfig: true, ExecutionMode: "build", Branch: "main"}); code == 0 {
		t.Fatal("production build used unsaved base config")
	}
	exec("UPDATE environment_service_override SET project_environment_id=2 WHERE project_environment_id=999 AND service_id=1")
	exec("UPDATE build_run SET service_id=999 WHERE id=42")
	if _, err = l.ProductionImagePreview(ctx, 7, wid, 42); err == nil {
		t.Fatal("foreign service accepted")
	}
	exec("UPDATE build_run SET service_id=1 WHERE id=42")
	for _, pick := range []ProductionPick{{}, {BuildID: 42, VerificationID: 1}, {BuildID: -1}, {BuildID: 999999}} {
		if _, err = loadProductionPickPreview(ctx, db, wid, pick); err == nil {
			t.Fatalf("invalid pick accepted %+v", pick)
		}
	}
	exec("UPDATE build_run SET status='failed' WHERE id=42")
	if _, err = l.ProductionImagePreview(ctx, 7, wid, 42); err == nil {
		t.Fatal("failed build accepted")
	}
	exec("UPDATE build_run SET status='success',image_digest='' WHERE id=42")
	if _, err = l.ProductionImagePreview(ctx, 7, wid, 42); err == nil {
		t.Fatal("missing digest accepted")
	}
	digest := p.build.ImageDigest
	exec("UPDATE build_run SET image_digest=?,project_id=20 WHERE id=42", digest)
	if _, err = l.ProductionImagePreview(ctx, 7, wid, 42); err == nil {
		t.Fatal("foreign project accepted")
	}
	exec("UPDATE build_run SET project_id=19 WHERE id=42")
	req := ProductionSubmission{RequestKey: "production-direct-artifact", Picks: []ProductionPick{{BuildID: 42, ExpectedToken: p.Token}}}
	exec("UPDATE variable_item SET value_text='draft-changed' WHERE variable_set_id=2")
	if _, err = l.SubmitProduction(ctx, 7, wid, req); err == nil {
		t.Fatal("stale config accepted")
	}
	exec("UPDATE variable_item SET value_text='new-prod-db' WHERE variable_set_id=2")
	p, err = l.ProductionImagePreview(ctx, 7, wid, 42)
	if err != nil {
		t.Fatal(err)
	}
	req.Picks[0].ExpectedToken = p.Token
	id, err := l.SubmitProduction(ctx, 7, wid, req)
	if err != nil {
		t.Fatal(err)
	}
	defer exec("UPDATE workflow_run SET status='failed' WHERE id=?", id)
	if retry, err := l.SubmitProduction(ctx, 7, wid, req); err != nil || retry != id {
		t.Fatalf("idempotency %d %v", retry, err)
	}
	run, err := repo.FindRun(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	var provenance struct {
		Mode string `json:"execution_mode"`
	}
	if json.Unmarshal(run.ContextSnapshot, &provenance) != nil || provenance.Mode != "selected_image" || run.TriggerUserID != 7 {
		t.Fatal("release provenance missing")
	}
	store := NewSQLDeploymentSnapshotStore(db)
	var task TaskRun
	for _, item := range run.Tasks {
		if item.NodeType == NodeYAMLDeploy {
			task = item
		}
	}
	if err = store.ValidateProductionExecution(ctx, *run, task); err != nil {
		t.Fatal("worker rejected selected artifact", err)
	}
	exec("UPDATE build_run SET git_commit='changed-artifact' WHERE id=42")
	if err = store.ValidateProductionExecution(ctx, *run, task); err == nil {
		t.Fatal("worker accepted altered artifact")
	}
	exec("UPDATE build_run SET git_commit=? WHERE id=42", p.build.Commit)
	exec("UPDATE project_environment SET namespace='wrong-prod' WHERE id=2")
	if err = store.ValidateProductionExecution(ctx, *run, task); err == nil {
		t.Fatal("worker followed changed target")
	}
	exec("UPDATE project_environment SET namespace='api-prod' WHERE id=2")
	exec("UPDATE workflow_run SET status='failed' WHERE id=?", id)
	// Explicit build runs work with the production workflow itself, independently of source environments.
	buildID, code, msg := l.RunWithValidation(ctx, 7, wid, &RunRequest{EnvironmentID: 2, ServiceID: 1, ServiceRevisionID: 11, UseEnvironmentConfig: true, ExecutionMode: "build", Branch: "main"})
	if code != 0 {
		t.Fatalf("explicit production build %d %s", code, msg)
	}
	buildRun, err := repo.FindRun(ctx, buildID)
	if err != nil {
		t.Fatal(err)
	}
	for i, item := range buildRun.Tasks {
		if item.NodeType == NodeBuild {
			buildRun.Tasks[i].ProviderRunID = "build:42"
			buildRun.Tasks[i].Status = RunSucceeded
		}
		if item.NodeType == NodeYAMLDeploy {
			task = item
			var cfg YAMLDeployConfig
			var n Node
			json.Unmarshal(item.NodeSnapshot, &n)
			json.Unmarshal(n.Config, &cfg)
			if cfg.FrozenYAML == nil || cfg.FrozenYAML.ImageRef != "" {
				t.Fatal("build did not freeze own configuration")
			}
		}
	}
	if err = store.ValidateProductionExecution(ctx, *buildRun, task); err != nil {
		t.Fatal("explicit build execution", err)
	}
	exec("UPDATE workflow_run SET status='failed' WHERE id=?", buildID)
}
