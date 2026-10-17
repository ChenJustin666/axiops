package workflow

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
)

// Runs only inside the disposable schema owned by the numbering/promotion suite.
func testMySQLVerifiedProduction(t *testing.T, db *sql.DB, productionWorkflowID int64) {
	ctx := actorContext(7, "qa-tester")
	repo := NewRepository(db)
	store := NewSQLDeploymentSnapshotStore(db)
	exec := func(query string, args ...interface{}) {
		t.Helper()
		if _, err := db.Exec(query, args...); err != nil {
			t.Fatal(err)
		}
	}
	body, err := os.ReadFile("../../../migrations/049_verified_production_release.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range strings.Split(string(body), ";") {
		if strings.TrimSpace(statement) != "" {
			exec(statement)
		}
	}
	logic := &Logic{repo: repo, authorizer: authz.New(policyStore{{Resource: "*", Action: "*", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 19, Environment: "*"}}), yamlResolver: NewDefaultYAMLResolver(NewSQLYAMLSourceStore(db))}
	t.Run("direct artifact and explicit production build", func(t *testing.T) { testMySQLProductionImageSelection(t, db, logic, productionWorkflowID) })
	qa := &Workflow{ProjectID: 19, TargetEnvironmentID: 1, Name: "QA verified", Code: "qa-verified", DeliveryType: DeliveryTypeK8sYAML, Nodes: []Node{{Key: "build", Type: NodeBuild, Config: mustJSON(map[string]interface{}{"service_id": 1})}, {Key: "deploy", Type: NodeYAMLDeploy, DependsOn: []string{"build"}, Config: mustJSON(YAMLDeployConfig{ServiceID: 1, ServiceRevisionID: 11})}}}
	qaID, err := repo.Create(ctx, qa)
	if err != nil {
		t.Fatal(err)
	}
	qaRunID, code, msg := logic.RunWithValidation(ctx, 7, qaID, &RunRequest{EnvironmentID: 1, ServiceID: 1, ServiceRevisionID: 11, UseEnvironmentConfig: true, ExecutionMode: "existing_image", SourceBuildID: 42})
	if code != 0 {
		t.Fatalf("QA %d %s", code, msg)
	}
	qaRun, err := repo.FindRun(ctx, qaRunID)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := yamlDeployConfig(qaRun.Tasks[0].NodeSnapshot)
	if err != nil {
		t.Fatal(err)
	}
	target, err := logic.yamlResolver.RenderYAML(ctx, *qaRun, cfg)
	if err != nil {
		t.Fatal(err)
	}
	snap, err := NewYAMLDeploymentSnapshot(*qaRun, qaRun.Tasks[0], cfg, target)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.Prepare(ctx, snap); err != nil {
		t.Fatal(err)
	}
	if err = store.MarkApplied(ctx, qaRunID, qaRun.Tasks[0].NodeKey, NodeYAMLDeploy); err != nil {
		t.Fatal(err)
	}
	exec("UPDATE workflow_run SET status='succeeded' WHERE id=?", qaRunID)
	snap, err = scanDeployment(db, ctx, "workflow_run_id=?", qaRunID)
	if err != nil {
		t.Fatal(err)
	}
	options, err := logic.VerificationOptions(ctx, 7, qaRunID)
	if err != nil || len(options) != 1 || options[0].Verification != nil || options[0].BlockedReason != "" {
		t.Fatalf("verification options %+v %v", options, err)
	}
	if _, err = logic.ProductionPolicy(ctx, 7, 2, &ProductionPolicy{SourceEnvironmentID: 2}); err == nil {
		t.Fatal("self source accepted")
	}
	policy, err := logic.ProductionPolicy(ctx, 7, 2, &ProductionPolicy{SourceEnvironmentID: 1})
	if err != nil || policy.Revision != 1 {
		t.Fatal("policy", policy, err)
	}
	initial, err := logic.ProductionOptions(ctx, 7, productionWorkflowID)
	if err != nil || len(initial.Services) != 1 || len(initial.Services[0].Candidates) != 0 {
		t.Fatalf("unverified build became eligible: %+v %v", initial, err)
	}
	deployedImage := false
	for _, image := range initial.Services[0].Images {
		if image.BuildID == 42 && image.EnvironmentID == 1 && image.DeploymentStatus == "succeeded" {
			deployedImage = true
		}
	}
	if !deployedImage {
		t.Fatal("actual applied snapshot was not recognized as a successful source deployment")
	}
	originalInput := VerificationInput{SnapshotID: snap.ID, ExpectedDigest: snap.ManifestDigest, Action: "pass", Note: "登录、查询接口回归通过"}
	bad := originalInput
	bad.ExpectedDigest = strings.Repeat("0", 64)
	if _, err = logic.SaveVerification(ctx, 7, qaRunID, bad); err == nil {
		t.Fatal("stale acceptance accepted")
	}
	noPermission := &Logic{repo: repo, authorizer: authz.New(policyStore{{Resource: "workflow", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 19}})}
	if _, err = noPermission.SaveVerification(ctx, 7, qaRunID, originalInput); err == nil {
		t.Fatal("read-only user accepted tests")
	}
	verification, err := logic.SaveVerification(ctx, 7, qaRunID, originalInput)
	if err != nil || verification.Username != "qa-tester" || verification.Method != "manual" {
		t.Fatalf("verification %+v %v", verification, err)
	}
	options2, err := logic.ProductionOptions(ctx, 7, productionWorkflowID)
	if err != nil || len(options2.Services[0].Candidates) != 1 || !options2.Services[0].Candidates[0].Recommended {
		t.Fatalf("recommendation %+v %v", options2, err)
	}
	// Repeated successful builds with the identical reference cannot rewrite provenance.
	exec(`INSERT INTO build_run(id,service_id,project_id,build_number,status,provider,image_url,image_digest,git_commit,git_branch) SELECT 44,service_id,project_id,3,status,provider,image_url,image_digest,'duplicate-build',git_branch FROM build_run WHERE id=42`)
	preview, err := logic.ProductionPreview(ctx, 7, productionWorkflowID, verification.ID)
	if err != nil || preview.BlockedReason != "" || !strings.Contains(preview.TargetManifest, "new-prod-db") || strings.Contains(preview.TargetManifest, "test-db") || !strings.Contains(preview.TargetManifest, "6Gi") {
		t.Fatalf("production config not isolated: %+v %v", preview, err)
	}
	if preview.Candidate.BuildID != 42 {
		t.Fatal("acceptance provenance changed to duplicate build")
	}
	if _, code, _ = logic.RunWithValidation(ctx, 7, productionWorkflowID, &RunRequest{EnvironmentID: 2, ServiceID: 1, ServiceRevisionID: 11, UseEnvironmentConfig: true, ExecutionMode: "existing_image", SourceBuildID: 42}); code == 0 {
		t.Fatal("old production API bypassed verification")
	}
	exec("UPDATE variable_item SET value_text='changed-after-preview' WHERE variable_set_id=2")
	request := ProductionSubmission{RequestKey: "production-stale-config", Picks: []ProductionPick{{VerificationID: verification.ID, ExpectedToken: preview.Token}}}
	if _, err = logic.SubmitProduction(ctx, 7, productionWorkflowID, request); err == nil {
		t.Fatal("stale config preview accepted")
	}
	exec("UPDATE variable_item SET value_text='new-prod-db' WHERE variable_set_id=2")
	preview, err = logic.ProductionPreview(ctx, 7, productionWorkflowID, verification.ID)
	if err != nil {
		t.Fatal(err)
	}
	type outcome struct {
		id      int64
		err     error
		request ProductionSubmission
	}
	out := make(chan outcome, 2)
	var wg sync.WaitGroup
	for _, key := range []string{"production-concurrent-a", "production-concurrent-b"} {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			req := ProductionSubmission{RequestKey: key, Picks: []ProductionPick{{VerificationID: verification.ID, ExpectedToken: preview.Token}}}
			id, err := logic.SubmitProduction(ctx, 7, productionWorkflowID, req)
			out <- outcome{id, err, req}
		}(key)
	}
	wg.Wait()
	close(out)
	var winner outcome
	count := 0
	for result := range out {
		if result.err == nil {
			winner = result
			count++
		}
	}
	if count != 1 {
		t.Fatalf("concurrent production submissions accepted=%d", count)
	}
	queued, err := repo.FindRun(ctx, winner.id)
	if err != nil || queued.Status != RunPendingApproval || len(queued.Tasks) != 2 {
		t.Fatalf("bad run %+v %v", queued, err)
	}
	for _, task := range queued.Tasks {
		if task.NodeType == NodeBuild {
			t.Fatal("production rebuilt the artifact")
		}
	}
	if id, err := logic.SubmitProduction(ctx, 7, productionWorkflowID, winner.request); err != nil || id != winner.id {
		t.Fatal("lost-response retry did not find original", err)
	}
	revoke := originalInput
	revoke.Action = "revoke"
	revoke.Revision = verification.Revision
	revoke.Note = "发现业务兼容性问题，撤销"
	revoked, err := logic.SaveVerification(ctx, 7, qaRunID, revoke)
	if err != nil {
		t.Fatal(err)
	}
	provider := &fakeYAMLProvider{}
	executor := NewYAMLDeployExecutor(logic.yamlResolver, provider, store)
	if _, err = executor.Execute(ctx, ExecutionRequest{Run: *queued, Task: queued.Tasks[1]}); err == nil {
		t.Fatal("worker deployed revoked version")
	}
	exec("UPDATE workflow_run SET status='failed' WHERE id=?", winner.id)
	originalInput.Revision = revoked.Revision
	verification, err = logic.SaveVerification(ctx, 7, qaRunID, originalInput)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = logic.SubmitProduction(ctx, 7, productionWorkflowID, ProductionSubmission{RequestKey: "production-old-acceptance", Picks: winner.request.Picks}); err == nil {
		t.Fatal("old acceptance revision submitted")
	}
	preview, err = logic.ProductionPreview(ctx, 7, productionWorkflowID, verification.ID)
	if err != nil {
		t.Fatal(err)
	}
	submit := ProductionSubmission{RequestKey: "production-final-release", Picks: []ProductionPick{{VerificationID: verification.ID, ExpectedToken: preview.Token}}}
	runID, err := logic.SubmitProduction(ctx, 7, productionWorkflowID, submit)
	if err != nil {
		t.Fatal(err)
	}
	run, err := repo.FindRun(ctx, runID)
	if err != nil {
		t.Fatal(err)
	}
	// Configuration changes after submission do not rewrite immutable run inputs.
	exec("UPDATE variable_item SET value_text='future-draft' WHERE variable_set_id=2")
	result, err := executor.Execute(ctx, ExecutionRequest{Run: *run, Task: run.Tasks[1]})
	if err != nil {
		t.Fatal(err)
	}
	if provider.request.Namespace != "api-prod" || !strings.Contains(string(provider.request.Manifest), verification.ImageRef) || !strings.Contains(string(provider.request.Manifest), "new-prod-db") || strings.Contains(string(provider.request.Manifest), "future-draft") {
		t.Fatal("runtime did not use exact approved image and frozen production config")
	}
	recovered, err := executor.Execute(ctx, ExecutionRequest{Run: *run, Task: run.Tasks[1]})
	if err != nil || recovered.ProviderRunID != result.ProviderRunID {
		t.Fatal("worker recovery not idempotent", err)
	}
	task := run.Tasks[1]
	task.ProviderRunID = result.ProviderRunID
	if _, err = executor.Reconcile(ctx, ExecutionRequest{Run: *run, Task: task}); err != nil {
		t.Fatal(err)
	}
	exec("UPDATE workflow_run SET status='succeeded' WHERE id=?", runID)
	exec("UPDATE variable_item SET value_text='new-prod-db' WHERE variable_set_id=2")
	same, err := logic.ProductionPreview(ctx, 7, productionWorkflowID, verification.ID)
	if err != nil || !strings.Contains(same.BlockedReason, "无需重复") {
		t.Fatalf("same version not blocked: %+v %v", same, err)
	}
	if id, err := logic.SubmitProduction(ctx, 7, productionWorkflowID, submit); err != nil || id != runID {
		t.Fatal("completed retry changed run", err)
	}
	// Any later deployment fences an old worker from replaying its publication.
	latest, err := scanDeployment(db, ctx, "workflow_run_id=?", runID)
	if err != nil {
		t.Fatal(err)
	}
	copy := *latest
	copy.ID = 0
	copy.WorkflowRunID = queued.ID
	copy.NodeKey = "newer-production"
	copy.WorkflowTaskRunID = queued.Tasks[1].ID
	copy.Status = "prepared"
	if err = store.Prepare(ctx, &copy); err != nil {
		t.Fatal(err)
	}
	if _, err = executor.Execute(ctx, ExecutionRequest{Run: *run, Task: run.Tasks[1]}); err == nil {
		t.Fatal("old production worker overwrote newer snapshot")
	}
	var events int
	if err = db.QueryRow("SELECT COUNT(*) FROM deployment_verification_event WHERE verification_id=?", verification.ID).Scan(&events); err != nil || events != 3 {
		t.Fatal("acceptance audit missing", events, err)
	}
	var frozen struct {
		Guards []ProductionGuard `json:"production_guards"`
	}
	if json.Unmarshal(run.ContextSnapshot, &frozen) != nil || len(frozen.Guards) != 1 || frozen.Guards[0].ImageRef != verification.ImageRef {
		t.Fatal("provenance missing")
	}
	// A matching digest in another service/project is never an eligible candidate.
	exec("UPDATE deployment_verification SET environment_id=999 WHERE id=?", verification.ID)
	if _, err = logic.ProductionPreview(context.Background(), 7, productionWorkflowID, verification.ID); err == nil {
		t.Fatal("foreign-source verification accepted")
	}
}
