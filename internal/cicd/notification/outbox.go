package notification

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

// EnqueueTerminal inserts one delivery per matching subscription in the same
// transaction as the state change. External delivery is always asynchronous.
func EnqueueTerminal(ctx context.Context, tx *sql.Tx, projectID, triggerUserID int64, eventType string, eventID int64, status string, message Message) error {
	if tx == nil || projectID <= 0 || eventID <= 0 {
		return fmt.Errorf("invalid notification event")
	}
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}
	success := status == "success" || status == "succeeded"
	_, err = tx.ExecContext(ctx, `INSERT IGNORE INTO notification_outbox(subscription_id,project_id,event_type,event_id,payload,status,delivery_key)
		SELECT s.id,s.project_id,?,?,?,'pending',CONCAT(s.id,':',?,':',?)
		FROM notification_subscription s WHERE s.project_id=? AND s.enabled=1 AND (s.user_id=0 OR s.user_id=?)
		AND ((?=1 AND s.on_success=1) OR (?=0 AND s.on_failure=1))`, eventType, eventID, payload, eventType, eventID, projectID, triggerUserID, success, success)
	return err
}
