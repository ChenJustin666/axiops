package department

import "time"

type Department struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	ParentID     int64     `json:"parent_id"`
	SortOrder    int       `json:"sort_order"`
	LeaderUserID int64     `json:"leader_user_id"`
	UserCount    int64     `json:"user_count"`
	CreatedBy    int64     `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CreateRequest struct {
	Name         string `json:"name"`
	ParentID     int64  `json:"parent_id"`
	SortOrder    int    `json:"sort_order"`
	LeaderUserID int64  `json:"leader_user_id"`
}

type UpdateRequest struct {
	Name         string `json:"name"`
	ParentID     int64  `json:"parent_id"`
	SortOrder    int    `json:"sort_order"`
	LeaderUserID int64  `json:"leader_user_id"`
}
