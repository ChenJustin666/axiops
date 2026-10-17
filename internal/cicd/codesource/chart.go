package codesource

import (
	"bytes"
	"context"
	"io/fs"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/ChenJustin666/axiops/pkg/gitprovider"
	"helm.sh/helm/v3/pkg/ignore"
)

// ReadChartFiles freezes the commit before traversing any directories. No
// branch-relative listing is mixed with content from another commit.
func (l *Logic) ReadChartFiles(ctx context.Context, userID, id int64, repoPath, ref, chartPath string) (*RepositorySnapshot, error) {
	root := path.Clean(strings.TrimSpace(chartPath))
	if id <= 0 || strings.TrimSpace(repoPath) == "" || strings.TrimSpace(ref) == "" || strings.TrimSpace(chartPath) == "" || root == ".." || strings.HasPrefix(root, "../") || strings.HasPrefix(root, "/") || strings.ContainsAny(root, "\\:\x00") {
		return nil, ErrInvalid
	}
	provider, err := l.authorizedProvider(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	source, err := l.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	commit, err := provider.GetCommit(ctx, repoPath, ref)
	if err != nil {
		return nil, err
	}
	if commit == nil || strings.TrimSpace(commit.ID) == "" {
		return nil, ErrInvalid
	}
	files, err := readChartTree(ctx, provider, repoPath, commit.ID, root)
	if err != nil {
		return nil, err
	}
	return &RepositorySnapshot{Source: *source, RepoPath: strings.TrimSpace(repoPath), Ref: strings.TrimSpace(ref), CommitID: commit.ID, Files: files}, nil
}

func readChartTree(ctx context.Context, p gitprovider.Provider, repo, commit, root string) ([]RepositoryFile, error) {
	rules := ignore.Empty()
	queue := []string{root}
	files := []RepositoryFile{}
	seen := map[string]bool{}
	dirs, total := 0, 0
	for len(queue) > 0 {
		dir := queue[0]
		queue = queue[1:]
		dirs++
		if dirs > 64 {
			return nil, ErrInvalid
		}
		browse := dir
		if browse == "." {
			browse = ""
		}
		entries, err := p.ListRepoFiles(ctx, repo, commit, browse)
		if err != nil {
			return nil, err
		}
		if dir == root {
			for _, entry := range entries {
				if entry.Path == path.Join(root, ".helmignore") && entry.Type == "file" {
					content, e := p.GetFileContent(ctx, repo, entry.Path, commit)
					if e != nil {
						return nil, e
					}
					if len(content) > 1<<20 {
						return nil, ErrInvalid
					}
					rules, e = ignore.Parse(bytes.NewReader(content))
					if e != nil {
						return nil, ErrInvalid
					}
				}
			}
			rules.AddDefaults()
		}
		for _, entry := range entries {
			name := path.Clean(entry.Path)
			if name != entry.Path || name == "." || path.Dir(name) != dir || strings.ContainsAny(name, "\\:\x00") || seen[name] || strings.Count(name, "/")-strings.Count(root, "/") > 16 {
				return nil, ErrInvalid
			}
			seen[name] = true
			if entry.Name == ".git" || entry.Name == ".DS_Store" {
				continue
			}
			rel := name
			if root != "." {
				rel = strings.TrimPrefix(name, root+"/")
			}
			if rules.Ignore(rel, chartEntryInfo{entry}) {
				continue
			}
			switch entry.Type {
			case "dir":
				queue = append(queue, name)
			case "file":
				if len(files) >= 500 || entry.Size > 1<<20 {
					return nil, ErrInvalid
				}
				content, err := p.GetFileContent(ctx, repo, name, commit)
				if err != nil {
					return nil, err
				}
				total += len(content)
				if len(content) > 1<<20 || total > 4<<20 {
					return nil, ErrInvalid
				}
				files = append(files, RepositoryFile{Path: name, Content: content})
			default:
				return nil, ErrInvalid // Symlinks/submodules cannot silently escape the Chart.
			}
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

type chartEntryInfo struct{ gitprovider.RepoFile }

func (e chartEntryInfo) Name() string { return e.RepoFile.Name }
func (e chartEntryInfo) Size() int64  { return e.RepoFile.Size }
func (e chartEntryInfo) Mode() fs.FileMode {
	if e.IsDir() {
		return fs.ModeDir | 0755
	}
	return 0644
}
func (e chartEntryInfo) ModTime() time.Time { return time.Time{} }
func (e chartEntryInfo) IsDir() bool        { return e.Type == "dir" }
func (e chartEntryInfo) Sys() interface{}   { return nil }
