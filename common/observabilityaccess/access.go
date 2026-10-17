// Package observabilityaccess adapts legacy observability targets to platform
// authorization. Legacy identifiers are never interpreted as platform IDs.
package observabilityaccess

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/ChenJustin666/axiops/internal/platform/identity"
	"github.com/ChenJustin666/axiops/internal/shared/authz"
)

var ErrDenied = errors.New("无此监控或日志资源的访问权限，请检查账号授权和资源映射")

type Config struct {
	DataSource string `json:",optional"`
	Provider   string `json:",default=legacy_autoops"`
}

type Target struct {
	ClusterUUID string
	Namespace   string
	ProjectUUID string
}

type WorkspaceResolver func(context.Context, uint64) (Target, error)

type TargetStore interface {
	Principal(context.Context, int64) (int64, error)
	Resolve(context.Context, Target) (authz.Request, error)
}

type Guard struct {
	store     TargetStore
	checker   authz.Checker
	workspace WorkspaceResolver
}

func New(store TargetStore, checker authz.Checker, workspace WorkspaceResolver) *Guard {
	return &Guard{store: store, checker: checker, workspace: workspace}
}

// NewSQL uses a separate, read-only-capable pool. Missing configuration fails
// closed for observability routes; unrelated legacy APIs are unaffected.
func NewSQL(c Config, workspace WorkspaceResolver) *Guard {
	if c.DataSource == "" {
		c.DataSource = os.Getenv("AXIOPS_RESOURCE_ACCESS_DSN")
	}
	if strings.TrimSpace(c.DataSource) == "" {
		return New(nil, nil, workspace)
	}
	db, err := sql.Open("mysql", c.DataSource)
	if err != nil {
		return New(nil, nil, workspace)
	}
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)
	provider := strings.TrimSpace(c.Provider)
	if provider == "" {
		provider = "legacy_autoops"
	}
	return New(&SQLStore{db: db, provider: provider}, authz.NewSQLAuthorizer(db), workspace)
}

type SQLStore struct {
	db       *sql.DB
	provider string
}

func (s *SQLStore) Principal(ctx context.Context, externalID int64) (int64, error) {
	p, err := identity.NewRepository(s.db).TargetPrincipal(ctx, s.provider, externalID)
	return p.UserID, err
}

func (s *SQLStore) Resolve(ctx context.Context, target Target) (authz.Request, error) {
	var req authz.Request
	err := s.db.QueryRowContext(ctx, `SELECT c.id FROM legacy_observability_link l
		JOIN cluster c ON c.id=l.target_id
		WHERE l.kind='cluster' AND l.external_id=? AND l.status=1 AND c.status <> 'deleted'`, target.ClusterUUID).Scan(&req.ClusterID)
	if err != nil {
		return req, ErrDenied
	}
	if target.Namespace != "" {
		rows, err := s.db.QueryContext(ctx, `SELECT e.id,e.project_id,e.code
			FROM project_environment e JOIN project p ON p.id=e.project_id
			WHERE e.cluster_id=? AND BINARY e.namespace=BINARY ? AND e.status='active' AND e.environment_type='k8s_namespace' AND p.status <> 'deleted'`, req.ClusterID, target.Namespace)
		if err != nil {
			return req, err
		}
		defer rows.Close()
		count := 0
		for rows.Next() {
			if err := rows.Scan(&req.EnvironmentID, &req.ProjectID, &req.Environment); err != nil {
				return req, err
			}
			count++
		}
		if err := rows.Err(); err != nil {
			return req, err
		}
		if count != 1 {
			return req, ErrDenied
		}
	}
	if target.ProjectUUID != "" {
		var projectID int64
		err := s.db.QueryRowContext(ctx, `SELECT l.target_id FROM legacy_observability_link l
			JOIN project p ON p.id=l.target_id AND p.status <> 'deleted'
			WHERE l.kind='project' AND l.external_id=? AND l.status=1`, target.ProjectUUID).Scan(&projectID)
		if err != nil || req.ProjectID == 0 || projectID != req.ProjectID {
			return req, ErrDenied
		}
	}
	return req, nil
}

func (g *Guard) authorize(ctx context.Context, externalID int64, target Target, resource, action string, aggregate, global bool) (authz.Request, error) {
	var req authz.Request
	if g == nil || g.store == nil || g.checker == nil || externalID <= 0 {
		return req, ErrDenied
	}
	if !global && (!identifier(target.ClusterUUID) || (target.Namespace != "" && !namespace(target.Namespace))) {
		return req, ErrDenied
	}
	userID, err := g.store.Principal(ctx, externalID)
	if err != nil || userID <= 0 {
		return req, ErrDenied
	}
	if !global {
		req, err = g.store.Resolve(ctx, target)
		if err != nil {
			return req, ErrDenied
		}
	}
	req.Resource, req.Action = resource, action
	allowed, err := g.checker.Authorize(ctx, userID, req)
	if err != nil || !allowed {
		return req, ErrDenied
	}
	// A namespace/cluster aggregate cannot safely subtract a denied service, or
	// a cluster aggregate a denied namespace. Deny these requests rather than
	// silently discarding restrictions that the legacy query cannot express.
	policies, err := g.checker.Policies(ctx, userID)
	if err != nil {
		return req, ErrDenied
	}
	for _, p := range policies {
		if !strings.EqualFold(p.Effect, "deny") || !matchesPermission(p, resource, action) {
			continue
		}
		if global {
			return req, ErrDenied
		}
		switch p.ScopeType {
		case authz.ScopeService, authz.ScopeBuild, authz.ScopeWorkflow:
			return req, ErrDenied
		case authz.ScopeProject, authz.ScopeEnvironment:
			if aggregate {
				return req, ErrDenied
			}
		}
		if aggregate && p.Environment != "" && p.Environment != "*" {
			return req, ErrDenied
		}
	}
	return req, nil
}

