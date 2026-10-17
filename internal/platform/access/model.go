package access

import "time"

type RoleBinding struct {
	ID          int64      `json:"id"`
	SubjectType string     `json:"subject_type"`
	SubjectID   int64      `json:"subject_id"`
	RoleID      int64      `json:"role_id"`
	ScopeType   string     `json:"scope_type"`
	ScopeID     int64      `json:"scope_id"`
	Environment string     `json:"environment"`
	Status      int        `json:"status"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedBy   int64      `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type CreateBindingRequest struct {
	SubjectType string     `json:"subject_type"`
	SubjectID   int64      `json:"subject_id"`
	RoleID      int64      `json:"role_id"`
	ScopeType   string     `json:"scope_type"`
	ScopeID     int64      `json:"scope_id"`
	Environment string     `json:"environment"`
	ExpiresAt   *time.Time `json:"expires_at"`
}

type UpdateBindingRequest struct {
	Status         *int       `json:"status"`
	ExpiresAt      *time.Time `json:"expires_at"`
	ClearExpiresAt bool       `json:"clear_expires_at"`
}

type Menu struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Code      string `json:"code"`
	ParentID  int64  `json:"parent_id"`
	Path      string `json:"path"`
	Icon      string `json:"icon"`
	SortOrder int    `json:"sort_order"`
	Type      int    `json:"type"`
}
