package authz

import (
	"context"
	"database/sql"
)

type SQLStore struct {
	db *sql.DB
}

func NewSQLAuthorizer(db *sql.DB) *Authorizer {
	return New(&SQLStore{db: db})
}

func (s *SQLStore) ProjectDeleted(ctx context.Context, id int64) (bool, error) {
	var status string
	err := s.db.QueryRowContext(ctx, "SELECT status FROM project WHERE id=?", id).Scan(&status)
	if err == sql.ErrNoRows {
		return true, nil
	}
	return status == "deleted", err
}

// ListUserPolicies merges platform bindings, project membership, and scoped role bindings.
func (s *SQLStore) ListUserPolicies(ctx context.Context, userID int64) ([]Policy, error) {
	const query = `
		SELECT rp.resource, rp.action, rp.effect, 'platform', 0, '*'
		FROM user_role ur
		JOIN user u ON u.id = ur.user_id AND u.status = 1 AND u.deleted_at IS NULL
		JOIN role_permission rp ON rp.role_id = ur.role_id
		WHERE ur.user_id = ?
		UNION ALL
		SELECT rp.resource, rp.action, rp.effect, 'project', pm.project_id, '*'
		FROM project_member pm
		JOIN user u ON u.id = pm.user_id AND u.status = 1 AND u.deleted_at IS NULL
		JOIN role_permission rp ON rp.role_id = pm.role_id
		WHERE pm.user_id = ?
		UNION ALL
		SELECT rp.resource, rp.action, rp.effect, rb.scope_type, rb.scope_id, rb.environment
		FROM user u
		JOIN role_binding rb ON
		     (rb.subject_type = 'user' AND rb.subject_id = u.id)
		  OR (rb.subject_type = 'department' AND rb.subject_id = u.department_id)
		JOIN role_permission rp ON rp.role_id = rb.role_id
		WHERE u.id = ? AND u.status = 1 AND u.deleted_at IS NULL AND rb.status = 1
		  AND (rb.expires_at IS NULL OR rb.expires_at > CURRENT_TIMESTAMP)`
	rows, err := s.db.QueryContext(ctx, query, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []Policy
	for rows.Next() {
		var policy Policy
		if err := rows.Scan(&policy.Resource, &policy.Action, &policy.Effect, &policy.ScopeType, &policy.ScopeID, &policy.Environment); err != nil {
			return nil, err
		}
		policies = append(policies, policy)
	}
	return policies, rows.Err()
}
