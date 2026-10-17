package build

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ChenJustin666/axiops/internal/cicd/commitlink"
)

const gitChangesPrefix = "AXIOPS_GIT_CHANGESET_V1|"

type GitChangeBaseline struct {
	BuildID     int64  `json:"build_id"`
	BuildNumber int    `json:"build_number"`
	Commit      string `json:"commit"`
}

type GitCommitChange struct {
	Commit     string `json:"commit"`
	Author     string `json:"author"`
	Email      string `json:"-"`
	AuthoredAt string `json:"authored_at"`
	Subject    string `json:"subject"`
	URL        string `json:"url,omitempty"`
}

type GitChangeSet struct {
	Status           string            `json:"status"` // complete, first_build, unavailable
	Head             string            `json:"head"`
	Latest           *GitCommitChange  `json:"latest,omitempty"`
	BaseCommit       string            `json:"base_commit,omitempty"`
	BaseBuildID      int64             `json:"base_build_id,omitempty"`
	BaseBuildNumber  int               `json:"base_build_number,omitempty"`
	Commits          []GitCommitChange `json:"commits"`
	Files            []GitFileChange   `json:"files"`
	CommitsTruncated bool              `json:"commits_truncated"`
	FilesTruncated   bool              `json:"files_truncated"`
	CompareURL       string            `json:"compare_url,omitempty"`
}

// Only checkout output is parsed, never arbitrary builder output. Base64 + NUL
// framing preserves Unicode, quotes, tabs and filenames without shell eval.
func parseGitChanges(log []byte) *GitChangeSet {
	var found *GitChangeSet
	for _, line := range strings.Split(string(log), "\n") {
		if !strings.HasPrefix(line, gitChangesPrefix) || len(line) > 384<<10 {
			continue
		}
		fields := strings.Split(strings.TrimPrefix(line, gitChangesPrefix), "|")
		if len(fields) != 7 || !validSourceCommit(fields[1]) || (fields[2] != "" && !validSourceCommit(fields[2])) {
			continue
		}
		if fields[0] != "complete" && fields[0] != "first_build" && fields[0] != "unavailable" {
			continue
		}
		if (fields[0] == "first_build") != (fields[2] == "") || (fields[3] != "0" && fields[3] != "1") || (fields[4] != "0" && fields[4] != "1") {
			continue
		}
		commits, err := base64.StdEncoding.DecodeString(fields[5])
		if err != nil {
			continue
		}
		files, err := base64.StdEncoding.DecodeString(fields[6])
		if err != nil {
			continue
		}
		if len(commits) > 128<<10 || len(files) > 128<<10 {
			continue
		}
		set := &GitChangeSet{Status: fields[0], Head: fields[1], BaseCommit: fields[2],
			Commits: []GitCommitChange{}, Files: []GitFileChange{}, CommitsTruncated: fields[3] == "1", FilesTruncated: fields[4] == "1"}
		parts := strings.Split(strings.TrimSuffix(string(commits), "\x00"), "\x00")
		valid := len(commits) == 0 || len(parts)%5 == 0
		if len(commits) > 0 && valid {
			for i := 0; i < len(parts); i += 5 {
				if !validSourceCommit(parts[i]) {
					valid = false
					break
				}
				if _, err := time.Parse(time.RFC3339, parts[i+3]); err != nil {
					valid = false
					break
				}
				if len(set.Commits) == 100 {
					set.CommitsTruncated = true
					break
				}
				set.Commits = append(set.Commits, GitCommitChange{Commit: parts[i], Author: parts[i+1], Email: parts[i+2], AuthoredAt: parts[i+3], Subject: parts[i+4]})
			}
		}
		parts = strings.Split(strings.TrimSuffix(string(files), "\x00"), "\x00")
		if len(files) > 0 {
			if len(parts)%2 != 0 {
				valid = false
			} else {
				for i := 0; i < len(parts); i += 2 {
					if !strings.Contains(" A M D T U X B ", " "+parts[i]+" ") || parts[i+1] == "" {
						valid = false
						break
					}
					if len(set.Files) == 200 {
						set.FilesTruncated = true
						break
					}
					set.Files = append(set.Files, GitFileChange{Action: parts[i], Path: parts[i+1]})
				}
			}
		}
		if valid {
			for i := range set.Commits {
				if set.Commits[i].Commit == set.Head {
					item := set.Commits[i]
					set.Latest = &item
					break
				}
			}
			if set.Latest == nil && found != nil && found.Head == set.Head && found.BaseCommit == set.BaseCommit {
				set.Latest = found.Latest
			}
			if found != nil && found.Head == set.Head && found.Status != "unavailable" && set.Status == "unavailable" {
				continue
			}
			found = set
		}
	}
	return found
}

func gitBrowseURL(repository, kind, from, to string) string {
	return commitlink.BrowseURL(repository, kind, from, to)
}

func (r *Repository) fillCommitLinks(ctx context.Context, runs []BuildRun) error {
	records := make([]commitlink.Record, len(runs))
	for i, run := range runs {
		records[i] = commitlink.Record{ProjectID: run.ProjectID, Commit: run.GitCommit, URL: run.GitCommitURL, Snapshot: commitlink.ParseSnapshot(run.ConfigSnapshot)}
	}
	if err := commitlink.Fill(ctx, r.db, records); err != nil {
		return err
	}
	for i := range runs {
		runs[i].GitCommitURL = records[i].URL
	}
	return nil
}

