package workflow

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"
)

func testMySQLLeaseRecoveryClock(t *testing.T, database *sql.DB, workflowID int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	result, err := database.ExecContext(ctx, `INSERT INTO workflow_run(workflow_id,project_id,environment_id,definition_version,status,context_snapshot) VALUES(?,19,0,1,'running','{}')`, workflowID)
	if err != nil {
		t.Fatal(err)
	}
	runID, _ := result.LastInsertId()
	node := Node{Key: "deploy-clock", Name: "Deployment", Type: NodeYAMLDeploy}
	_, err = database.ExecContext(ctx, `INSERT INTO workflow_task_run(workflow_run_id,node_key,node_type,depends_on,node_snapshot,status,locked_by,started_at,lease_until) VALUES(?,?,'yaml_deploy','[]',?,'running','crashed',DATE_SUB(CURRENT_TIMESTAMP,INTERVAL 65 SECOND),DATE_SUB(CURRENT_TIMESTAMP,INTERVAL 2 SECOND))`, runID, node.Key, mustJSON(node))
	if err != nil {
		t.Fatal(err)
	}
	repo := NewRepository(database)
	run, err := repo.FindRun(ctx, runID)
	if err != nil {
		t.Fatal(err)
	}
	if len(run.Tasks) != 1 || !run.Tasks[0].LeaseUntil.Before(time.Now()) {
		t.Fatal("SQL-expired lease appears unexpired in Go")
	}
	if ok, err := repo.ClaimTask(ctx, runID, node.Key, "recovery", time.Minute); err != nil || !ok {
		t.Fatalf("claim=%v err=%v", ok, err)
	}
	// Contending lanes must not deadlock the owner renewal or steal its live lease.
	var wg sync.WaitGroup
	errors := make(chan error, 40)
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			if ok, err := repo.ClaimTask(ctx, runID, node.Key, "other-lane", time.Minute); err != nil {
				errors <- err
			} else if ok {
				errors <- fmt.Errorf("live lease stolen")
			}
		}()
		go func() {
			defer wg.Done()
			if status, err := repo.RenewTaskLease(ctx, runID, node.Key, "recovery", time.Minute); err != nil {
				errors <- err
			} else if status != RunRunning {
				errors <- fmt.Errorf("renewal status=%s", status)
			}
		}()
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		t.Error(err)
	}
	if t.Failed() {
		return
	}
	if err := repo.ReleaseTaskLease(ctx, runID, node.Key, "recovery"); err != nil {
		t.Fatal(err)
	}
	run, err = repo.FindRun(ctx, runID)
	if err != nil {
		t.Fatal(err)
	}
	executors, _ := NewExecutorRegistry(fakeExecutor{typeName: NodeYAMLDeploy})
	if err := NewWorker(repo, executors, time.Minute, 10).processRun(ctx, "resumed", *run); err != nil {
		t.Fatal(err)
	}
	run, err = repo.FindRun(ctx, runID)
	if err != nil {
		t.Fatal(err)
	}
	task := run.Tasks[0]
	if task.Status != RunSucceeded || task.StartedAt == nil || task.FinishedAt == nil || task.FinishedAt.Sub(*task.StartedAt) < 65*time.Second || task.LogText == "" {
		t.Fatalf("recovery lost original duration or progress: %+v", task)
	}
}
