package workflow

import (
	"context"
	"regexp"
	"strings"
	"time"
)

// Do not collect manifests, env values or Secret bodies. Mask common credential
// forms which can also be present in Kubernetes event/error messages.
var diagnosticCredentials = regexp.MustCompile(`(?i)(password|passwd|token|secret|authorization|api[_-]?key)(["']?\s*[:=]\s*["']?)[^\s,;"']+`)
var diagnosticURLAuth = regexp.MustCompile(`(https?://)[^\s/@]+:[^\s/@]+@`)
var diagnosticBearer = regexp.MustCompile(`(?i)Bearer\s+[^\s,;]+`)

func safeDiagnostic(text string) string {
	text = diagnosticURLAuth.ReplaceAllString(text, "${1}[REDACTED]@")
	text = diagnosticBearer.ReplaceAllString(text, "Bearer [REDACTED]")
	text = diagnosticCredentials.ReplaceAllString(text, "${1}${2}[REDACTED]")
	return truncateDiagnostic(text, 4096)
}

func truncateDiagnostic(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	return strings.ToValidUTF8(text[:limit], "") + " [truncated]"
}

func appendTaskLog(previous, message string, now time.Time) string {
	message = truncateDiagnostic(strings.TrimSpace(message), 24*1024)
	// Kubernetes polling returns observations, not new application output.
	// Persist changed facts only; derive state from durable history so worker
	// restarts do not replay every unchanged condition and event.
	if strings.HasPrefix(message, "[check] cluster=") {
		message = deploymentObservationDelta(previous, message)
	}
	if message == "" || strings.HasSuffix(previous, "\n"+message+"\n") {
		return previous
	}
	entry := "[" + now.UTC().Format(time.RFC3339) + "]\n" + message + "\n"
	const limit = 64 * 1024
	if len(previous)+len(entry) > limit {
		// Preserve initial apply evidence and the most recent diagnostic snapshots.
		head := truncateDiagnostic(previous, 4096)
		tailSize := limit - len(head) - len(entry) - 128
		tail := previous
		if len(tail) > tailSize {
			tail = strings.ToValidUTF8(tail[len(tail)-tailSize:], "")
		}
		previous = head + "\n[system] earlier checks truncated\n" + tail
	}
	return previous + entry
}

func deploymentObservationDelta(previous, message string) string {
	latest := map[string]string{}
	keyFor := func(line string) string {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return ""
		}
		switch fields[0] {
		case "[event]", "[resource]":
			return line // preserve new timestamps/messages, omit exact event replays
		case "[check]":
			if strings.HasPrefix(fields[1], "cluster=") {
				return "target"
			}
		case "[rollout]":
			if strings.Contains(fields[1], "/") {
				return "rollout:" + fields[1]
			}
		case "[condition]", "[pod]":
			if len(fields) >= 3 {
				property, _, _ := strings.Cut(fields[2], "=")
				return fields[0] + fields[1] + ":" + property
			}
		case "[waiting]":
			return "waiting"
		}
		return "" // failures, unknown messages and observation gaps stay visible
	}
	for _, line := range strings.Split(previous, "\n") {
		if key := keyFor(line); key != "" {
			latest[key] = line
		}
	}
	var delta []string
	for _, line := range strings.Split(message, "\n") {
		key := keyFor(line)
		if key != "" {
			if latest[key] == line {
				continue
			}
			latest[key] = line
		}
		delta = append(delta, line)
	}
	return strings.TrimSpace(strings.Join(delta, "\n"))
}

func (r *Repository) AppendTaskLog(ctx context.Context, runID int64, nodeKey, message string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var previous, status string
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(log_text,''),status FROM workflow_task_run WHERE workflow_run_id=? AND node_key=? FOR UPDATE`, runID, nodeKey).Scan(&previous, &status); err != nil {
		return err
	}
	if status != RunRunning {
		return tx.Commit()
	}
	next := appendTaskLog(previous, message, time.Now())
	if next != previous {
		if _, err := tx.ExecContext(ctx, `UPDATE workflow_task_run SET log_text=? WHERE workflow_run_id=? AND node_key=?`, next, runID, nodeKey); err != nil {
			return err
		}
	}
	return tx.Commit()
}
