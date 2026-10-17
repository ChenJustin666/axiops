package build

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var bearerChallengePattern = regexp.MustCompile(`(?i)([a-z][a-z0-9_-]*)\s*=\s*("(?:\\.|[^"])*"|[^,\s]+)`)

// registryAuthForCredential loads and decrypts one platform registry
// credential. It deliberately returns no database data to HTTP handlers.
func (l *Logic) registryAuthForCredential(ctx context.Context, credentialID int64, baseURL string, namespaces []string) (*RegistryAuth, error) {
	if credentialID <= 0 {
		return nil, errors.New("请选择镜像仓库凭据")
	}
	if l.decryptor == nil {
		return nil, errors.New("平台凭据解密器未配置")
	}
	var credentialType string
	var encrypted sql.NullString
	if err := l.repo.db.QueryRowContext(ctx, `SELECT type,data FROM credential WHERE id=? AND project_id=0`, credentialID).Scan(&credentialType, &encrypted); err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("镜像仓库凭据不存在或不是平台凭据")
		}
		return nil, fmt.Errorf("读取镜像仓库凭据失败: %w", err)
	}
	if credentialType != "harbor_auth" && credentialType != "registry_auth" {
		return nil, errors.New("凭据类型不是镜像仓库凭据")
	}
	if !encrypted.Valid || strings.TrimSpace(encrypted.String) == "" {
		return nil, errors.New("镜像仓库凭据为空")
	}
	plaintext, err := l.decryptor.Decrypt(encrypted.String)
	if err != nil {
		return nil, errors.New("镜像仓库凭据解密失败，请重新创建凭据")
	}
	var value struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Token    string `json:"token"`
	}
	if err := json.Unmarshal([]byte(plaintext), &value); err != nil {
		return nil, errors.New("镜像仓库凭据格式无效")
	}
	value.Username = strings.TrimSpace(value.Username)
	value.Token = strings.TrimSpace(value.Token)
	if value.Username == "" && value.Token == "" {
		return nil, errors.New("镜像仓库凭据缺少用户名或访问令牌")
	}
	return &RegistryAuth{BaseURL: baseURL, Namespaces: namespaces, Username: value.Username, Password: value.Password, Token: value.Token}, nil
}

type registrySession struct {
	authorization string
}

// testRegistryConnectivity checks the Docker Registry API before a catalog
// entry is persisted. The client supports both Basic auth and the standard
// Bearer challenge used by ACR, Harbor and distribution registries.
func testRegistryConnectivity(ctx context.Context, rawURL string, auth *RegistryAuth) error {
	base, err := normalizedRegistryURL(rawURL)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
		return errors.New("镜像仓库不允许重定向")
	}}
	session, status, challenge, err := registryRequest(ctx, client, base+"/v2/", auth, "")
	if err != nil {
		return fmt.Errorf("无法连接镜像仓库: %w", err)
	}
	if status >= 200 && status < 300 {
		return verifyRegistryNamespaces(ctx, client, base, auth, session)
	}
	if status != http.StatusUnauthorized {
		return registryStatusError(status)
	}

	// A token supplied in the credential can be used directly. Username and
	// password credentials exchange the Bearer challenge for a short-lived token.
	if auth != nil && auth.Token != "" {
		session, status, _, err = registryRequest(ctx, client, base+"/v2/", auth, "Bearer "+auth.Token)
		if err != nil {
			return fmt.Errorf("镜像仓库认证失败: %w", err)
		}
		if status >= 200 && status < 300 {
			return verifyRegistryNamespaces(ctx, client, base, auth, session)
		}
	}
	if challengeScheme(challenge) == "bearer" {
		token, tokenErr := fetchRegistryBearerToken(ctx, client, challenge, auth)
		if tokenErr != nil {
			return fmt.Errorf("镜像仓库认证失败: %w", tokenErr)
		}
		session, status, _, err = registryRequest(ctx, client, base+"/v2/", auth, "Bearer "+token)
		if err != nil {
			return fmt.Errorf("镜像仓库认证失败: %w", err)
		}
		if status >= 200 && status < 300 {
			return verifyRegistryNamespaces(ctx, client, base, auth, session)
		}
	} else if auth != nil && auth.Username != "" {
		session, status, _, err = registryRequest(ctx, client, base+"/v2/", auth, basicAuthorization(auth))
		if err != nil {
			return fmt.Errorf("镜像仓库认证失败: %w", err)
		}
		if status >= 200 && status < 300 {
			return verifyRegistryNamespaces(ctx, client, base, auth, session)
		}
	}
	_ = session
	return registryStatusError(status)
}

func normalizedRegistryURL(raw string) (string, error) {
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "https" && u.Scheme != "http") {
		return "", errors.New("镜像仓库地址无效")
	}
	// The optional path is a namespace hint, not an API base path.
	return u.Scheme + "://" + u.Host, nil
}

