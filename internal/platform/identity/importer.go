package identity

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type ImportEntry struct {
	ExternalUserID int64  `json:"external_user_id"`
	ExternalName   string `json:"external_username"`
	TargetUserID   int64  `json:"target_user_id"`
}

type ImportResult struct {
	Validated int `json:"validated"`
	Inserted  int `json:"inserted"`
	Existing  int `json:"existing"`
}

type Importer struct {
	legacyDB *sql.DB
	targetDB *sql.DB
}

func NewImporter(legacyDB, targetDB *sql.DB) *Importer {
	return &Importer{legacyDB: legacyDB, targetDB: targetDB}
}

func (i *Importer) Import(ctx context.Context, provider string, entries []ImportEntry, apply bool) (*ImportResult, error) {
	if provider == "" || len(entries) == 0 {
		return nil, errors.New("provider and at least one mapping are required")
	}
	if err := validateImportEntries(entries); err != nil {
		return nil, err
	}
	result := &ImportResult{}
	for _, entry := range entries {
		if err := i.validateEntry(ctx, entry); err != nil {
			return nil, err
		}
		result.Validated++
	}
	if !apply {
		return result, nil
	}

	tx, err := i.targetDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	for _, entry := range entries {
		existing, err := existingMapping(ctx, tx, provider, entry)
		if err != nil {
			return nil, err
		}
		if existing {
			result.Existing++
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO identity_link
			(provider, external_user_id, user_id, external_username, status)
			VALUES (?, ?, ?, ?, 1)`, provider, entry.ExternalUserID, entry.TargetUserID, entry.ExternalName); err != nil {
			return nil, fmt.Errorf("insert mapping for external user %d: %w", entry.ExternalUserID, err)
		}
		result.Inserted++
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

func validateImportEntries(entries []ImportEntry) error {
	externalIDs := make(map[int64]struct{}, len(entries))
	targetIDs := make(map[int64]struct{}, len(entries))
	for index, entry := range entries {
		if entry.ExternalUserID <= 0 || entry.TargetUserID <= 0 || entry.ExternalName == "" {
			return fmt.Errorf("mapping %d has invalid identifiers or username", index)
		}
		if _, exists := externalIDs[entry.ExternalUserID]; exists {
			return fmt.Errorf("duplicate external user id %d", entry.ExternalUserID)
		}
		if _, exists := targetIDs[entry.TargetUserID]; exists {
			return fmt.Errorf("duplicate target user id %d", entry.TargetUserID)
		}
		externalIDs[entry.ExternalUserID] = struct{}{}
		targetIDs[entry.TargetUserID] = struct{}{}
	}
	return nil
}

func (i *Importer) validateEntry(ctx context.Context, entry ImportEntry) error {
	var legacyName string
	err := i.legacyDB.QueryRowContext(ctx, `SELECT username FROM sys_user
		WHERE id = ? AND status = 1 AND is_deleted = 0`, entry.ExternalUserID).Scan(&legacyName)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("legacy user %d is missing or disabled", entry.ExternalUserID)
	}
	if err != nil {
		return err
	}
	if legacyName != entry.ExternalName {
		return fmt.Errorf("legacy user %d username mismatch: manifest=%q database=%q", entry.ExternalUserID, entry.ExternalName, legacyName)
	}
	var targetName string
	err = i.targetDB.QueryRowContext(ctx, `SELECT username FROM user
		WHERE id = ? AND status = 1 AND deleted_at IS NULL`, entry.TargetUserID).Scan(&targetName)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("target user %d is missing or disabled", entry.TargetUserID)
	}
	if err != nil {
		return err
	}
	if targetName != entry.ExternalName {
		return fmt.Errorf("target user %d username %q does not match reviewed legacy username %q", entry.TargetUserID, targetName, entry.ExternalName)
	}
	return nil
}

type queryer interface {
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

func existingMapping(ctx context.Context, query queryer, provider string, entry ImportEntry) (bool, error) {
	var externalUserID, targetUserID int64
	err := query.QueryRowContext(ctx, `SELECT external_user_id, user_id FROM identity_link
		WHERE provider = ? AND (external_user_id = ? OR user_id = ?)`, provider, entry.ExternalUserID, entry.TargetUserID).
		Scan(&externalUserID, &targetUserID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if externalUserID != entry.ExternalUserID || targetUserID != entry.TargetUserID {
		return false, fmt.Errorf("conflicting mapping: external user %d is linked to target user %d", externalUserID, targetUserID)
	}
	return true, nil
}
