package workflow

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

// Called only with a disposable schema by the environment binding integration test.
func testMySQLOrderedResults(t *testing.T, db *sql.DB, workflowID int64) {
	ctx := context.Background()
	repo := NewRepository(db)
	for _, failure := range []string{"none", "submission", "health", "build"} {
		t.Run(failure, func(t *testing.T) {
			result, err := db.Exec(`INSERT INTO workflow_run(workflow_id,project_id,environment_id,definition_version,status,context_snapshot) VALUES(?,19,0,2,'running','{}')`, workflowID)
			if err != nil {
				t.Fatal(err)
			}
			runID, _ := result.LastInsertId()
			for index, key := range []string{"server", "gateway"} {
				node := Node{Key: key, Type: NodeYAMLDeploy, Config: mustJSON(map[string]int{"start_order": index + 1})}
				if _, err := db.Exec(`INSERT INTO workflow_task_run(workflow_run_id,node_key,node_type,depends_on,node_snapshot,status) VALUES(?,?,'yaml_deploy','[]',?,'pending')`, runID, key, mustJSON(node)); err != nil {
					t.Fatal(err)
				}
			}
			checkStatus := func(want string) {
				t.Helper()
				var status string
				if err := db.QueryRow(`SELECT status FROM workflow_run WHERE id=?`, runID).Scan(&status); err != nil || status != want {
					t.Fatalf("status=%s want=%s err=%v", status, want, err)
				}
			}
			if failure == "build" {
				node := Node{Key: "build", Type: NodeBuild}
				if _, err := db.Exec(`INSERT INTO workflow_task_run(workflow_run_id,node_key,node_type,depends_on,node_snapshot,status) VALUES(?,'build','build','[]',?,'pending')`, runID, mustJSON(node)); err != nil {
					t.Fatal(err)
				}
				if ok, err := repo.ClaimTask(ctx, runID, "build", "test", time.Minute); err != nil || !ok {
					t.Fatalf("claim %t %v", ok, err)
				}
				if err := repo.CompleteTask(ctx, runID, "build", "test", RunFailed, "image push failed"); err != nil {
					t.Fatal(err)
				}
				checkStatus(RunRunning)
				for _, key := range []string{"server", "gateway"} {
					if err := repo.SkipTask(ctx, runID, key); err != nil {
						t.Fatal(err)
					}
				}
				checkStatus(RunFailed)
				return
			}
			if ok, err := repo.ClaimTask(ctx, runID, "server", "test", time.Minute); err != nil || !ok {
				t.Fatalf("claim %t %v", ok, err)
			}
			if failure == "submission" {
				if err := repo.CompleteTask(ctx, runID, "server", "test", RunFailed, "apply failed"); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := repo.MarkTaskSubmitted(ctx, runID, "server", "test", ExecutionResult{ProviderRunID: "yaml:server"}); err != nil {
					t.Fatal(err)
				}
				if failure == "health" {
					if err := repo.CompleteObservedTask(ctx, runID, TaskRun{NodeKey: "server", ProviderRunID: "yaml:server"}, RunFailed, "CrashLoopBackOff"); err != nil {
						t.Fatal(err)
					}
				}
			}
			checkStatus(RunRunning)
			run, err := repo.FindRun(ctx, runID)
			if err != nil {
				t.Fatal(err)
			}
			plan, err := Plan(*run)
			if err != nil || len(plan.Runnable) != 1 || plan.Runnable[0] != "gateway" {
				t.Fatalf("gateway blocked: %+v %v", plan, err)
			}
			if ok, err := repo.ClaimTask(ctx, runID, "gateway", "test", time.Minute); err != nil || !ok {
				t.Fatalf("claim %t %v", ok, err)
			}
			if err := repo.CompleteTask(ctx, runID, "gateway", "test", RunSucceeded, ""); err != nil {
				t.Fatal(err)
			}
			if failure == "none" {
				checkStatus(RunRunning)
				if err := repo.CompleteObservedTask(ctx, runID, TaskRun{NodeKey: "server", ProviderRunID: "yaml:server"}, RunSucceeded, ""); err != nil {
					t.Fatal(err)
				}
				checkStatus(RunSucceeded)
			} else {
				checkStatus(RunFailed)
			}
		})
	}
}
