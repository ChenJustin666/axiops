package projectstate

import (
	"context"
	"database/sql"
	"errors"
)

var ErrDeleted = errors.New("project is deleted and read-only")

// LockWritable serializes task submission with project deletion. Lock the
// project before service/workflow rows to keep submission lock order consistent.
func LockWritable(ctx context.Context, tx *sql.Tx, id int64) error {
	var status string
	if err := tx.QueryRowContext(ctx, "SELECT status FROM project WHERE id=? FOR UPDATE", id).Scan(&status); err != nil {
		return err
	}
	if status == "deleted" {
		return ErrDeleted
	}
	return nil
}
