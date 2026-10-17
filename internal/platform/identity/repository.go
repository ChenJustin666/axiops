package identity

import (
	"context"
	"database/sql"
	"errors"

	"github.com/ChenJustin666/axiops/internal/shared/authn"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) TargetPrincipal(ctx context.Context, provider string, externalUserID int64) (authn.Principal, error) {
	var principal authn.Principal
	err := r.db.QueryRowContext(ctx, `SELECT u.id, u.username
		FROM identity_link il
		JOIN user u ON u.id = il.user_id
		WHERE il.provider = ? AND il.external_user_id = ? AND il.status = 1
		  AND u.status = 1 AND u.deleted_at IS NULL`, provider, externalUserID).
		Scan(&principal.UserID, &principal.Username)
	if err == nil {
		return principal, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return authn.Principal{}, err
	}
	return authn.Principal{}, r.mappingState(ctx, provider, externalUserID)
}

func (r *Repository) mappingState(ctx context.Context, provider string, externalUserID int64) error {
	var status int
	var targetStatus int
	var deletedAt sql.NullTime
	err := r.db.QueryRowContext(ctx, `SELECT il.status, u.status, u.deleted_at
		FROM identity_link il JOIN user u ON u.id = il.user_id
		WHERE il.provider = ? AND il.external_user_id = ?`, provider, externalUserID).
		Scan(&status, &targetStatus, &deletedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrIdentityUnmapped
	}
	if err != nil {
		return err
	}
	if status != 1 || targetStatus != 1 || deletedAt.Valid {
		return ErrTargetUserDisabled
	}
	return ErrIdentityUnmapped
}
