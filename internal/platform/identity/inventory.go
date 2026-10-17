package identity

import (
	"context"
	"database/sql"
)

type InventoryReport struct {
	LegacyUsers         int64          `json:"legacy_users"`
	LegacyActiveUsers   int64          `json:"legacy_active_users"`
	TargetUsers         int64          `json:"target_users"`
	TargetActiveUsers   int64          `json:"target_active_users"`
	ExistingLinks       int64          `json:"existing_links"`
	MappedActiveUsers   int64          `json:"mapped_active_users"`
	UnmappedActiveUsers int64          `json:"unmapped_active_users"`
	LegacyDepartments   int64          `json:"legacy_departments"`
	TargetDepartments   int64          `json:"target_departments"`
	UsernameConflicts   []UsernameLink `json:"username_conflicts"`
}

type UsernameLink struct {
	LegacyUserID int64  `json:"legacy_user_id"`
	Username     string `json:"username"`
	TargetUserID int64  `json:"target_user_id"`
}

type Inventory struct {
	legacyDB *sql.DB
	targetDB *sql.DB
}

func NewInventory(legacyDB, targetDB *sql.DB) *Inventory {
	return &Inventory{legacyDB: legacyDB, targetDB: targetDB}
}

func (i *Inventory) Report(ctx context.Context, provider string, limit int) (*InventoryReport, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	report := &InventoryReport{}
	var err error
	if report.LegacyUsers, err = count(ctx, i.legacyDB, "SELECT COUNT(*) FROM sys_user"); err != nil {
		return nil, err
	}
	if report.LegacyActiveUsers, err = count(ctx, i.legacyDB, "SELECT COUNT(*) FROM sys_user WHERE status = 1 AND is_deleted = 0"); err != nil {
		return nil, err
	}
	if report.LegacyDepartments, err = count(ctx, i.legacyDB, "SELECT COUNT(*) FROM sys_dept WHERE status = 1 AND is_deleted = 0"); err != nil {
		return nil, err
	}
	if report.TargetUsers, err = count(ctx, i.targetDB, "SELECT COUNT(*) FROM user WHERE deleted_at IS NULL"); err != nil {
		return nil, err
	}
	if report.TargetActiveUsers, err = count(ctx, i.targetDB, "SELECT COUNT(*) FROM user WHERE status = 1 AND deleted_at IS NULL"); err != nil {
		return nil, err
	}
	if report.TargetDepartments, err = count(ctx, i.targetDB, "SELECT COUNT(*) FROM department"); err != nil {
		return nil, err
	}
	if report.ExistingLinks, err = countArgs(ctx, i.targetDB, "SELECT COUNT(*) FROM identity_link WHERE provider = ?", provider); err != nil {
		return nil, err
	}
	if report.MappedActiveUsers, err = i.countMappedActiveUsers(ctx, provider); err != nil {
		return nil, err
	}
	report.UnmappedActiveUsers = report.LegacyActiveUsers - report.MappedActiveUsers
	if report.UnmappedActiveUsers < 0 {
		report.UnmappedActiveUsers = 0
	}
	conflicts, err := i.usernameConflicts(ctx, limit)
	if err != nil {
		return nil, err
	}
	report.UsernameConflicts = conflicts
	return report, nil
}

func (i *Inventory) countMappedActiveUsers(ctx context.Context, provider string) (int64, error) {
	rows, err := i.legacyDB.QueryContext(ctx, `SELECT id FROM sys_user
		WHERE status = 1 AND is_deleted = 0 ORDER BY id`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var mapped int64
	for rows.Next() {
		var externalUserID int64
		if err := rows.Scan(&externalUserID); err != nil {
			return 0, err
		}
		var count int
		err := i.targetDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM identity_link il
			JOIN user u ON u.id = il.user_id
			WHERE il.provider = ? AND il.external_user_id = ? AND il.status = 1
			  AND u.status = 1 AND u.deleted_at IS NULL`, provider, externalUserID).Scan(&count)
		if err != nil {
			return 0, err
		}
		if count == 1 {
			mapped++
		}
	}
	return mapped, rows.Err()
}

func (i *Inventory) usernameConflicts(ctx context.Context, limit int) ([]UsernameLink, error) {
	// The legacy and target databases may be on different MySQL instances, so
	// fetch the bounded source set first instead of issuing a cross-database join.
	rows, err := i.legacyDB.QueryContext(ctx, `SELECT id, username FROM sys_user
		WHERE status = 1 AND is_deleted = 0 ORDER BY id LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var conflicts []UsernameLink
	for rows.Next() {
		var conflict UsernameLink
		if err := rows.Scan(&conflict.LegacyUserID, &conflict.Username); err != nil {
			return nil, err
		}
		err := i.targetDB.QueryRowContext(ctx, `SELECT id FROM user WHERE username = ? AND deleted_at IS NULL`, conflict.Username).Scan(&conflict.TargetUserID)
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			return nil, err
		}
		conflicts = append(conflicts, conflict)
	}
	return conflicts, rows.Err()
}

func count(ctx context.Context, db *sql.DB, query string) (int64, error) {
	return countArgs(ctx, db, query)
}

func countArgs(ctx context.Context, db *sql.DB, query string, args ...interface{}) (int64, error) {
	var value int64
	err := db.QueryRowContext(ctx, query, args...).Scan(&value)
	return value, err
}
