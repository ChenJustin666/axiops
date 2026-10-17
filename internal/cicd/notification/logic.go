package notification

import (
	"context"
	"database/sql"
	"errors"
	"net/mail"
	"strings"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/secretbox"
)

var (
	ErrInvalid      = errors.New("invalid notification subscription")
	ErrNoPermission = errors.New("notification permission denied")
)

type Logic struct {
	repo       logicRepository
	authorizer authz.Checker
	box        *secretbox.Box
}

type logicRepository interface {
	Create(context.Context, Subscription) (int64, error)
	List(context.Context, int64, authz.ScopeSet) ([]Subscription, error)
	Delete(context.Context, int64, int64) error
	FindCredential(context.Context, int64) (*Credential, error)
	ListDeliveries(context.Context, int64, string, int, int, authz.ScopeSet) ([]Delivery, int64, error)
}

func NewLogic(db *sql.DB, authorizer authz.Checker, key string) *Logic {
	box, _ := secretbox.New(key)
	return &Logic{repo: NewRepository(db), authorizer: authorizer, box: box}
}
func (l *Logic) Create(ctx context.Context, userID, projectID int64, req CreateRequest) (int64, error) {
	if !validChannel(req.Channel) || strings.TrimSpace(req.Name) == "" || len(strings.TrimSpace(req.Name)) > 64 || req.CredentialID <= 0 || (!req.OnSuccess && !req.OnFailure) || req.UserID < 0 {
		return 0, ErrInvalid
	}
	if req.Channel == ChannelEmail {
		if _, err := mail.ParseAddress(strings.TrimSpace(req.Recipient)); err != nil {
			return 0, ErrInvalid
		}
	}
	if err := l.authorize(ctx, userID, "create", projectID); err != nil {
		return 0, err
	}
	credential, err := l.repo.FindCredential(ctx, req.CredentialID)
	if err != nil || credential == nil || !validCredentialType(req.Channel, credential.Type) || (credential.ProjectID != 0 && credential.ProjectID != projectID) {
		return 0, ErrInvalid
	}
	allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "credential", Action: "view", ProjectID: projectID})
	if err != nil {
		return 0, err
	}
	if !allowed {
		return 0, ErrNoPermission
	}
	return l.repo.Create(ctx, Subscription{ProjectID: projectID, UserID: req.UserID, Name: strings.TrimSpace(req.Name), Channel: req.Channel, CredentialID: req.CredentialID, Recipient: strings.TrimSpace(req.Recipient), OnSuccess: req.OnSuccess, OnFailure: req.OnFailure, CreatedBy: userID})
}
func (l *Logic) List(ctx context.Context, userID, projectID int64) ([]Subscription, error) {
	scope, err := l.authorizer.Scopes(ctx, userID, "notification", "view")
	if err != nil {
		return nil, err
	}
	if !scope.HasAnyAllow() {
		return nil, ErrNoPermission
	}
	return l.repo.List(ctx, projectID, scope)
}
func (l *Logic) Delete(ctx context.Context, userID, id, projectID int64) error {
	if id <= 0 || projectID <= 0 {
		return ErrInvalid
	}
	if err := l.authorize(ctx, userID, "delete", projectID); err != nil {
		return err
	}
	return l.repo.Delete(ctx, id, projectID)
}
func (l *Logic) ListDeliveries(ctx context.Context, userID, projectID int64, status string, page, pageSize int) ([]Delivery, int64, error) {
	status = strings.TrimSpace(status)
	if projectID < 0 || !validDeliveryStatus(status) {
		return nil, 0, ErrInvalid
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	scope, err := l.authorizer.Scopes(ctx, userID, "notification", "view")
	if err != nil {
		return nil, 0, err
	}
	if !scope.HasAnyAllow() {
		return nil, 0, ErrNoPermission
	}
	return l.repo.ListDeliveries(ctx, projectID, status, page, pageSize, scope)
}
func (l *Logic) authorize(ctx context.Context, userID int64, action string, projectID int64) error {
	allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "notification", Action: action, ProjectID: projectID})
	if err != nil {
		return err
	}
	if !allowed {
		return ErrNoPermission
	}
	return nil
}
func validChannel(v string) bool {
	switch v {
	case ChannelFeishu, ChannelDingTalk, ChannelWeCom, ChannelEmail, ChannelWebhook:
		return true
	}
	return false
}
func validCredentialType(channel, typ string) bool {
	if channel == ChannelEmail {
		return typ == "smtp"
	}
	return typ == "notification_webhook"
}

func validDeliveryStatus(status string) bool {
	switch status {
	case "", "pending", "processing", "succeeded", "dead":
		return true
	default:
		return false
	}
}
