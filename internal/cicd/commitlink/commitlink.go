// Package commitlink resolves repository links from immutable build inputs.
package commitlink

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"path"
	"sort"
	"strings"
)

type Snapshot struct {
	RepositoryURL  string `json:"repository_url"`
	CodeSourceID   int64  `json:"code_source_id"`
	CodeSourceType string `json:"code_source_type"`
}

type Record struct {
	ProjectID int64
	Commit    string
	URL       string
	Snapshot  Snapshot
}

func ParseSnapshot(raw *string) Snapshot {
	var snapshot Snapshot
	if raw != nil {
		if json.Unmarshal([]byte(*raw), &snapshot) != nil {
			return Snapshot{}
		}
	}
	return snapshot
}

// Fill only enriches the response. It never changes saved snapshots, fabricates
// change history, or substitutes the service's current repository. New records
// keep their stored links. Old records need at most one batch code-source lookup.
func Fill(ctx context.Context, db *sql.DB, records []Record) error {
	ids := map[int64]bool{}
	for i := range records {
		r := &records[i]
		if r.URL != "" || !validCommit(r.Commit) {
			continue
		}
		r.URL = BrowseURL(r.Snapshot.RepositoryURL, r.Snapshot.CodeSourceType, "", r.Commit)
		if r.URL == "" && r.Snapshot.CodeSourceType == "" && r.Snapshot.CodeSourceID > 0 && httpURL(r.Snapshot.RepositoryURL) != nil {
			ids[r.Snapshot.CodeSourceID] = true
		}
	}
	if len(ids) == 0 {
		return nil
	}
	keys := make([]int64, 0, len(ids))
	for id := range ids {
		keys = append(keys, id)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	args := make([]interface{}, len(keys))
	for i, id := range keys {
		args[i] = id
	}
	rows, err := db.QueryContext(ctx, `SELECT id,project_id,type,url FROM code_source WHERE id IN (`+strings.TrimRight(strings.Repeat("?,", len(keys)), ",")+`)`, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	type source struct {
		project   int64
		kind, url string
	}
	sources := map[int64]source{}
	for rows.Next() {
		var id int64
		var item source
		if err := rows.Scan(&id, &item.project, &item.kind, &item.url); err != nil {
			return err
		}
		sources[id] = item
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for i := range records {
		r := &records[i]
		if r.URL != "" || r.Snapshot.CodeSourceType != "" {
			continue
		}
		s, ok := sources[r.Snapshot.CodeSourceID]
		// Use the referenced provider only while it still describes this origin
		// and server base path. Changed/deleted/cross-project sources fail closed.
		if ok && (s.project == 0 || s.project == r.ProjectID) && sameServer(r.Snapshot.RepositoryURL, s.url) {
			r.URL = BrowseURL(r.Snapshot.RepositoryURL, s.kind, "", r.Commit)
		}
	}
	return nil
}

func validCommit(commit string) bool {
	if len(commit) != 40 && len(commit) != 64 {
		return false
	}
	_, err := hex.DecodeString(commit)
	return err == nil
}

func httpURL(raw string) *url.URL {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
		return nil
	}
	return u
}

func sameServer(repository, server string) bool {
	repo, source := httpURL(repository), httpURL(server)
	if repo == nil || source == nil || repo.Scheme != source.Scheme || !strings.EqualFold(repo.Hostname(), source.Hostname()) {
		return false
	}
	port := func(u *url.URL) string {
		if u.Port() != "" {
			return u.Port()
		}
		if u.Scheme == "https" {
			return "443"
		}
		return "80"
	}
	if port(repo) != port(source) {
		return false
	}
	base := strings.TrimRight(path.Clean("/"+source.Path), "/")
	return strings.HasPrefix(path.Clean("/"+repo.Path), base+"/")
}

func BrowseURL(repository, kind, from, to string) string {
	if !validCommit(to) || (from != "" && !validCommit(from)) {
		return ""
	}
	u := httpURL(repository)
	if u == nil {
		return ""
	}
	u.User, u.RawQuery, u.Fragment, u.RawPath = nil, "", "", ""
	u.Path = strings.TrimSuffix(strings.TrimRight(u.Path, "/"), ".git")
	if u.Path == "" || u.Path == "/" {
		return ""
	}
	if kind == "" {
		switch strings.ToLower(u.Hostname()) {
		case "github.com":
			kind = "github"
		case "gitlab.com":
			kind = "gitlab"
		case "gitee.com":
			kind = "gitee"
		}
	}
	switch kind {
	case "gitlab", "github", "gitea", "gitee":
	default:
		return ""
	}
	prefix := "/"
	if kind == "gitlab" {
		prefix = "/-/"
	}
	if from == "" {
		u.Path += prefix + "commit/" + to
	} else {
		u.Path += prefix + "compare/" + from + "..." + to
	}
	if len(u.String()) > 2048 {
		return ""
	}
	return u.String()
}
