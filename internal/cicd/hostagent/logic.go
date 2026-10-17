package hostagent

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
)

var (
	ErrInvalid      = errors.New("invalid host agent request")
	ErrNoPermission = errors.New("host agent permission denied")
	ErrUnauthorized = errors.New("host agent authentication failed")
	identifier      = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
)

type Logic struct {
	repo       *Repository
	authorizer authz.Checker
	lease      time.Duration
}

func NewLogic(db *sql.DB, authorizer authz.Checker, lease time.Duration) *Logic {
	if lease < 30*time.Second || lease > 10*time.Minute {
		lease = 2 * time.Minute
	}
	return &Logic{repo: NewRepository(db), authorizer: authorizer, lease: lease}
}

func (l *Logic) CreateRegistrationToken(ctx context.Context, userID, projectID int64, req RegistrationTokenRequest) (*RegistrationTokenResponse, error) {
	req.HostGroupRef = strings.TrimSpace(req.HostGroupRef)
	if projectID <= 0 || !identifier.MatchString(req.HostGroupRef) || req.ExpiresMinutes < 1 || req.ExpiresMinutes > 1440 || req.MaxUses < 1 || req.MaxUses > 100 {
		return nil, ErrInvalid
	}
	allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "host_agent", Action: "create", ProjectID: projectID})
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrNoPermission
	}
	expiresAt := time.Now().Add(time.Duration(req.ExpiresMinutes) * time.Minute)
	token, err := l.repo.CreateRegistrationToken(ctx, projectID, req.HostGroupRef, expiresAt, req.MaxUses, userID)
	if err != nil {
		return nil, err
	}
	return &RegistrationTokenResponse{Token: token, ExpiresAt: expiresAt}, nil
}

func (l *Logic) List(ctx context.Context, userID, projectID int64) ([]Agent, error) {
	scope, err := l.authorizer.Scopes(ctx, userID, "host_agent", "view")
	if err != nil {
		return nil, err
	}
	if !scope.HasAnyAllow() {
		return nil, ErrNoPermission
	}
	return l.repo.List(ctx, projectID, scope)
}

func (l *Logic) Register(ctx context.Context, req RegisterRequest) (*RegisterResponse, error) {
	req.Token = strings.TrimSpace(req.Token)
	req.AssetRef = strings.TrimSpace(req.AssetRef)
	req.Name = strings.TrimSpace(req.Name)
	if len(req.Token) != 64 || !identifier.MatchString(req.AssetRef) || !identifier.MatchString(req.Name) || len(req.Version) > 32 || len(req.OS) > 32 || len(req.Arch) > 32 {
		return nil, ErrInvalid
	}
	result, err := l.repo.Register(ctx, req)
	if err == sql.ErrNoRows {
		return nil, ErrUnauthorized
	}
	return result, err
}

func (l *Logic) Authenticate(ctx context.Context, value string) (*Agent, error) {
	token, ok := strings.CutPrefix(strings.TrimSpace(value), "Bearer ")
	if !ok || len(token) != 64 {
		return nil, ErrUnauthorized
	}
	agent, err := l.repo.Authenticate(ctx, token)
	if err != nil {
		return nil, err
	}
	if agent == nil {
		return nil, ErrUnauthorized
	}
	return agent, nil
}

func (l *Logic) Heartbeat(ctx context.Context, agent *Agent, req HeartbeatRequest) error {
	if agent == nil || len(req.Version) > 32 {
		return ErrInvalid
	}
	return l.repo.Heartbeat(ctx, agent.ID, strings.TrimSpace(req.Version))
}

func (l *Logic) Poll(ctx context.Context, agent *Agent) (*Command, error) {
	if agent == nil {
		return nil, ErrUnauthorized
	}
	if err := l.repo.Heartbeat(ctx, agent.ID, agent.Version); err != nil {
		return nil, err
	}
	return l.repo.Poll(ctx, agent.ID, l.lease)
}

func (l *Logic) Report(ctx context.Context, agent *Agent, commandID int64, req ReportRequest) error {
	if agent == nil || commandID <= 0 || len(req.LogText) > 2<<20 || len(req.ErrorMessage) > 4096 {
		return ErrInvalid
	}
	return l.repo.Report(ctx, agent.ID, commandID, req)
}
