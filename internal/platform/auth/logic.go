package auth

import (
	"context"
	"database/sql"

	"github.com/ChenJustin666/axiops/internal/platform/user"
	"github.com/ChenJustin666/axiops/internal/shared/config"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"github.com/ChenJustin666/axiops/internal/shared/jwtx"
	"golang.org/x/crypto/bcrypt"
)

// Logic 认证业务逻辑
type Logic struct {
	userRepo *user.Repository
	jwtCfg   config.JWTConfig
}

func NewLogic(db *sql.DB, jwtCfg config.JWTConfig) *Logic {
	return &Logic{
		userRepo: user.NewRepository(db),
		jwtCfg:   jwtCfg,
	}
}

// Login 登录
func (l *Logic) Login(ctx context.Context, req *LoginRequest, clientIP string) (*LoginResponse, int) {
	// 查找用户
	u, err := l.userRepo.FindByUsername(ctx, req.Username)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if u == nil {
		return nil, errcode.CodeLoginFailed
	}

	// 检查状态
	if u.Status != 1 {
		return nil, errcode.CodeUserDisabled
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.Password)); err != nil {
		return nil, errcode.CodeLoginFailed
	}

	// 获取角色
	roleIDs, err := l.userRepo.GetUserRoleIDs(ctx, u.ID)
	if err != nil {
		return nil, errcode.CodeDBError
	}

	// 生成 Token
	accessToken, err := jwtx.GenerateToken(l.jwtCfg.Secret, u.ID, u.Username, roleIDs, l.jwtCfg.AccessExpireSeconds)
	if err != nil {
		return nil, errcode.CodeInternalError
	}

	refreshToken, err := jwtx.GenerateRefreshToken(l.jwtCfg.Secret, u.ID, l.jwtCfg.RefreshExpireSeconds)
	if err != nil {
		return nil, errcode.CodeInternalError
	}

	// 更新登录信息
	_ = l.userRepo.UpdateLoginInfo(ctx, u.ID, clientIP)

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    l.jwtCfg.AccessExpireSeconds,
		UserInfo: &UserInfo{
			ID:       u.ID,
			Username: u.Username,
			Nickname: u.Nickname,
			Email:    u.Email,
			Avatar:   u.Avatar,
			RoleIDs:  roleIDs,
		},
	}, errcode.CodeSuccess
}

// RefreshToken 刷新 Token
func (l *Logic) RefreshToken(ctx context.Context, req *RefreshTokenRequest) (*RefreshTokenResponse, int) {
	// 解析刷新 Token
	claims, err := jwtx.ParseToken(req.RefreshToken, l.jwtCfg.Secret)
	if err != nil {
		return nil, errcode.CodeRefreshTokenFail
	}

	// 查找用户
	u, err := l.userRepo.FindByID(ctx, claims.UserID)
	if err != nil || u == nil {
		return nil, errcode.CodeRefreshTokenFail
	}

	if u.Status != 1 {
		return nil, errcode.CodeUserDisabled
	}

	// 获取角色
	roleIDs, err := l.userRepo.GetUserRoleIDs(ctx, u.ID)
	if err != nil {
		return nil, errcode.CodeDBError
	}

	// 生成新 Access Token
	accessToken, err := jwtx.GenerateToken(l.jwtCfg.Secret, u.ID, u.Username, roleIDs, l.jwtCfg.AccessExpireSeconds)
	if err != nil {
		return nil, errcode.CodeInternalError
	}

	return &RefreshTokenResponse{
		AccessToken: accessToken,
		ExpiresIn:   l.jwtCfg.AccessExpireSeconds,
	}, errcode.CodeSuccess
}

// GetCurrentUser 获取当前用户信息
func (l *Logic) GetCurrentUser(ctx context.Context, userID int64) (*UserInfo, int) {
	u, err := l.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if u == nil {
		return nil, errcode.CodeUserNotFound
	}

	roleIDs, err := l.userRepo.GetUserRoleIDs(ctx, u.ID)
	if err != nil {
		return nil, errcode.CodeDBError
	}

	return &UserInfo{
		ID:       u.ID,
		Username: u.Username,
		Nickname: u.Nickname,
		Email:    u.Email,
		Avatar:   u.Avatar,
		RoleIDs:  roleIDs,
	}, errcode.CodeSuccess
}
