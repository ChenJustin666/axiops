package workflow

import (
	"context"
	"strings"

	"github.com/ChenJustin666/axiops/internal/cicd/commitlink"
	"github.com/ChenJustin666/axiops/internal/shared/authz"
)

// One bounded query for the visible page, with independent build-view scope.
// Each build node keeps its own commit; parallel services are never collapsed.
func (r *Repository) LoadBuildCommits(ctx context.Context, runs []WorkflowRun, scope authz.ScopeSet) error {
	if len(runs) == 0 {
		return nil
	}
	byID := map[int64]int{}
	args := make([]interface{}, 0, len(runs))
	for i, run := range runs {
		byID[run.ID] = i
		args = append(args, run.ID)
	}
	predicate, permissions := scope.SQLPredicate("br.project_id", "br.service_id", "br.id")
	args = append(args, permissions...)
	rows, err := r.db.QueryContext(ctx, `SELECT task.workflow_run_id,br.id,br.service_id,br.git_branch,br.git_commit,
		COALESCE(br.git_message,''),br.git_author,br.git_commit_url,br.project_id,
		COALESCE(JSON_UNQUOTE(JSON_EXTRACT(br.config_snapshot,'$.repository_url')),''),
		COALESCE(CAST(JSON_UNQUOTE(JSON_EXTRACT(br.config_snapshot,'$.code_source_id')) AS SIGNED),0),
		COALESCE(JSON_UNQUOTE(JSON_EXTRACT(br.config_snapshot,'$.code_source_type')),'')
		FROM workflow_task_run task JOIN workflow_run wr ON wr.id=task.workflow_run_id
		JOIN build_run br ON br.id=CAST(SUBSTRING(task.provider_run_id,7) AS UNSIGNED)
		AND task.provider_run_id=CONCAT('build:',br.id) AND br.project_id=wr.project_id
		WHERE task.node_type='build' AND task.workflow_run_id IN (`+strings.TrimRight(strings.Repeat("?,", len(runs)), ",")+`) AND (`+predicate+`) ORDER BY task.id`, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	var records []commitlink.Record
	var positions [][2]int
	for rows.Next() {
		var id int64
		var item BuildCommitSummary
		var record commitlink.Record
		if err := rows.Scan(&id, &item.BuildID, &item.ServiceID, &item.Branch, &item.Commit, &item.Subject, &item.Author, &item.URL,
			&record.ProjectID, &record.Snapshot.RepositoryURL, &record.Snapshot.CodeSourceID, &record.Snapshot.CodeSourceType); err != nil {
			return err
		}
		if i, ok := byID[id]; ok {
			positions = append(positions, [2]int{i, len(runs[i].BuildCommits)})
			record.Commit, record.URL = item.Commit, item.URL
			records = append(records, record)
			runs[i].BuildCommits = append(runs[i].BuildCommits, item)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if err := commitlink.Fill(ctx, r.db, records); err != nil {
		return err
	}
	for i, pos := range positions {
		runs[pos[0]].BuildCommits[pos[1]].URL = records[i].URL
	}
	return nil
}
