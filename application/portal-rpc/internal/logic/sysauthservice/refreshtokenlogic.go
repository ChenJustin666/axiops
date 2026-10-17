package sysauthservicelogic

import (
	"context"
	"fmt"

	"github.com/ChenJustin666/axiops/application/portal-rpc/internal/code"
	"github.com/ChenJustin666/axiops/application/portal-rpc/internal/common"
	"github.com/ChenJustin666/axiops/application/portal-rpc/pb"
	"github.com/ChenJustin666/axiops/common/vars"
	"github.com/ChenJustin666/axiops/pkg/jwt"

	"github.com/ChenJustin666/axiops/application/portal-rpc/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

type RefreshTokenLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRefreshTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RefreshTokenLogic {
	return &RefreshTokenLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 刷新令牌
func (l *RefreshTokenLogic) RefreshToken(in *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	// 拿到刷新令牌
	if in.RefreshToken == "" {
		return nil, code.RefreshTokenEmptyErr
	}
	jwtR, err := jwt.VerifyToken("Bearer "+in.RefreshToken, vars.RefreshSecret)
	if err != nil {
		return nil, err
	}
	account, principalErr := l.currentRefreshAccount(jwtR)
	if principalErr != nil {
		return nil, principalErr
	}
	// 查询用户信息
	redisKey := fmt.Sprintf("%s%s", common.UuidKeyPrefix, jwtR.UserName.UserName)
	if common.AllowMultiLogin {
		redisKey = fmt.Sprintf("%s%s:%s", common.UuidKeyPrefix, jwtR.UserName.UserName, jwtR.UserName.Uuid)
	}
	key, errs := l.svcCtx.Cache.Get(redisKey)
	if errs != nil {
		l.Errorf("Redis 查询该 uuid 的访问令牌, 错误: %v, uuid: %v", err, redisKey)
		return nil, code.UUIDQueryErr
	}
	if key == "" {
		return nil, code.UUIDNotExistErr
	}
	accessToken, errs := jwt.CreateJWTToken(account, vars.AccessSecret, account.Uuid, vars.AccessExpire)
	if errs != nil {
		l.Errorf("生成JWT令牌失败, 错误: %v", err)
		return nil, code.GenerateJWTTokenErr
	}
	errs = l.svcCtx.Cache.Setex(redisKey, accessToken.AccessToken, int(vars.AccessExpire))
	if errs != nil {
		l.Errorf("存储访问令牌到 Redis 失败: %v", err)
		return nil, code.RedisStorageErr
	}
	return &pb.RefreshTokenResponse{
		AccessToken:     accessToken.AccessToken,
		AccessExpiresIn: accessToken.ExpiresAt,
	}, nil
}

func (l *RefreshTokenLogic) currentRefreshAccount(claims *jwt.JWTClaims) (*jwt.AccountInfo, error) {
	principal, err := NewVerifyTokenLogic(l.ctx, l.svcCtx).currentJWTPrincipal(claims)
	if err != nil {
		return nil, err
	}
	if !principal.IsValid {
		return nil, jwt.ErrInvalidToken
	}
	return &jwt.AccountInfo{
		UserId: principal.UserId, UserName: principal.Username,
		NickName: principal.NickName, Roles: principal.Roles, Uuid: principal.Uuid,
	}, nil
}
