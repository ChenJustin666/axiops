package notification

import "time"

const (
	ChannelFeishu   = "feishu"
	ChannelDingTalk = "dingtalk"
	ChannelWeCom    = "wecom"
	ChannelEmail    = "email"
	ChannelWebhook  = "webhook"
)

type Subscription struct {
	ID           int64     `json:"id"`
	ProjectID    int64     `json:"project_id"`
	UserID       int64     `json:"user_id"`
	Name         string    `json:"name"`
	Channel      string    `json:"channel"`
	CredentialID int64     `json:"credential_id"`
	Recipient    string    `json:"recipient,omitempty"`
	OnSuccess    bool      `json:"on_success"`
	OnFailure    bool      `json:"on_failure"`
	Enabled      bool      `json:"enabled"`
	CreatedBy    int64     `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
type CreateRequest struct {
	UserID       int64  `json:"user_id"`
	Name         string `json:"name"`
	Channel      string `json:"channel"`
	CredentialID int64  `json:"credential_id"`
	Recipient    string `json:"recipient"`
	OnSuccess    bool   `json:"on_success"`
	OnFailure    bool   `json:"on_failure"`
}
type Outbox struct {
	ID, SubscriptionID, ProjectID, EventID                       int64
	EventType, Payload, Status, LockedBy, LastError, DeliveryKey string
	Attempts                                                     int
}

// Delivery is the non-sensitive notification delivery view exposed to users.
// Payload and credential data are intentionally excluded.
type Delivery struct {
	ID               int64     `json:"id"`
	SubscriptionID   int64     `json:"subscription_id"`
	ProjectID        int64     `json:"project_id"`
	SubscriptionName string    `json:"subscription_name"`
	Channel          string    `json:"channel"`
	EventType        string    `json:"event_type"`
	EventID          int64     `json:"event_id"`
	Status           string    `json:"status"`
	Attempts         int       `json:"attempts"`
	NextAttemptAt    time.Time `json:"next_attempt_at"`
	LastError        string    `json:"last_error"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
type Credential struct {
	Type, Data string
	ProjectID  int64
}
type Message struct {
	EventType  string `json:"event_type"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	Status     string `json:"status"`
	ProjectID  int64  `json:"project_id"`
	ResourceID int64  `json:"resource_id"`
	URL        string `json:"url,omitempty"`
}
