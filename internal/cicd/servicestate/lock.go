// Package servicestate serializes service deletion with new delivery references.
package servicestate

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
)

func LockActive(ctx context.Context, tx *sql.Tx, projectID int64, ids ...int64) error {
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	var previous int64
	for _, id := range ids {
		if id <= 0 || id == previous {
			continue
		}
		previous = id
		var status string
		var project int64
		if err := tx.QueryRowContext(ctx, `SELECT project_id,status FROM app_service WHERE id=? FOR UPDATE`, id).Scan(&project, &status); err != nil {
			return err
		}
		if status != "active" || projectID > 0 && project != projectID {
			return fmt.Errorf("service %d is unavailable; restore it before using it", id)
		}
	}
	return nil
}

// IDs extracts only explicit service identity, never similarly named text fields.
func IDs(configs ...json.RawMessage) ([]int64, error) {
	var result []int64
	for _, raw := range configs {
		if len(raw) == 0 {
			continue
		}
		var value struct {
			ServiceID int64 `json:"service_id"`
		}
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, err
		}
		if value.ServiceID > 0 {
			result = append(result, value.ServiceID)
		}
	}
	return result, nil
}
