package workflow

import (
	"context"
	"database/sql"
	"testing"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"github.com/ChenJustin666/axiops/internal/shared/middleware"
)

func actorContext(id int64, name string) context.Context {
	ctx := context.WithValue(context.Background(), middleware.ContextKeyUserID, id)
	return context.WithValue(ctx, middleware.ContextKeyUsername, name)
}

func TestAuthenticatedRunActor(t *testing.T) {
	if got := authenticatedRunActor(actorContext(7, " deployer "), 7); got == nil || got.UserID != 7 || got.Username != "deployer" {
		t.Fatalf("missing authenticated actor: %+v", got)
	}
	for _, ctx := range []context.Context{context.Background(), actorContext(8, "viewer"), actorContext(7, " ")} {
		if got := authenticatedRunActor(ctx, 7); got != nil {
			t.Fatalf("untrusted actor accepted: %+v", got)
		}
	}
	if got := authenticatedRunActor(actorContext(0, "system-name"), 0); got != nil {
		t.Fatal("system gained a user identity")
	}
}

func TestRunViewsUseRecordedActorNotViewer(t *testing.T) {
	recorded := WorkflowRun{ID: 21, WorkflowID: 9, ProjectID: 10, TriggerUserID: 7, ContextSnapshot: mustJSON(map[string]interface{}{"trigger_actor": RunActor{UserID: 7, Username: "original-deployer"}})}
	repo := &logicRepository{item: &Workflow{ID: 9, ProjectID: 10}, run: &recorded, runs: []WorkflowRun{recorded}, runTotal: 1}
	logic := &Logic{repo: repo, authorizer: authz.New(policyStore{{Resource: "workflow", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10}})}
	ctx := actorContext(8, "current-viewer")
	run, code := logic.GetRun(ctx, 8, 21)
	if code != errcode.CodeSuccess || run.TriggerUsername != "original-deployer" {
		t.Fatalf("wrong detail actor: %+v code=%d", run, code)
	}
	runs, _, code := logic.ListRuns(ctx, 8, 9, 1, 20)
	if code != errcode.CodeSuccess || len(runs) != 1 || runs[0].TriggerUsername != "original-deployer" {
		t.Fatalf("wrong list actor: %+v code=%d", runs, code)
	}
}

func TestSnapshotActorRejectsWrongIdentity(t *testing.T) {
	for _, snapshot := range []string{`{"trigger_actor":{"user_id":8,"username":"other"}}`, `{"trigger_actor":{"user_id":7,"username":" "}}`, `invalid`, `{}`} {
		run := WorkflowRun{TriggerUserID: 7, TriggerUsername: "stale", ContextSnapshot: []byte(snapshot)}
		if snapshotRunActor(&run) || run.TriggerUsername != "" {
			t.Fatalf("invalid snapshot trusted: %s", snapshot)
		}
	}
}

// Runs inside the disposable schema; no production accounts are modified.
func testMySQLRunActors(t *testing.T, db *sql.DB) {
	if _, err := db.Exec("CREATE TABLE user (id BIGINT PRIMARY KEY,username VARCHAR(100) NOT NULL,deleted_at DATETIME NULL)"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO user(id,username,deleted_at) VALUES(7,'renamed-deployer',NULL),(8,'deleted-deployer',CURRENT_TIMESTAMP)"); err != nil {
		t.Fatal(err)
	}
	runs := []WorkflowRun{
		{TriggerUserID: 7, ContextSnapshot: mustJSON(map[string]interface{}{"trigger_actor": RunActor{UserID: 7, Username: "original-deployer"}})},
		{TriggerUserID: 7}, {TriggerUserID: 8}, {TriggerUserID: 999}, {},
		{TriggerUserID: 8, ContextSnapshot: mustJSON(map[string]interface{}{"trigger_actor": RunActor{UserID: 7, Username: "wrong-deployer"}})},
	}
	if err := NewRepository(db).LoadRunActors(context.Background(), runs); err != nil {
		t.Fatal(err)
	}
	for i, want := range []string{"original-deployer", "renamed-deployer", "deleted-deployer", "", "", "deleted-deployer"} {
		if runs[i].TriggerUsername != want {
			t.Fatalf("actor[%d]=%q want %q", i, runs[i].TriggerUsername, want)
		}
	}
}
