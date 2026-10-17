package gitprovider

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// restProvider keeps the public Git providers on one small adapter. GitLab
// has a dedicated implementation because its webhook payloads are richer;
// this adapter makes GitHub, Gitee and Gitea usable for repository delivery.
type restProvider struct {
	typeName string
	baseURL  string
	apiBase  string
	token    string
	client   *http.Client
}

func newRESTProvider(cfg *Config, providerType string) *restProvider {
	base := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if base == "" {
		switch providerType {
		case "github":
			base = "https://api.github.com"
		case "gitee":
			base = "https://gitee.com"
		default:
			base = "http://gitea.local"
		}
	}
	apiBase := base
	switch providerType {
	case "gitea":
		apiBase += "/api/v1"
	case "gitee":
		apiBase += "/api/v5"
	case "github":
		// GitHub Enterprise accepts either an API URL or the web URL.
		if !strings.HasSuffix(apiBase, "/api/v3") && strings.Contains(apiBase, "github.com") == false {
			apiBase += "/api/v3"
		}
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = internalAwareProxy
	return &restProvider{typeName: providerType, baseURL: base, apiBase: strings.TrimRight(apiBase, "/"), token: cfg.Token,
		client: &http.Client{Timeout: 30 * time.Second, Transport: transport}}
}

// internalAwareProxy keeps public provider traffic compatible with a corporate
// proxy while preventing RFC1918/loopback Git services from being sent to a
// proxy that usually cannot route private network addresses.
func internalAwareProxy(req *http.Request) (*url.URL, error) {
	host := req.URL.Hostname()
	if ip := net.ParseIP(host); ip != nil && (ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast()) {
		return nil, nil
	}
	return http.ProxyFromEnvironment(req)
}

func (p *restProvider) Type() string             { return p.typeName }
func (p *restProvider) GetWebhookSecret() string { return "" }

type restRepositoryRow struct {
	ID            interface{} `json:"id"`
	Name          string      `json:"name"`
	FullName      string      `json:"full_name"`
	CloneURL      string      `json:"clone_url"`
	HTMLURL       string      `json:"html_url"`
	DefaultBranch string      `json:"default_branch"`
	Private       bool        `json:"private"`
	Owner         struct {
		Login string `json:"login"`
	} `json:"owner"`
}

func (p *restProvider) ListRepositories(ctx context.Context) ([]Repository, error) {
	path := "/user/repos?per_page=100"
	if p.typeName == "gitea" {
		// Gitea uses limit/page and needs organization_member affiliation to
		// include repositories granted through an organization.
		path = "/user/repos?limit=100&page=1&affiliation=owner,collaborator,organization_member"
	}
	rows, err := readAllPages[restRepositoryRow](ctx, path, p.getJSON)
	if err != nil {
		return nil, err
	}
	if p.typeName == "gitea" && len(rows) == 0 {
		// Older Gitea versions may omit organization repositories from
		// /user/repos even when the token can read them. Best-effort org
		// discovery keeps the primary connection check independent of this API.
		type organization struct {
			Username string `json:"username"`
			Name     string `json:"name"`
		}
		if orgs, err := readAllPages[organization](ctx, "/user/orgs?limit=100", p.getJSON); err == nil {
			for _, org := range orgs {
				orgName := org.Username
				if orgName == "" {
					orgName = org.Name
				}
				if orgName == "" {
					continue
				}
				if orgRows, err := readAllPages[restRepositoryRow](ctx, "/orgs/"+url.PathEscape(orgName)+"/repos?limit=100", p.getJSON); err == nil {
					rows = append(rows, orgRows...)
				}
			}
		}
	}
	items := make([]Repository, 0, len(rows))
	seen := map[string]bool{}
	for _, row := range rows {
		clone := row.CloneURL
		if clone == "" {
			clone = row.HTMLURL
		}
		fullName := row.FullName
		if fullName == "" && row.Owner.Login != "" {
			fullName = row.Owner.Login + "/" + row.Name
		}
		key := fullName
		if key == "" {
			key = fmt.Sprint(row.ID)
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		items = append(items, Repository{ID: fmt.Sprint(row.ID), Name: row.Name, FullName: fullName,
			CloneURL: clone, DefaultBranch: row.DefaultBranch, Private: row.Private})
	}
	return items, nil
}

func (p *restProvider) ListBranches(ctx context.Context, repoPath string) ([]Branch, error) {
	var rows []struct {
		Name   string `json:"name"`
		Commit struct {
			SHA string `json:"sha"`
		} `json:"commit"`
		Default bool `json:"default"`
	}
	if err := p.getJSON(ctx, p.repoPath(repoPath)+"/branches?per_page=100", &rows); err != nil {
		return nil, err
	}
	result := make([]Branch, 0, len(rows))
	for _, row := range rows {
		result = append(result, Branch{Name: row.Name, Commit: row.Commit.SHA, IsDefault: row.Default})
	}
	return result, nil
}

func (p *restProvider) ListTags(ctx context.Context, repoPath string) ([]Tag, error) {
	var rows []struct {
		Name   string `json:"name"`
		Commit struct {
			SHA string `json:"sha"`
		} `json:"commit"`
	}
	if err := p.getJSON(ctx, p.repoPath(repoPath)+"/tags?per_page=100", &rows); err != nil {
		return nil, err
	}
	result := make([]Tag, 0, len(rows))
	for _, row := range rows {
		result = append(result, Tag{Name: row.Name, Commit: row.Commit.SHA})
	}
	return result, nil
}

func (p *restProvider) GetCommit(ctx context.Context, repoPath, ref string) (*Commit, error) {
	var row struct {
		SHA    string `json:"sha"`
		Commit struct {
			Message string `json:"message"`
			Author  struct {
				Name  string `json:"name"`
				Email string `json:"email"`
				Date  string `json:"date"`
			} `json:"author"`
		} `json:"commit"`
		HTMLURL string `json:"html_url"`
	}
	if err := p.getJSON(ctx, p.repoPath(repoPath)+"/commits/"+url.PathEscape(ref), &row); err != nil {
		return nil, err
	}
	return &Commit{ID: row.SHA, ShortID: shortID(row.SHA), Message: row.Commit.Message, AuthorName: row.Commit.Author.Name, AuthorEmail: row.Commit.Author.Email, URL: row.HTMLURL}, nil
}

func (p *restProvider) ListCommits(ctx context.Context, repoPath, branch string, limit int) ([]Commit, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	var rows []struct {
		SHA    string `json:"sha"`
		Commit struct {
			Message string `json:"message"`
			Author  struct {
				Name  string `json:"name"`
				Email string `json:"email"`
			} `json:"author"`
		} `json:"commit"`
		HTMLURL string `json:"html_url"`
	}
	path := p.repoPath(repoPath) + "/commits?sha=" + url.QueryEscape(branch) + fmt.Sprintf("&per_page=%d", limit)
	if err := p.getJSON(ctx, path, &rows); err != nil {
		return nil, err
	}
	result := make([]Commit, 0, len(rows))
	for _, row := range rows {
		result = append(result, Commit{ID: row.SHA, ShortID: shortID(row.SHA), Message: row.Commit.Message, AuthorName: row.Commit.Author.Name, AuthorEmail: row.Commit.Author.Email, URL: row.HTMLURL})
	}
	return result, nil
}

func (p *restProvider) GetFileContent(ctx context.Context, repoPath, filePath, ref string) ([]byte, error) {
	var row struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	path := p.repoPath(repoPath) + "/contents/" + strings.TrimPrefix(url.PathEscape(filePath), "%2F") + "?ref=" + url.QueryEscape(ref)
	if err := p.getJSON(ctx, path, &row); err != nil {
		return nil, err
	}
	if row.Encoding != "" && row.Encoding != "base64" {
		return nil, ErrInvalidResponse
	}
	content := strings.ReplaceAll(row.Content, "\n", "")
	data, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (p *restProvider) ListRepoFiles(ctx context.Context, repoPath, ref, dirPath string) ([]RepoFile, error) {
	path := p.repoPath(repoPath) + "/contents/" + strings.TrimPrefix(url.PathEscape(strings.Trim(dirPath, "/")), "%2F") + "?ref=" + url.QueryEscape(ref)
	var rows []struct {
		Name, Path, Type string
		Size             int64 `json:"size"`
	}
	if err := p.getJSON(ctx, path, &rows); err != nil {
		return nil, err
	}
	result := make([]RepoFile, 0, len(rows))
	for _, row := range rows {
		result = append(result, RepoFile{Name: row.Name, Path: row.Path, Type: row.Type, Size: row.Size})
	}
	return result, nil
}

func (p *restProvider) ParseWebhookEvent(*http.Request) (*WebhookEvent, error) {
	return nil, ErrUnsupportedProvider
}

func (p *restProvider) repoPath(repoPath string) string {
	parts := strings.Split(strings.Trim(repoPath, "/"), "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return p.apiBase + "/repos/" + strings.Join(parts, "/")
}

func (p *restProvider) getJSON(ctx context.Context, path string, target interface{}) error {
	reqURL := path
	if !strings.HasPrefix(path, "http://") && !strings.HasPrefix(path, "https://") {
		reqURL = p.apiBase + "/" + strings.TrimLeft(path, "/")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if p.token != "" {
		req.Header.Set("Authorization", "token "+p.token)
		req.Header.Set("PRIVATE-TOKEN", p.token)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return ErrNetworkTimeout
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return ErrUnauthorized
	}
	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{Status: resp.StatusCode, Body: string(body)}
	}
	return json.Unmarshal(body, target)
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}
