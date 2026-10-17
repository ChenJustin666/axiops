package authz

import (
	"context"
	"database/sql"
	"strings"

	"github.com/ChenJustin666/axiops/application/portal-rpc/internal/model"
)

// PrincipalReader reads the current security state, independently of JWT claims
// and model caches. A missing, deleted, disabled or reset-required user is nil.
type PrincipalReader interface {
	User(context.Context, uint64) (*Principal, error)
	Role(context.Context, uint64) (string, error)
	Token(context.Context, string) (*model.SysToken, error)
}

type Principal struct {
	UserID   uint64
	Username string
	Nickname string
	Roles    []string
}

type SQLPrincipalReader struct{ db *sql.DB }

func NewSQLPrincipalReader(db *sql.DB) *SQLPrincipalReader {
	return &SQLPrincipalReader{db: db}
}

func (s *SQLPrincipalReader) User(ctx context.Context, id uint64) (*Principal, error) {
	if id == 0 {
		return nil, nil
	}
	// One statement gives user status and assignments from the same DB snapshot.
	// LEFT JOIN preserves an active account with no roles: it gets no privileges.
	rows, err := s.db.QueryContext(ctx, `SELECT u.id, u.username, u.nickname, r.code
		FROM sys_user u
		LEFT JOIN sys_user_role ur ON ur.user_id=u.id AND ur.is_deleted=0
		LEFT JOIN sys_role r ON r.id=ur.role_id AND r.is_deleted=0
		WHERE u.id=? AND u.status=1 AND u.is_deleted=0 AND u.is_need_reset_pwd=0
		ORDER BY r.id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var principal *Principal
	seen := make(map[string]bool)
	for rows.Next() {
		var current Principal
		var code sql.NullString
		if err := rows.Scan(&current.UserID, &current.Username, &current.Nickname, &code); err != nil {
			return nil, err
		}
		if principal == nil {
			principal = &current
			principal.Roles = []string{}
		}
		value := strings.TrimSpace(code.String)
		key := strings.ToUpper(value)
		if value != "" && !seen[key] {
			principal.Roles = append(principal.Roles, value)
			seen[key] = true
		}
	}
	return principal, rows.Err()
}

func (s *SQLPrincipalReader) Role(ctx context.Context, id uint64) (string, error) {
	var code string
	err := s.db.QueryRowContext(ctx, "SELECT code FROM sys_role WHERE id=? AND is_deleted=0", id).Scan(&code)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return strings.TrimSpace(code), err
}

func (s *SQLPrincipalReader) Token(ctx context.Context, value string) (*model.SysToken, error) {
	var token model.SysToken
	// Never select or return token plaintext. Revocation cannot depend on a cached
	// model object (including negative cache entries).
	err := s.db.QueryRowContext(ctx, `SELECT id, owner_type, owner_id, name, expire_time, status
		FROM sys_token WHERE token=? AND BINARY token=BINARY ? AND is_deleted=0 LIMIT 1`, value, value).
		Scan(&token.Id, &token.OwnerType, &token.OwnerId, &token.Name, &token.ExpireTime, &token.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &token, nil
}