func matchesPermission(p authz.Policy, resource, action string) bool {
	return (p.Resource == "*" || p.Resource == resource) &&
		(p.Action == "*" || p.Action == action || (p.Action == "read" && action == "view") || (p.Action == "write" && (action == "create" || action == "update" || action == "delete")))
}

type recheckKey struct{}
type targetsKey struct{}
type capabilityKey struct{}

// Can is a UI capability hint. Each actual operation is authorized separately.
func Can(ctx context.Context, action string, global bool) bool {
	check, _ := ctx.Value(capabilityKey{}).(func(string, bool) bool)
	return check != nil && check(action, global)
}

// CheckTarget prevents a second workspace lookup in the handler from changing
// the authorized destination between middleware and the Kubernetes request.
func CheckTarget(ctx context.Context, actual Target) error {
	targets, _ := ctx.Value(targetsKey{}).([]Target)
	for _, target := range targets {
		if target.ClusterUUID == actual.ClusterUUID && target.Namespace == actual.Namespace {
			return Recheck(ctx)
		}
	}
	return ErrDenied
}

// Recheck validates both the legacy session/Casbin decision and current platform
// identity, grants, mapping, and target. Used before delivering stream output.
func Recheck(ctx context.Context) error {
	check, _ := ctx.Value(recheckKey{}).(func() error)
	if check == nil {
		return fmt.Errorf("日志流权限上下文缺失")
	}
	return check()
}

// Check is called only after legacy JWT and API authorization succeed.
func (g *Guard) Check(r *http.Request, revalidateSession func(context.Context) error) (*http.Request, error) {
	checks, protected, err := g.requestChecks(r)
	if !protected {
		return r, nil
	}
	if err != nil {
		return r, ErrDenied
	}
	externalID, ok := r.Context().Value("userId").(uint64)
	if !ok || externalID == 0 || externalID > uint64(^uint64(0)>>1) {
		return r, ErrDenied
	}
	// Freeze both the legacy workspace and its mapped platform environment.
	// Rebinding either must close an already-open stream, never retarget it.
	frozen := make([]authz.Request, len(checks))
	initialized := false
	var frozenPrincipal int64
	var mu sync.Mutex
	check := func() error {
		mu.Lock()
		defer mu.Unlock()
		ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
		defer cancel()
		if revalidateSession == nil || revalidateSession(ctx) != nil {
			return ErrDenied
		}
		if g == nil || g.store == nil {
			return ErrDenied
		}
		principal, err := g.store.Principal(ctx, int64(externalID))
		if err != nil || principal <= 0 || (initialized && principal != frozenPrincipal) {
			return ErrDenied
		}
		frozenPrincipal = principal
		for i := range checks {
			c := &checks[i]
			target := c.target
			if c.workspaceID > 0 {
				if g.workspace == nil {
					return ErrDenied
				}
				actual, err := g.workspace(ctx, c.workspaceID)
				if err != nil || (target.ClusterUUID != "" && target.ClusterUUID != actual.ClusterUUID) || (target.Namespace != "" && target.Namespace != actual.Namespace) || (target.ProjectUUID != "" && target.ProjectUUID != actual.ProjectUUID) {
					return ErrDenied
				}
				if initialized && actual != target {
					return ErrDenied
				}
				target = actual
				c.target = actual
			}
			resolved, err := g.authorize(ctx, int64(externalID), target, c.resource, c.action, c.aggregate, c.global)
			if err != nil {
				return err
			}
			if initialized && resolved != frozen[i] {
				return ErrDenied
			}
			frozen[i] = resolved
		}
		initialized = true
		return nil
	}
	if err := check(); err != nil {
		return r, err
	}
	targets := make([]Target, len(checks))
	for i, c := range checks {
		targets[i] = c.target
	}
	ctx := context.WithValue(r.Context(), recheckKey{}, check)
	ctx = context.WithValue(ctx, targetsKey{}, targets)
	ctx = context.WithValue(ctx, capabilityKey{}, func(action string, global bool) bool {
		if action != "view" && action != "export" && action != "manage" {
			return false
		}
		if check() != nil || len(checks) != 1 {
			return false
		}
		c := checks[0]
		ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
		defer cancel()
		_, err := g.authorize(ctx, int64(externalID), c.target, c.resource, action, c.aggregate, global)
		return err == nil
	})
	return r.WithContext(ctx), nil
}
