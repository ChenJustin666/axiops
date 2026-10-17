package workflow

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/ChenJustin666/axiops/internal/shared/middleware"
)

type RunActor struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
}

func authenticatedRunActor(ctx context.Context, userID int64) *RunActor {
	name := strings.TrimSpace(middleware.GetUsername(ctx))
	if userID <= 0 || middleware.GetUserID(ctx) != userID || name == "" {
		return nil
	}
	return &RunActor{UserID: userID, Username: name}
}

func snapshotRunActor(run *WorkflowRun) bool {
	run.TriggerUsername = ""
	if run.TriggerUserID <= 0 {
		return true
	}
	var snapshot struct {
		Actor *RunActor `json:"trigger_actor"`
	}
	if json.Unmarshal(run.ContextSnapshot, &snapshot) == nil && snapshot.Actor != nil && snapshot.Actor.UserID == run.TriggerUserID && strings.TrimSpace(snapshot.Actor.Username) != "" {
		run.TriggerUsername = strings.TrimSpace(snapshot.Actor.Username)
		return true
	}
	return false
}

// Account lookup is supplemental display data, never an authorization source.
// Older runs use a single batch lookup; new runs keep their trigger-time account.
func (r *Repository) LoadRunActors(ctx context.Context, runs []WorkflowRun) error {
	ids := map[int64]bool{}
	for i := range runs {
		if !snapshotRunActor(&runs[i]) {
			ids[runs[i].TriggerUserID] = true
		}
	}
	if len(ids) == 0 {
		return nil
	}
	args := make([]interface{}, 0, len(ids))
	marks := make([]string, 0, len(ids))
	for id := range ids {
		args = append(args, id)
		marks = append(marks, "?")
	}
	rows, err := r.db.QueryContext(ctx, "SELECT id,username FROM user WHERE id IN ("+strings.Join(marks, ",")+")", args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	names := map[int64]string{}
	for rows.Next() {
		var id int64
		var name string
		if err = rows.Scan(&id, &name); err != nil {
			return err
		}
		names[id] = name
	}
	if err = rows.Err(); err != nil {
		return err
	}
	for i := range runs {
		if runs[i].TriggerUsername == "" && runs[i].TriggerUserID > 0 {
			runs[i].TriggerUsername = names[runs[i].TriggerUserID]
		}
	}
	return nil
}

func (l *Logic) loadRunActors(ctx context.Context, runs []WorkflowRun) {
	for i := range runs {
		snapshotRunActor(&runs[i])
	}
	if loader, ok := l.repo.(interface {
		LoadRunActors(context.Context, []WorkflowRun) error
	}); ok {
		if err := loader.LoadRunActors(ctx, runs); err != nil {
			// Keep status/logs available when legacy account data is unavailable.
			log.Printf("workflow executor account lookup unavailable: %v", err)
		}
	}
}
