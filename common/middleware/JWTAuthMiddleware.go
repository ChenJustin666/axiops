package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/ChenJustin666/axiops/application/portal-rpc/client/sysauthservice"
)

type Response struct {
	Code    int64  `json:"code"`    // 应用自定义状态码
	Data    any    `json:"data"`    // 响应数据
	Message string `json:"message"` // 消息描述
}

type JWTAuthMiddleware struct {
	auth      sysauthservice.SysAuthService
	resources ResourceAccess
}

type ResourceAccess interface {
	Check(*http.Request, func(context.Context) error) (*http.Request, error)
}

func (m *JWTAuthMiddleware) WithResourceAccess(resources ResourceAccess) *JWTAuthMiddleware {
	m.resources = resources
	return m
}

func NewJWTAuthMiddleware(auth sysauthservice.SysAuthService) *JWTAuthMiddleware {
	return &JWTAuthMiddleware{
		auth: auth,
	}
}

// writeJSONResponse 统一封装JSON响应，不包含HTTP状态码
func writeJSONResponse(w http.ResponseWriter, code int64, message string, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(Response{
		Code:    code,
		Message: message,
		Data:    data,
	})
}

func (m *JWTAuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		//// 获取Token并校验
		token := r.Header.Get("Authorization")
		if token == "" {
			// 从 url 中获取 token
			urlToken := r.URL.Query().Get("token")
			// 组装Bearer
			token = "Bearer " + urlToken
		}

		res, err := m.auth.VerifyToken(r.Context(), &sysauthservice.VerifyTokenRequest{
			Token: token,
		})
		if err != nil {
			w.WriteHeader(http.StatusOK) // 200
			writeJSONResponse(w, 100002, "Token验证失败: "+err.Error(), nil)
			return
		}
		if res == nil {
			w.WriteHeader(http.StatusOK)
			writeJSONResponse(w, 100002, "Token验证失败", nil)
			return
		}
		if !res.IsValid {
			w.WriteHeader(http.StatusOK) // 200
			writeJSONResponse(w, res.ErrorType, res.ErrorMessage, nil)
			return
		}

		ctx := context.WithValue(r.Context(), "username", res.Username)
		ctx = context.WithValue(ctx, "userId", res.UserId)
		ctx = context.WithValue(ctx, "roles", res.Roles)
		ctx = context.WithValue(ctx, "nickName", res.NickName)
		ctx = context.WithValue(ctx, "uuid", res.Uuid)
		if platformId := parsePlatformID(r.Header.Get("X-Platform-Id")); platformId > 0 {
			ctx = context.WithValue(ctx, "platformId", platformId)
		}

		// 获取请求的 url
		url := r.URL.Path
		// 获取 请求的方法
		method := r.Method
		isAuth, authErr := m.auth.ApiAuth(ctx, &sysauthservice.ApiAuthRequest{
			ApiPath:   url,
			ApiMethod: method,
			UserRoles: res.Roles,
		})
		if authErr != nil {
			w.WriteHeader(http.StatusUnauthorized) // 401
			writeJSONResponse(w, 100999, authErr.Error(), nil)
			return
		}
		if isAuth == nil || !isAuth.IsAuth {
			w.WriteHeader(http.StatusUnauthorized) // 401
			writeJSONResponse(w, 100999, "无权限访问!", nil)
			return
		}
		request := r.WithContext(ctx)
		if m.resources != nil {
			request, err = m.resources.Check(request, func(checkCtx context.Context) error {
				current, e := m.auth.VerifyToken(checkCtx, &sysauthservice.VerifyTokenRequest{Token: token})
				if e != nil || current == nil || !current.IsValid || current.UserId != res.UserId {
					return errors.New("session revoked")
				}
				permission, e := m.auth.ApiAuth(checkCtx, &sysauthservice.ApiAuthRequest{ApiPath: url, ApiMethod: method, UserRoles: current.Roles})
				if e != nil || permission == nil || !permission.IsAuth {
					return errors.New("permission revoked")
				}
				return nil
			})
			if err != nil {
				w.WriteHeader(http.StatusForbidden)
				writeJSONResponse(w, 100999, err.Error(), nil)
				return
			}
		}
		next(w, request)
	}
}

func parsePlatformID(raw string) uint64 {
	if raw == "" {
		return 0
	}
	id, _ := strconv.ParseUint(raw, 10, 64)
	return id
}
