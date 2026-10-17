package codesource

import "time"

type RepositoryFile struct {
	Path    string `json:"path"`
	Content []byte `json:"-"`
}

type RepositorySnapshot struct {
	Source   CodeSource       `json:"source"`
	RepoPath string           `json:"repo_path"`
	Ref      string           `json:"ref"`
	CommitID string           `json:"commit_id"`
	Files    []RepositoryFile `json:"files"`
}

type CodeSource struct {
	ID                  int64     `json:"id"`
	ProjectID           int64     `json:"project_id"`
	Name                string    `json:"name"`
	Type                string    `json:"type"`
	URL                 string    `json:"url"`
	CredentialID        int64     `json:"credential_id"`
	WebhookCredentialID int64     `json:"webhook_credential_id,omitempty"`
	Status              string    `json:"status"`
	CreatedBy           int64     `json:"created_by"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type CreateRequest struct {
	Name                string `json:"name"`
	Type                string `json:"type"`
	URL                 string `json:"url"`
	CredentialID        int64  `json:"credential_id"`
	WebhookCredentialID int64  `json:"webhook_credential_id,omitempty"`
}

type UpdateRequest struct {
	Name                string `json:"name"`
	Type                string `json:"type"`
	URL                 string `json:"url"`
	CredentialID        int64  `json:"credential_id"`
	WebhookCredentialID int64  `json:"webhook_credential_id,omitempty"`
	Status              string `json:"status"`
}

type CredentialRef struct {
	ID        int64
	Type      string
	Data      string
	ProjectID int64
}
