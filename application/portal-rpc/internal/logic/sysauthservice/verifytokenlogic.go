package sysauthservicelogic

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ChenJustin666/axiops/application/portal-rpc/internal/code"
	"github.com/ChenJustin666/axiops/application/portal-rpc/internal/common"
	"github.com/ChenJustin666/axiops/application/portal-rpc/internal/model"
	"github.com/ChenJustin666/axiops/application/portal-rpc/pb"
	"github.com/ChenJustin666/axiops/common/handler/errorx"
	"github.com/ChenJustin666/axiops/common/vars"
	"github.com/ChenJustin666/axiops/pkg/jwt"

	"github.com/ChenJustin666/axiops/application/portal-rpc/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

type VerifyTokenLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewVerifyTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *VerifyTokenLogic {
	return &VerifyTokenLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 验证令牌
func (l *VerifyTokenLogic) VerifyToken(in *pb.VerifyTokenRequest) (*pb.VerifyTokenResponse, error) {
	jwtToken, err := l.verifyJWTToken(in.Token)
	if err != nil {
		tokenResp, matched, tokenErr := l.verifySysToken(in.Token)
		if tokenErr != nil {
			return nil, tokenErr
		}
		if matched {
			return tokenResp, nil
		}
		return &pb.VerifyTokenResponse{
			IsValid:      false,
			ErrorType:    int64(err.Code()),
			ErrorMessage: err.Message(),
		}, nil
	}
	if jwtToken.UserName == nil || jwtToken.UserName.UserId == 0 || strings.TrimSpace(jwtToken.UserName.UserName) == "" {
		return invalidTokenResponse("会话账号无效"), nil
	}
	redisKey := fmt.Sprintf("%s%s", common.UuidKeyPrefix, jwtToken.UserName.UserName)
	if common.AllowMultiLogin {
		redisKey = fmt.Sprintf("%s%s:%s", common.UuidKeyPrefix, jwtToken.UserName.UserName, jwtToken.UserName.Uuid)
	}
	key, errs := l.svcCtx.Cache.Get(redisKey)
	if errs != nil {
		l.Errorf("Redis 查询该 uuid 的访问令牌, 错误: %v, uuid: %v", err, redisKey)
		return nil, code.UUIDQueryErr
	}
	if key == "" {
		l.Infof("Redis 不存在该 uuid 的访问令牌, 错误: %v , key: %v", key, err)
		return &pb.VerifyTokenResponse{
			IsValid:      false,
			ErrorType:    100099,
			ErrorMessage: "token 已经被禁用，请联系管理员处理!",
		}, nil
	}

	if "Bearer "+key != in.Token {
		return &pb.VerifyTokenResponse{
			IsValid:      false,
			ErrorType:    int64(jwt.ErrTokenUsed.Code()),
			ErrorMessage: jwt.ErrTokenUsed.Message(),
		}, nil
	}

	return l.currentJWTPrincipal(jwtToken)
}

// A signed JWT identifies the session; it does not freeze its role grants.
func (l *VerifyTokenLogic) currentJWTPrincipal(jwtToken *jwt.JWTClaims) (*pb.VerifyTokenResponse, error) {
	if jwtToken == nil || jwtToken.UserName == nil || jwtToken.UserName.UserId == 0 {
		return invalidTokenResponse("会话账号无效"), nil
	}
	if l.svcCtx.Principals == nil {
		return nil, errorx.Msg("账号权限校验尚未就绪")
	}
	principal, err := l.svcCtx.Principals.User(l.ctx, jwtToken.UserName.UserId)
	if err != nil {
		l.Error("查询当前账号权限失败")
		return nil, errorx.Msg("查询当前账号权限失败")
	}
	if principal == nil || principal.UserID != jwtToken.UserName.UserId || principal.Username != jwtToken.UserName.UserName {
		return invalidTokenResponse("账号已失效，请重新登录"), nil
	}
	return &pb.VerifyTokenResponse{
		IsValid:      true,
		ExpireTime:   jwtToken.ExpiresAt,
		Username:     principal.Username,
		UserId:       principal.UserID,
		Roles:        principal.Roles,
		NickName:     principal.Nickname,
		Uuid:         jwtToken.UserName.Uuid,
		ErrorType:    0,
		ErrorMessage: "",
	}, nil
}

func (l *VerifyTokenLogic) verifyJWTToken(token string) (*jwt.JWTClaims, errorx.ErrorX) {
	jwtToken, err := jwt.VerifyToken(token, vars.AccessSecret)
	if err == nil {
		return jwtToken, nil
	}
	configSecret := strings.TrimSpace(l.svcCtx.Config.AuthConfig.AccessSecret)
	if configSecret == "" || configSecret == vars.AccessSecret {
		return nil, err
	}
	if jwtToken, configErr := jwt.VerifyToken(token, configSecret); configErr == nil {
		return jwtToken, nil
	}
	return nil, err
}

func (l *VerifyTokenLogic) verifySysToken(authToken string) (*pb.VerifyTokenResponse, bool, error) {
	tokenValue := normalizeAuthToken(authToken)
	if tokenValue == "" {
		return nil, false, nil
	}
	if l.svcCtx.Principals == nil {
		return nil, true, errorx.Msg("账号权限校验尚未就绪")
	}
	token, err := l.svcCtx.Principals.Token(l.ctx, tokenValue)
	if err != nil {
		l.Error("查询系统Token失败")
		return nil, true, errorx.Msg("Token查询失败")
	}
	if token == nil {
		return nil, false, nil
	}
	if token.Status != 1 {
		return invalidTokenResponse("Token已禁用"), true, nil
	}
	if token.ExpireTime.Valid && !token.ExpireTime.Time.After(time.Now()) {
		return &pb.VerifyTokenResponse{
			IsValid:      false,
			ErrorType:    int64(jwt.ErrTokenExpired.Code()),
			ErrorMessage: "Token已过期",
		}, true, nil
	}
	resp, err := l.sysTokenPrincipal(token)
	if err != nil {
		return nil, true, err
	}
	return resp, true, nil
}

func (l *VerifyTokenLogic) sysTokenPrincipal(token *model.SysToken) (*pb.VerifyTokenResponse, error) {
	if l.svcCtx.Principals == nil {
		return nil, errorx.Msg("账号权限校验尚未就绪")
	}
	expireTime := int64(0)
	if token.ExpireTime.Valid {
		expireTime = token.ExpireTime.Time.Unix()
	}
	uuid := fmt.Sprintf("sys-token:%d", token.Id)
	switch token.OwnerType {
	case 1:
		user, err := l.svcCtx.Principals.User(l.ctx, token.OwnerId)
		if err != nil {
			l.Error("查询Token所属用户失败")
			return nil, errorx.Msg("Token所属用户查询失败")
		}
		if user == nil || user.UserID != token.OwnerId {
			return invalidTokenResponse("Token所属用户已失效"), nil
		}
		return &pb.VerifyTokenResponse{
			IsValid:      true,
			ExpireTime:   expireTime,
			Username:     user.Username,
			UserId:       user.UserID,
			Roles:        user.Roles,
			NickName:     user.Nickname,
			Uuid:         uuid,
			ErrorType:    0,
			ErrorMessage: "",
		}, nil
	case 2:
		role, err := l.svcCtx.Principals.Role(l.ctx, token.OwnerId)
		if err != nil {
			l.Error("查询Token所属角色失败")
			return nil, errorx.Msg("Token所属角色查询失败")
		}
		if role == "" {
			return invalidTokenResponse("Token所属角色已失效"), nil
		}
		return &pb.VerifyTokenResponse{
			IsValid:      true,
			ExpireTime:   expireTime,
			Username:     token.Name,
			UserId:       0,
			Roles:        []string{role},
			NickName:     token.Name,
			Uuid:         uuid,
			ErrorType:    0,
			ErrorMessage: "",
		}, nil
	default:
		return invalidTokenResponse("Token所有者类型无效"), nil
	}
}

func invalidTokenResponse(message string) *pb.VerifyTokenResponse {
	return &pb.VerifyTokenResponse{
		IsValid:      false,
		ErrorType:    int64(jwt.ErrInvalidToken.Code()),
		ErrorMessage: message,
	}
}

func normalizeAuthToken(token string) string {
	token = strings.Trim(strings.TrimSpace(token), `"'`)
	if token == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(token), "authorization:") {
		token = strings.TrimSpace(token[len("authorization:"):])
	}
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		token = strings.TrimSpace(token[7:])
	}
	return strings.Trim(strings.TrimSpace(token), `"'`)
}
