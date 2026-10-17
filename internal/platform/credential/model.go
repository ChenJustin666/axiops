package credential

import "time"

// Credential 凭据模型
type Credential struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Data        string    `json:"-"` // 列表不返回敏感数据
	ProjectID   int64     `json:"project_id"`
	CreatedBy   int64     `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Keys        []string  `json:"keys,omitempty"`
	Token       string    `json:"token,omitempty"`
}

// CreateCredentialRequest 创建凭据
type CreateCredentialRequest struct {
	Name        string `json:"name" validate:"required,max=64"`
	Type        string `json:"type" validate:"required"` // git_token/harbor_auth/kubeconfig/ssh_key/secret
	Description string `json:"description"`
	Data        string `json:"data" validate:"required"` // JSON 格式的凭据数据
	ProjectID   int64  `json:"project_id"`
}

// UpdateCredentialRequest 更新凭据
type UpdateCredentialRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Data        string `json:"data"`
}

// DeriveSecretRequest creates an independent version. Retained values are
// copied server-side and never returned to the browser.
type DeriveSecretRequest struct {
	Name   string       `json:"name"`
	Token  string       `json:"token"`
	Retain []string     `json:"retain"`
	Values SecretValues `json:"values"`
}

// SecretValues rejects ambiguous JSON values such as null or duplicate keys.
type SecretValues map[string]string
