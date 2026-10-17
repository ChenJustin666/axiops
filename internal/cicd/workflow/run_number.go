package workflow

import (
	"context"
	"database/sql"
)

func allocateRunNumber(ctx context.Context, tx *sql.Tx, workflowID int64) (int64, error) {
	if workflowID == 0 {
		return 0, nil
	} // Standalone rollback has no workflow Job.
	var number int64
	if err := tx.QueryRowContext(ctx, `SELECT run_sequence FROM workflow WHERE id=? FOR UPDATE`, workflowID).Scan(&number); err != nil {
		return 0, err
	}
	number++
	_, err := tx.ExecContext(ctx, `UPDATE workflow SET run_sequence=? WHERE id=?`, number, workflowID)
	return number, err
}
func runNumberValue(number int64) interface{} {
	if number == 0 {
		return nil
	}
	return number
}
