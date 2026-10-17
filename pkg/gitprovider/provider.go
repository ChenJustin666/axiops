// Package gitprovider 提供统一的 Git 平台操作接口
// 支持 GitLab / GitHub / Gitee / Gitea
package gitprovider

import (
	"context"
	"net/http"
	"time"
)

// Provider Git 平台操作统一接口
type Provider interface {
	// 基本信息
	Type() string // gitlab/github/gitee/gitea
	ListRepositories(ctx context.Context) ([]Repository, error)

	// 仓库操作
	ListBranches(ctx context.Context, repoPath string) ([]Branch, error)
	ListTags(ctx context.Context, repoPath string) ([]Tag, error)
	GetCommit(ctx context.Context, repoPath, ref string) (*Commit, error)
	ListCommits(ctx context.Context, repoPath, branch string, limit int) ([]Commit, error)
	GetFileContent(ctx context.Context, repoPath, filePath, ref string) ([]byte, error)
	ListRepoFiles(ctx context.Context, repoPath, ref, path string) ([]RepoFile, error)

	// Webhook
	ParseWebhookEvent(r *http.Request) (*WebhookEvent, error)
	GetWebhookSecret() string
}

// Repository is the safe repository metadata exposed to the UI. Credentials
// and private tokens never enter this model.
type Repository struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	CloneURL      string `json:"clone_url"`
	DefaultBranch string `json:"default_branch"`
	Private       bool   `json:"private"`
}

// Branch Git 分支
type Branch struct {
	Name      string    `json:"name"`
	Commit    string    `json:"commit"`     // 最新 commit SHA
	IsDefault bool      `json:"is_default"` // 是否默认分支
	UpdatedAt time.Time `json:"updated_at"`
}

// Tag Git Tag
type Tag struct {
	Name      string    `json:"name"`
	Commit    string    `json:"commit"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

// Commit Git 提交
type Commit struct {
	ID          string     `json:"id"`       // full SHA
	ShortID     string     `json:"short_id"` // 前 8 位
	Message     string     `json:"message"`
	AuthorName  string     `json:"author_name"`
	AuthorEmail string     `json:"author_email"`
	Timestamp   time.Time  `json:"timestamp"`
	URL         string     `json:"url"` // Web 查看链接
	DiffFiles   []DiffFile `json:"diff_files,omitempty"`
}

// DiffFile 变更文件
type DiffFile struct {
	Action  string `json:"action"` // added/modified/deleted/renamed
	Path    string `json:"path"`
	OldPath string `json:"old_path,omitempty"` // renamed 时有值
}

// RepoFile 仓库文件
type RepoFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"` // file/dir
	Size int64  `json:"size"`
}

// WebhookEvent 统一的 Webhook 事件
type WebhookEvent struct {
	// 事件类型
	Source    string `json:"source"`     // gitlab/github/gitee/gitea
	EventType string `json:"event_type"` // push/tag_push/merge_request/pull_request

	// 仓库信息
	RepoURL   string `json:"repo_url"`  // clone URL
	RepoHTTP  string `json:"repo_http"` // HTTP URL
	RepoName  string `json:"repo_name"`
	Namespace string `json:"namespace"` // owner/group

	// 分支/提交信息
	Ref    string `json:"ref"`    // refs/heads/main
	Branch string `json:"branch"` // main (提取出的分支名)
	Before string `json:"before"` // 推送前 SHA
	After  string `json:"after"`  // 推送后 SHA (最新 commit)

	// 提交者
	UserName  string `json:"user_name"`
	UserEmail string `json:"user_email"`

	// 提交列表
	Commits []WebhookCommit `json:"commits"`

	// Merge/Pull Request（合并请求事件时有值）
	MergeRequest *MergeRequestInfo `json:"merge_request,omitempty"`

	// 原始 Payload
	RawPayload []byte `json:"-"`
}

// WebhookCommit Webhook 中的提交信息
type WebhookCommit struct {
	ID        string   `json:"id"`
	Message   string   `json:"message"`
	Timestamp string   `json:"timestamp"`
	Author    string   `json:"author"`
	Email     string   `json:"email"`
	URL       string   `json:"url"`
	Added     []string `json:"added"`
	Modified  []string `json:"modified"`
	Removed   []string `json:"removed"`
}

// MergeRequestInfo 合并请求信息
type MergeRequestInfo struct {
	ID           int64  `json:"id"`
	Title        string `json:"title"`
	State        string `json:"state"` // opened/closed/merged
	SourceBranch string `json:"source_branch"`
	TargetBranch string `json:"target_branch"`
	URL          string `json:"url"`
}

// Config Git Provider 配置
type Config struct {
	Type          string `json:"type"`     // gitlab/github/gitee/gitea
	BaseURL       string `json:"base_url"` // API 地址（自建 GitLab 时必填）
	Token         string `json:"token"`    // Personal Access Token 或 OAuth Token
	WebhookSecret string `json:"webhook_secret"`
}

// New 创建 Git Provider 实例
func New(cfg *Config) (Provider, error) {
	switch cfg.Type {
	case "gitlab":
		return newGitLabProvider(cfg)
	case "github":
		return newGitHubProvider(cfg)
	case "gitee":
		return newGiteeProvider(cfg)
	case "gitea":
		return newGiteaProvider(cfg)
	default:
		return nil, ErrUnsupportedProvider
	}
}