// Freeze the last successful baseline before enqueueing. Concurrent and failed
// builds cannot later move this comparison, and retries retain the original.
func freezeGitBaseline(ctx context.Context, tx *sql.Tx, run *BuildRun) error {
	if run.ConfigSnapshot == nil {
		return nil
	}
	var snapshot BuildExecutionSnapshot
	if err := json.Unmarshal([]byte(*run.ConfigSnapshot), &snapshot); err != nil {
		return err
	}
	if snapshot.RepositoryURL == "" {
		return nil
	}
	if snapshot.ChangeBaseline != nil {
		run.GitCommitURL = gitBrowseURL(snapshot.RepositoryURL, snapshot.CodeSourceType, "", run.GitCommit)
		return nil
	}
	snapshot.ChangeBaseline = &GitChangeBaseline{}
	if snapshot.CodeSourceID > 0 {
		err := tx.QueryRowContext(ctx, `SELECT type FROM code_source WHERE id=? AND project_id IN (0,?)`, snapshot.CodeSourceID, run.ProjectID).Scan(&snapshot.CodeSourceType)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
	}
	base := snapshot.ChangeBaseline
	err := tx.QueryRowContext(ctx, `SELECT id,build_number,git_commit FROM build_run
		WHERE project_id=? AND service_id=? AND git_branch=? AND status='success' AND git_commit<>''
		AND JSON_UNQUOTE(JSON_EXTRACT(config_snapshot,'$.repository_url'))=?
		ORDER BY id DESC LIMIT 1`, run.ProjectID, run.ServiceID, run.GitBranch, snapshot.RepositoryURL).Scan(&base.BuildID, &base.BuildNumber, &base.Commit)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if !validSourceCommit(base.Commit) {
		snapshot.ChangeBaseline = &GitChangeBaseline{}
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	text := string(raw)
	run.ConfigSnapshot = &text
	run.GitCommitURL = gitBrowseURL(snapshot.RepositoryURL, snapshot.CodeSourceType, "", run.GitCommit)
	return nil
}

// Persist before transitioning to terminal / acknowledging resource cleanup.
// Repeat observations are idempotent; a partial observation cannot replace a
// complete one. No network calls are needed to read the record later.
func (r *Repository) SaveGitChanges(ctx context.Context, run BuildRun, changes *GitChangeSet) error {
	if changes == nil {
		return nil
	}
	if !validSourceCommit(changes.Head) || (run.GitCommit != "" && !strings.EqualFold(run.GitCommit, changes.Head)) {
		return fmt.Errorf("Git change head differs from checked-out commit")
	}
	var snapshot BuildExecutionSnapshot
	if run.ConfigSnapshot != nil {
		if err := json.Unmarshal([]byte(*run.ConfigSnapshot), &snapshot); err != nil {
			return err
		}
	}
	set := *changes
	set.Commits = append([]GitCommitChange{}, changes.Commits...)
	if baseline := snapshot.ChangeBaseline; baseline != nil {
		if baseline.Commit != set.BaseCommit {
			return fmt.Errorf("Git change baseline differs from build snapshot")
		}
		set.BaseBuildID, set.BaseBuildNumber = baseline.BuildID, baseline.BuildNumber
	}
	for i := range set.Commits {
		set.Commits[i].URL = gitBrowseURL(snapshot.RepositoryURL, snapshot.CodeSourceType, "", set.Commits[i].Commit)
	}
	if set.BaseCommit != "" {
		set.CompareURL = gitBrowseURL(snapshot.RepositoryURL, snapshot.CodeSourceType, set.BaseCommit, set.Head)
	}
	commitURL := gitBrowseURL(snapshot.RepositoryURL, snapshot.CodeSourceType, "", set.Head)
	var author, email, subject string
	if set.Latest != nil {
		item := *set.Latest
		if item.Commit != set.Head {
			return fmt.Errorf("Git latest metadata differs from checked-out commit")
		}
		item.URL = commitURL
		set.Latest = &item
		author, email, subject = item.Author, item.Email, item.Subject
	}
	data, err := json.Marshal(set)
	if err != nil {
		return err
	}
	files, err := json.Marshal(set.Files)
	if err != nil {
		return err
	}
	// Git permits longer identities/subjects than the legacy summary columns.
	// Keep full metadata in JSON; fit summaries without breaking Unicode or TEXT.
	author = truncateGitSummary(author, 128, 512)
	email = truncateGitSummary(email, 256, 1024)
	subject = truncateGitSummary(subject, 4096, 16384)
	_, err = r.db.ExecContext(ctx, `UPDATE build_run SET git_commit=?,git_commit_short=?,git_commit_url=?,git_changes=?,
		git_author=IF(?='',git_author,?),git_author_email=IF(?='',git_author_email,?),git_message=IF(?='',git_message,?),git_changed_files=?,git_compare_url=?
		WHERE id=? AND (git_commit='' OR git_commit=?)
		AND (git_changes IS NULL OR JSON_UNQUOTE(JSON_EXTRACT(git_changes,'$.status'))='unavailable')`,
		set.Head, set.Head[:8], commitURL, string(data), author, author, email, email, subject, subject, string(files), set.CompareURL, run.ID, set.Head)
	return err
}

func truncateGitSummary(value string, maxRunes, maxBytes int) string {
	value = strings.ToValidUTF8(value, "\uFFFD")
	runes := 0
	for offset := range value {
		if runes == maxRunes || offset >= maxBytes {
			return value[:offset]
		}
		_, size := utf8.DecodeRuneInString(value[offset:])
		if offset+size > maxBytes {
			return value[:offset]
		}
		runes++
	}
	return value
}
