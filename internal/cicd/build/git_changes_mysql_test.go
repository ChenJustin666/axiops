package build

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/ChenJustin666/axiops/internal/cicd/commitlink"
	"github.com/ChenJustin666/axiops/internal/cicd/workflow"
	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/config"
	"github.com/zeromicro/go-zero/core/conf"
)

// Copy schemas only and confine every write to a uniquely named test database.
func TestMySQLGitChangesSnapshotsAndVisibility(t *testing.T) {
	path := os.Getenv("AXIOPS_TEST_CONFIG")
	if path == "" {
		t.Skip("MySQL integration test not configured")
	}
	var config config.PlatformAPIConfig
	if conf.Load(path, &config) != nil {
		t.Fatal("could not load test configuration")
	}
	cfg, err := mysql.ParseDSN(config.MySQL.DSN())
	if err != nil {
		t.Fatal("invalid test DSN")
	}
	admin, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	source := "`" + strings.ReplaceAll(cfg.DBName, "`", "``") + "`"
	schema := fmt.Sprintf("axiops_changes_test_%d", time.Now().UnixNano())
	if _, err = admin.Exec("CREATE DATABASE `" + schema + "` CHARACTER SET utf8mb4"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := admin.Exec("DROP DATABASE `" + schema + "`"); err != nil {
			t.Error(err)
		}
	}()
	cfg.DBName, cfg.ParseTime = schema, true
	db, err := sql.Open("mysql", cfg.FormatDSN())
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
	for _, table := range []string{"build_run", "build_command", "app_service", "code_source", "workflow_run", "workflow_task_run"} {
		exec("CREATE TABLE `" + table + "` LIKE " + source + ".`" + table + "`")
	}
	exec("CREATE TABLE project(id BIGINT PRIMARY KEY,status VARCHAR(16) DEFAULT 'active')")
	exec("INSERT INTO project(id) VALUES(1),(2)")
	var hasColumn int
	if err := db.QueryRow("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=? AND table_name='build_run' AND column_name='git_changes'", schema).Scan(&hasColumn); err != nil {
		t.Fatal(err)
	}
	if hasColumn == 0 {
		migration, err := os.ReadFile("../../../migrations/043_build_git_changes.sql")
		if err != nil {
			t.Fatal(err)
		}
		exec(string(migration))
	}
	exec("INSERT INTO app_service(id,project_id,name,code,registry_id) VALUES(1,1,'api','api',1),(2,1,'worker','worker',1),(3,2,'other','other',1)")
	exec("INSERT INTO code_source(id,name,type,url,project_id) VALUES(1,'Git','gitea','https://git.example',1)")
	head, base := strings.Repeat("b", 40), strings.Repeat("a", 40)
	ctx, repo := context.Background(), NewRepository(db)
	newRun := func(service, project int64, number int, branch, url, commit string) *BuildRun {
		t.Helper()
		raw, _ := json.Marshal(BuildExecutionSnapshot{RepositoryURL: url, CodeSourceID: 1})
		snapshot := string(raw)
		run := &BuildRun{ServiceID: service, ProjectID: project, BuildNumber: number, Status: StatusQueued, Provider: KubeJobProviderName, GitBranch: branch, GitCommit: commit, ConfigSnapshot: &snapshot}
		id, err := repo.CreateRunWithSubmitCommand(ctx, run)
		if err != nil {
			t.Fatal(err)
		}
		run.ID = id
		return run
	}
	url := "https://git.example/team/original.git"
	previous := newRun(1, 1, 1, "main", url, base)
	exec("UPDATE build_run SET status='success' WHERE id=?", previous.ID)
	for _, item := range []*BuildRun{newRun(1, 1, 2, "qa", url, head), newRun(1, 1, 3, "main", "https://git.example/team/other", head), newRun(2, 1, 1, "main", url, head)} {
		exec("UPDATE build_run SET status='success' WHERE id=?", item.ID)
	}
	failed := newRun(1, 1, 4, "main", url, head)
	exec("UPDATE build_run SET status='failed' WHERE id=?", failed.ID)
	run := newRun(1, 1, 5, "main", url, head)
	var snapshot BuildExecutionSnapshot
	if err := json.Unmarshal([]byte(*run.ConfigSnapshot), &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.ChangeBaseline == nil || snapshot.ChangeBaseline.BuildID != previous.ID || snapshot.CodeSourceType != "gitea" {
		t.Fatalf("wrong frozen baseline: %+v", snapshot.ChangeBaseline)
	}
	// A later success and an edited service repository cannot rewrite this run.
	later := newRun(1, 1, 6, "main", url, head)
	exec("UPDATE build_run SET status='success' WHERE id=?", later.ID)
	exec("UPDATE app_service SET repo_url='https://git.example/team/moved.git' WHERE id=1")
	clone := *run
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err := freezeGitBaseline(ctx, tx, &clone); err != nil {
		t.Fatal(err)
	}
	if *clone.ConfigSnapshot != *run.ConfigSnapshot {
		t.Fatal("retry changed baseline")
	}
	tx.Rollback()
	latest := GitCommitChange{Commit: head, Author: strings.Repeat("中", 200), Email: strings.Repeat("x", 300), Subject: strings.Repeat("说明", 20000), AuthoredAt: "2026-10-02T10:00:00+08:00"}
	changes := &GitChangeSet{Status: "unavailable", Head: head, BaseCommit: base, Latest: &latest, Commits: []GitCommitChange{latest}, Files: []GitFileChange{}}
	if err := repo.SaveGitChanges(ctx, *run, changes); err != nil {
		t.Fatal(err)
	}
	changes.Status = "complete"
	changes.Files = []GitFileChange{{Action: "M", Path: "部署/service.yaml"}}
	if err := repo.SaveGitChanges(ctx, *run, changes); err != nil {
		t.Fatal(err)
	}
	partial := *changes
	partial.Status = "unavailable"
	partial.Files = nil
	if err := repo.SaveGitChanges(ctx, *run, &partial); err != nil {
		t.Fatal(err)
	}
	stored, err := NewRepository(db).FindRunByID(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	var saved GitChangeSet
	if err := json.Unmarshal([]byte(stored.GitChangesJSON), &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Status != "complete" || len(saved.Files) != 1 || saved.BaseBuildID != previous.ID || !strings.Contains(stored.GitCommitURL, "/team/original/commit/") || len([]rune(stored.GitAuthor)) != 128 || len(saved.Latest.Subject) != len(latest.Subject) {
		t.Fatal("metadata failed persistence, immutable source, or Unicode bounds checks")
	}
	// One batch returns each service, enforces build scope, excludes other projects.
	exec("INSERT INTO workflow_run(id,workflow_id,project_id,environment_id,definition_version,context_snapshot) VALUES(1,1,1,1,1,'{}')")
	other := newRun(3, 2, 1, "main", url, head)
	for i, item := range []*BuildRun{run, later, other} {
		exec("INSERT INTO workflow_task_run(workflow_run_id,node_key,node_type,depends_on,node_snapshot,provider_run_id) VALUES(1,?,'build','[]','{}',?)", fmt.Sprintf("build%d", i), fmt.Sprintf("build:%d", item.ID))
	}
	for _, tc := range []struct {
		scope authz.ScopeSet
		want  int
	}{{authz.ScopeSet{AllowAll: true}, 2}, {authz.ScopeSet{BuildIDs: []int64{run.ID}}, 1}, {authz.ScopeSet{DenyAll: true}, 0}} {
		runs := []workflow.WorkflowRun{{ID: 1}}
		if err := workflow.NewRepository(db).LoadBuildCommits(ctx, runs, tc.scope); err != nil {
			t.Fatal(err)
		}
		if len(runs[0].BuildCommits) != tc.want {
			t.Fatalf("scope leaked commits: wanted %d got %d", tc.want, len(runs[0].BuildCommits))
		}
	}
	t.Run("legacy links work without change history", func(t *testing.T) {
		exec("UPDATE build_run SET git_commit_url='',git_changes=NULL,config_snapshot=JSON_REMOVE(config_snapshot,'$.code_source_type','$.change_baseline') WHERE id=?", later.ID)
		legacy, err := repo.FindRunByID(ctx, later.ID)
		if err != nil {
			t.Fatal(err)
		}
		batch := []BuildRun{*legacy}
		if err := repo.fillCommitLinks(ctx, batch); err != nil {
			t.Fatal(err)
		}
		want := "https://git.example/team/original/commit/" + head
		if batch[0].GitCommitURL != want || batch[0].GitChangesJSON != "" {
			t.Fatal("legacy link needs no fabricated changes or current service repo")
		}
		runs := []workflow.WorkflowRun{{ID: 1}}
		if err := workflow.NewRepository(db).LoadBuildCommits(ctx, runs, authz.ScopeSet{AllowAll: true}); err != nil {
			t.Fatal(err)
		}
		if len(runs[0].BuildCommits) != 2 || runs[0].BuildCommits[1].URL != want {
			t.Fatal("workflow list did not resolve same legacy link")
		}
		var savedURL string
		if err := db.QueryRow("SELECT git_commit_url FROM build_run WHERE id=?", later.ID).Scan(&savedURL); err != nil || savedURL != "" {
			t.Fatal("read path changed immutable record")
		}
		// Unknown, moved, deleted, or other-project provider data cannot be used.
		for _, item := range []struct {
			query string
			args  []interface{}
		}{
			{"UPDATE code_source SET url='https://moved.example/' WHERE id=1", nil},
			{"UPDATE code_source SET url='https://git.example',project_id=2 WHERE id=1", nil},
			{"UPDATE code_source SET project_id=1,type='unknown' WHERE id=1", nil},
			{"DELETE FROM code_source WHERE id=1", nil},
		} {
			exec(item.query, item.args...)
			records := []commitlink.Record{{ProjectID: 1, Commit: head, Snapshot: commitlink.ParseSnapshot(legacy.ConfigSnapshot)}}
			if err := commitlink.Fill(ctx, db, records); err != nil {
				t.Fatal(err)
			}
			if records[0].URL != "" {
				t.Fatal("untrusted provider generated a historical URL")
			}
		}
	})
}
