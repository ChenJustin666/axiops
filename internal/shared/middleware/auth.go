package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/ChenJustin666/axiops/internal/shared/authn"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"github.com/ChenJustin666/axiops/internal/shared/jwtx"
	"github.com/ChenJustin666/axiops/internal/shared/response"
)

// context key 类型
type contextKey string

const (
	ContextKeyUserID   contextKey = "user_id"
	ContextKeyUsername contextKey = "username"
	ContextKeyRoleIDs  contextKey = "role_ids"
)

// AuthMiddleware JWT 认证中间件
type AuthMiddleware struct {
	secret string
	audit  *AuditMiddleware
	legacy authn.LegacyAuthenticator
}

func NewAuthMiddleware(secret string) *AuthMiddleware {
	return &AuthMiddleware{secret: secret}
}

func (m *AuthMiddleware) WithAudit(audit *AuditMiddleware) *AuthMiddleware {
	m.audit = audit
	return m
}

func (m *AuthMiddleware) WithLegacyAuthenticator(authenticator authn.LegacyAuthenticator) *AuthMiddleware {
	m.legacy = authenticator
	return m
}

func (m *AuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 从 Header 获取 Token
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			response.Error(w, errcode.CodeTokenInvalid)
			return
		}

		// Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Error(w, errcode.CodeTokenInvalid)
			return
		}

		tokenStr := parts[1]

		// 解析 Token
		claims, err := jwtx.ParseToken(tokenStr, m.secret)
		var userID int64
		var username string
		var roleIDs []int64
		if err == nil {
			userID = claims.UserID
			username = claims.Username
			roleIDs = claims.RoleIDs
		} else if err == jwtx.ErrTokenInvalid && m.legacy != nil {
			principal, legacyErr := m.legacy.Authenticate(r.Context(), authHeader)
			if legacyErr == nil {
				userID = principal.UserID
				username = principal.Username
			} else {
				response.Error(w, errcode.CodeTokenInvalid)
				return
			}
		} else {
			if err == jwtx.ErrTokenExpired {
				response.Error(w, errcode.CodeTokenExpired)
			} else {
				response.Error(w, errcode.CodeTokenInvalid)
			}
			return
		}

		// 将用户信息注入 context
		ctx := r.Context()
		ctx = context.WithValue(ctx, ContextKeyUserID, userID)
		ctx = context.WithValue(ctx, ContextKeyUsername, username)
		ctx = context.WithValue(ctx, ContextKeyRoleIDs, roleIDs)

		authenticatedRequest := r.WithContext(ctx)
		if m.audit != nil {
			m.audit.Handle(next)(w, authenticatedRequest)
			return
		}
		next(w, authenticatedRequest)
	}
}

// GetUserID 从 context 获取用户 ID
func GetUserID(ctx context.Context) int64 {
	if v, ok := ctx.Value(ContextKeyUserID).(int64); ok {
		return v
	}
	return 0
}

// GetUsername 从 context 获取用户名
func GetUsername(ctx context.Context) string {
	if v, ok := ctx.Value(ContextKeyUsername).(string); ok {
		return v
	}
	return ""
}

// GetRoleIDs 从 context 获取角色 ID 列表
func GetRoleIDs(ctx context.Context) []int64 {
	if v, ok := ctx.Value(ContextKeyRoleIDs).([]int64); ok {
		return v
	}
	return nil
}