func registryRequest(ctx context.Context, client *http.Client, endpoint string, auth *RegistryAuth, authorization string) (registrySession, int, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return registrySession{}, 0, "", err
	}
	effectiveAuthorization := authorization
	if effectiveAuthorization == "" && auth != nil {
		if auth.Token != "" {
			effectiveAuthorization = "Bearer " + auth.Token
		} else if auth.Username != "" {
			effectiveAuthorization = basicAuthorization(auth)
		}
	}
	if effectiveAuthorization != "" {
		req.Header.Set("Authorization", effectiveAuthorization)
	}
	resp, err := client.Do(req)
	if err != nil {
		return registrySession{}, 0, "", err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	return registrySession{authorization: effectiveAuthorization}, resp.StatusCode, resp.Header.Get("WWW-Authenticate"), nil
}

func verifyRegistryNamespaces(ctx context.Context, client *http.Client, base string, auth *RegistryAuth, session registrySession) error {
	if auth == nil || len(auth.Namespaces) == 0 {
		return nil
	}
	for _, namespace := range auth.Namespaces {
		endpoint := base + "/v2/"
		for _, segment := range strings.Split(strings.Trim(namespace, "/"), "/") {
			if segment != "" {
				endpoint += url.PathEscape(segment) + "/"
			}
		}
		endpoint += "_catalog"
		_, status, challenge, err := registryRequest(ctx, client, endpoint, nil, session.authorization)
		if err != nil {
			return fmt.Errorf("命名空间 %q 检查失败: %w", namespace, err)
		}
		if status == http.StatusUnauthorized && challengeScheme(challenge) == "bearer" {
			token, tokenErr := fetchRegistryBearerToken(ctx, client, challenge, auth)
			if tokenErr != nil {
				return fmt.Errorf("命名空间 %q 权限验证失败: %w", namespace, tokenErr)
			}
			_, status, _, err = registryRequest(ctx, client, endpoint, nil, "Bearer "+token)
			if err != nil {
				return fmt.Errorf("命名空间 %q 检查失败: %w", namespace, err)
			}
		}
		// A registry may hide repository names and return 404 even when the
		// authenticated account is valid. 401/403, however, are explicit scope
		// failures and should block saving the catalog entry.
		if status == http.StatusUnauthorized || status == http.StatusForbidden {
			return fmt.Errorf("凭据没有命名空间 %q 的访问权限（HTTP %d）", namespace, status)
		}
		if status != http.StatusNotFound && (status < 200 || status >= 300) {
			return fmt.Errorf("命名空间 %q 检查返回 HTTP %d", namespace, status)
		}
	}
	return nil
}

func fetchRegistryBearerToken(ctx context.Context, client *http.Client, challenge string, auth *RegistryAuth) (string, error) {
	scheme, params := parseAuthChallenge(challenge)
	if scheme != "bearer" || params["realm"] == "" {
		return "", errors.New("仓库未返回有效的 Bearer 认证信息")
	}
	tokenURL, err := url.Parse(params["realm"])
	if err != nil || tokenURL.Scheme != "https" && tokenURL.Scheme != "http" || tokenURL.Host == "" {
		return "", errors.New("仓库认证地址无效")
	}
	query := tokenURL.Query()
	if params["service"] != "" {
		query.Set("service", params["service"])
	}
	if params["scope"] != "" {
		query.Set("scope", params["scope"])
	}
	tokenURL.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL.String(), nil)
	if err != nil {
		return "", err
	}
	if auth != nil && auth.Username != "" {
		req.SetBasicAuth(auth.Username, auth.Password)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", registryStatusError(resp.StatusCode)
	}
	var value struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&value); err != nil {
		return "", errors.New("仓库认证响应格式无效")
	}
	if value.Token == "" {
		value.Token = value.AccessToken
	}
	if value.Token == "" {
		return "", errors.New("仓库认证响应缺少 token")
	}
	return value.Token, nil
}

func parseAuthChallenge(value string) (string, map[string]string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", map[string]string{}
	}
	parts := strings.Fields(value)
	if len(parts) == 0 {
		return "", map[string]string{}
	}
	params := map[string]string{}
	for _, match := range bearerChallengePattern.FindAllStringSubmatch(value, -1) {
		params[strings.ToLower(match[1])] = strings.Trim(match[2], `"`)
	}
	return strings.ToLower(parts[0]), params
}

func challengeScheme(value string) string {
	scheme, _ := parseAuthChallenge(value)
	return scheme
}

func basicAuthorization(auth *RegistryAuth) string {
	if auth == nil || auth.Username == "" {
		return ""
	}
	request, _ := http.NewRequest(http.MethodGet, "http://localhost", nil)
	request.SetBasicAuth(auth.Username, auth.Password)
	return request.Header.Get("Authorization")
}

func registryStatusError(status int) error {
	switch status {
	case http.StatusUnauthorized:
		return errors.New("账号、密码或访问令牌无效（401 Unauthorized）")
	case http.StatusForbidden:
		return errors.New("凭据没有访问镜像仓库的权限（403 Forbidden）")
	case http.StatusNotFound:
		return errors.New("镜像仓库地址不存在（404 Not Found）")
	default:
		if status == 0 {
			return errors.New("镜像仓库无响应")
		}
		return errors.New("镜像仓库返回 HTTP " + strconv.Itoa(status))
	}
}
