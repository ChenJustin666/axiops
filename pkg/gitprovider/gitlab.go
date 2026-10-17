package gitprovider

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// gitlabProvider GitLab API v4 实现
type gitlabProvider struct {
	baseURL       string
	token         string
	webhookSecret string
	client        *http.Client
}

func newGitLabProvider(cfg *Config) (Provider, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://gitlab.com"
	}
	baseURL = strings.TrimRight(baseURL, "/")

	return &gitlabProvider{
		baseURL:       baseURL,
		token:         cfg.Token,
		webhookSecret: cfg.WebhookSecret,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (g *gitlabProvider) Type() string             { return "gitlab" }
func (g *gitlabProvider) GetWebhookSecret() string { return g.webhookSecret }

func (g *gitlabProvider) ListRepositories(ctx context.Context) ([]Repository, error) {
	type repositoryRow struct {
		ID            int64  `json:"id"`
		Name          string `json:"name"`
		PathNamespace string `json:"path_with_namespace"`
		HTTPURL       string `json:"http_url_to_repo"`
		DefaultBranch string `json:"default_branch"`
		Visibility    string `json:"visibility"`
	}
	result, err := readAllPages[repositoryRow](ctx, "/api/v4/projects?membership=true&per_page=100&order_by=id&sort=asc", g.doGet)
	if err != nil {
		return nil, err
	}
	items := make([]Repository, 0, len(result))
	for _, repo := range result {
		items = append(items, Repository{ID: fmt.Sprint(repo.ID), Name: repo.Name, FullName: repo.PathNamespace,
			CloneURL: repo.HTTPURL, DefaultBranch: repo.DefaultBranch, Private: repo.Visibility == "private"})
	}
	return items, nil
}

// ListBranches 获取仓库分支列表
func (g *gitlabProvider) ListBranches(ctx context.Context, repoPath string) ([]Branch, error) {
	path := fmt.Sprintf("/api/v4/projects/%s/repository/branches?per_page=100", url.PathEscape(repoPath))
	var result []struct {
		Name    string `json:"name"`
		Default bool   `json:"default"`
		Commit  struct {
			ID        string `json:"id"`
			CreatedAt string `json:"created_at"`
		} `json:"commit"`
	}
	if err := g.doGet(ctx, path, &result); err != nil {
		return nil, err
	}

	branches := make([]Branch, 0, len(result))
	for _, b := range result {
		t, _ := time.Parse(time.RFC3339, b.Commit.CreatedAt)
		branches = append(branches, Branch{
			Name:      b.Name,
			Commit:    b.Commit.ID,
			IsDefault: b.Default,
			UpdatedAt: t,
		})
	}
	return branches, nil
}

// ListTags 获取仓库 Tag 列表
func (g *gitlabProvider) ListTags(ctx context.Context, repoPath string) ([]Tag, error) {
	path := fmt.Sprintf("/api/v4/projects/%s/repository/tags?per_page=100", url.PathEscape(repoPath))
	var result []struct {
		Name    string `json:"name"`
		Message string `json:"message"`
		Commit  struct {
			ID        string `json:"id"`
			CreatedAt string `json:"created_at"`
		} `json:"commit"`
	}
	if err := g.doGet(ctx, path, &result); err != nil {
		return nil, err
	}

	tags := make([]Tag, 0, len(result))
	for _, t := range result {
		created, _ := time.Parse(time.RFC3339, t.Commit.CreatedAt)
		tags = append(tags, Tag{
			Name:      t.Name,
			Commit:    t.Commit.ID,
			Message:   t.Message,
			CreatedAt: created,
		})
	}
	return tags, nil
}

// GetCommit 获取指定 commit 信息
func (g *gitlabProvider) GetCommit(ctx context.Context, repoPath, ref string) (*Commit, error) {
	path := fmt.Sprintf("/api/v4/projects/%s/repository/commits/%s", url.PathEscape(repoPath), url.PathEscape(ref))
	var result struct {
		ID          string `json:"id"`
		ShortID     string `json:"short_id"`
		Title       string `json:"title"`
		Message     string `json:"message"`
		AuthorName  string `json:"author_name"`
		AuthorEmail string `json:"author_email"`
		CreatedAt   string `json:"created_at"`
		WebURL      string `json:"web_url"`
	}
	if err := g.doGet(ctx, path, &result); err != nil {
		return nil, err
	}

	ts, _ := time.Parse(time.RFC3339, result.CreatedAt)
	commit := &Commit{
		ID:          result.ID,
		ShortID:     result.ShortID,
		Message:     result.Message,
		AuthorName:  result.AuthorName,
		AuthorEmail: result.AuthorEmail,
		Timestamp:   ts,
		URL:         result.WebURL,
	}

	// 获取 diff files
	diffPath := fmt.Sprintf("/api/v4/projects/%s/repository/commits/%s/diff", url.PathEscape(repoPath), url.PathEscape(ref))
	var diffs []struct {
		NewPath     string `json:"new_path"`
		OldPath     string `json:"old_path"`
		NewFile     bool   `json:"new_file"`
		RenamedFile bool   `json:"renamed_file"`
		DeletedFile bool   `json:"deleted_file"`
	}
	if err := g.doGet(ctx, diffPath, &diffs); err == nil {
		for _, d := range diffs {
			action := "modified"
			if d.NewFile {
				action = "added"
			} else if d.DeletedFile {
				action = "deleted"
			} else if d.RenamedFile {
				action = "renamed"
			}
			commit.DiffFiles = append(commit.DiffFiles, DiffFile{
				Action:  action,
				Path:    d.NewPath,
				OldPath: d.OldPath,
			})
		}
	}
	return commit, nil
}

// ListCommits 获取 commit 列表
func (g *gitlabProvider) ListCommits(ctx context.Context, repoPath, branch string, limit int) ([]Commit, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	path := fmt.Sprintf("/api/v4/projects/%s/repository/commits?ref_name=%s&per_page=%d",
		url.PathEscape(repoPath), url.QueryEscape(branch), limit)
	var result []struct {
		ID          string `json:"id"`
		ShortID     string `json:"short_id"`
		Message     string `json:"message"`
		AuthorName  string `json:"author_name"`
		AuthorEmail string `json:"author_email"`
		CreatedAt   string `json:"created_at"`
		WebURL      string `json:"web_url"`
	}
	if err := g.doGet(ctx, path, &result); err != nil {
		return nil, err
	}

	commits := make([]Commit, 0, len(result))
	for _, c := range result {
		ts, _ := time.Parse(time.RFC3339, c.CreatedAt)
		commits = append(commits, Commit{
			ID:          c.ID,
			ShortID:     c.ShortID,
			Message:     c.Message,
			AuthorName:  c.AuthorName,
			AuthorEmail: c.AuthorEmail,
			Timestamp:   ts,
			URL:         c.WebURL,
		})
	}
	return commits, nil
}

// GetFileContent 获取文件内容
func (g *gitlabProvider) GetFileContent(ctx context.Context, repoPath, filePath, ref string) ([]byte, error) {
	path := fmt.Sprintf("/api/v4/projects/%s/repository/files/%s/raw?ref=%s",
		url.PathEscape(repoPath), url.PathEscape(filePath), url.QueryEscape(ref))
	return g.doGetRaw(ctx, path)
}

// ListRepoFiles 列出仓库文件（某目录下）
func (g *gitlabProvider) ListRepoFiles(ctx context.Context, repoPath, ref, dirPath string) ([]RepoFile, error) {
	path := fmt.Sprintf("/api/v4/projects/%s/repository/tree?ref=%s&path=%s&per_page=100",
		url.PathEscape(repoPath), url.QueryEscape(ref), url.QueryEscape(dirPath))
	type treeEntry struct {
		Name string `json:"name"`
		Path string `json:"path"`
		Type string `json:"type"` // blob/tree
		Mode string `json:"mode"`
	}
	result, err := readAllPages[treeEntry](ctx, path, g.doGet)
	if err != nil {
		return nil, err
	}

	files := make([]RepoFile, 0, len(result))
	for _, f := range result {
		fileType := f.Type
		if f.Mode == "120000" {
			fileType = "symlink"
		} else if f.Type == "tree" {
			fileType = "dir"
		} else if f.Type == "blob" {
			fileType = "file"
		}
		files = append(files, RepoFile{
			Name: f.Name,
			Path: f.Path,
			Type: fileType,
		})
	}
	return files, nil
}

// ParseWebhookEvent 解析 GitLab Webhook
func (g *gitlabProvider) ParseWebhookEvent(r *http.Request) (*WebhookEvent, error) {
	// 验证 Token
	if g.webhookSecret != "" {
		token := r.Header.Get("X-Gitlab-Token")
		if !hmac.Equal([]byte(token), []byte(g.webhookSecret)) {
			return nil, ErrWebhookSignature
		}
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("gitprovider: 读取 webhook body 失败: %w", err)
	}

	eventType := r.Header.Get("X-Gitlab-Event")
	event := &WebhookEvent{
		Source:     "gitlab",
		RawPayload: body,
	}

	switch eventType {
	case "Push Hook":
		return g.parsePushEvent(body, event)
	case "Tag Push Hook":
		return g.parseTagPushEvent(body, event)
	case "Merge Request Hook":
		return g.parseMergeRequestEvent(body, event)
	default:
		event.EventType = eventType
		return event, nil
	}
}

func (g *gitlabProvider) parsePushEvent(body []byte, event *WebhookEvent) (*WebhookEvent, error) {
	var payload struct {
		Ref       string `json:"ref"`
		Before    string `json:"before"`
		After     string `json:"after"`
		UserName  string `json:"user_name"`
		UserEmail string `json:"user_email"`
		Project   struct {
			PathWithNamespace string `json:"path_with_namespace"`
			WebURL            string `json:"web_url"`
			HTTPURL           string `json:"http_url"`
			SSHURL            string `json:"ssh_url_to_repo"`
			Name              string `json:"name"`
			Namespace         string `json:"namespace"`
		} `json:"project"`
		Commits []struct {
			ID        string `json:"id"`
			Message   string `json:"message"`
			Timestamp string `json:"timestamp"`
			Author    struct {
				Name  string `json:"name"`
				Email string `json:"email"`
			} `json:"author"`
			URL      string   `json:"url"`
			Added    []string `json:"added"`
			Modified []string `json:"modified"`
			Removed  []string `json:"removed"`
		} `json:"commits"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("gitprovider: 解析 push event 失败: %w", err)
	}

	event.EventType = "push"
	event.Ref = payload.Ref
	event.Branch = extractBranch(payload.Ref)
	event.Before = payload.Before
	event.After = payload.After
	event.UserName = payload.UserName
	event.UserEmail = payload.UserEmail
	event.RepoURL = payload.Project.HTTPURL
	event.RepoHTTP = payload.Project.WebURL
	event.RepoName = payload.Project.Name
	event.Namespace = payload.Project.Namespace

	for _, c := range payload.Commits {
		event.Commits = append(event.Commits, WebhookCommit{
			ID:        c.ID,
			Message:   c.Message,
			Timestamp: c.Timestamp,
			Author:    c.Author.Name,
			Email:     c.Author.Email,
			URL:       c.URL,
			Added:     c.Added,
			Modified:  c.Modified,
			Removed:   c.Removed,
		})
	}
	return event, nil
}

func (g *gitlabProvider) parseTagPushEvent(body []byte, event *WebhookEvent) (*WebhookEvent, error) {
	var payload struct {
		Ref      string `json:"ref"`
		Before   string `json:"before"`
		After    string `json:"after"`
		UserName string `json:"user_name"`
		Project  struct {
			PathWithNamespace string `json:"path_with_namespace"`
			HTTPURL           string `json:"http_url"`
			WebURL            string `json:"web_url"`
			Name              string `json:"name"`
			Namespace         string `json:"namespace"`
		} `json:"project"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("gitprovider: 解析 tag push event 失败: %w", err)
	}

	event.EventType = "tag_push"
	event.Ref = payload.Ref
	event.Branch = extractTag(payload.Ref)
	event.Before = payload.Before
	event.After = payload.After
	event.UserName = payload.UserName
	event.RepoURL = payload.Project.HTTPURL
	event.RepoHTTP = payload.Project.WebURL
	event.RepoName = payload.Project.Name
	event.Namespace = payload.Project.Namespace
	return event, nil
}

func (g *gitlabProvider) parseMergeRequestEvent(body []byte, event *WebhookEvent) (*WebhookEvent, error) {
	var payload struct {
		User struct {
			Name string `json:"name"`
		} `json:"user"`
		Project struct {
			PathWithNamespace string `json:"path_with_namespace"`
			HTTPURL           string `json:"http_url"`
			WebURL            string `json:"web_url"`
			Name              string `json:"name"`
			Namespace         string `json:"namespace"`
		} `json:"project"`
		ObjectAttributes struct {
			IID          int64  `json:"iid"`
			Title        string `json:"title"`
			State        string `json:"state"`
			SourceBranch string `json:"source_branch"`
			TargetBranch string `json:"target_branch"`
			URL          string `json:"url"`
			LastCommit   struct {
				ID string `json:"id"`
			} `json:"last_commit"`
		} `json:"object_attributes"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("gitprovider: 解析 merge request event 失败: %w", err)
	}

	event.EventType = "merge_request"
	event.UserName = payload.User.Name
	event.RepoURL = payload.Project.HTTPURL
	event.RepoHTTP = payload.Project.WebURL
	event.RepoName = payload.Project.Name
	event.Namespace = payload.Project.Namespace
	event.Branch = payload.ObjectAttributes.SourceBranch
	event.After = payload.ObjectAttributes.LastCommit.ID
	event.MergeRequest = &MergeRequestInfo{
		ID:           payload.ObjectAttributes.IID,
		Title:        payload.ObjectAttributes.Title,
		State:        payload.ObjectAttributes.State,
		SourceBranch: payload.ObjectAttributes.SourceBranch,
		TargetBranch: payload.ObjectAttributes.TargetBranch,
		URL:          payload.ObjectAttributes.URL,
	}
	return event, nil
}

// ==================== HTTP 工具方法 ====================

func (g *gitlabProvider) doGet(ctx context.Context, path string, result interface{}) error {
	body, err := g.doGetRaw(ctx, path)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, result)
}

func (g *gitlabProvider) doGetRaw(ctx context.Context, path string) ([]byte, error) {
	reqURL := g.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("PRIVATE-TOKEN", g.token)

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, ErrNetworkTimeout
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return body, nil
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, ErrUnauthorized
	case http.StatusNotFound:
		return nil, ErrNotFound
	case http.StatusTooManyRequests:
		return nil, ErrRateLimit
	default:
		return nil, fmt.Errorf("gitprovider: GitLab API 返回 %d: %s", resp.StatusCode, string(body))
	}
}

// ==================== 辅助函数 ====================

func extractBranch(ref string) string {
	return strings.TrimPrefix(ref, "refs/heads/")
}

func extractTag(ref string) string {
	return strings.TrimPrefix(ref, "refs/tags/")
}

// verifyHMACSHA256 验证 HMAC-SHA256 签名（GitHub 用）
func verifyHMACSHA256(secret string, body []byte, signature string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}
