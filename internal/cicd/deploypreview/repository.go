package deploypreview

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"strings"
)

type previewRepository interface {
	ResolveTarget(context.Context, int64, int64, int64) (*Target, error)
	LoadLatestHelmInventory(context.Context, int64, int64, string) ([]byte, error)
}

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) ResolveTarget(ctx context.Context, environmentID, serviceID, requestedRevisionID int64) (*Target, error) {
	if r == nil || r.db == nil || environmentID <= 0 || serviceID <= 0 || requestedRevisionID < 0 {
		return nil, fmt.Errorf("environment, service, and database are required")
	}
	var target Target
	var currentRevisionID, overrideRevisionID int64
	err := r.db.QueryRowContext(ctx, `SELECT env.project_id,env.id,env.code,env.environment_type,env.binding_status,env.status,
		env.cluster_id,env.namespace,svc.id,svc.code,svc.delivery_type,svc.current_revision_id,
		COALESCE(ov.service_revision_id,0)
		FROM project_environment env
		JOIN app_service svc ON svc.project_id=env.project_id AND svc.id=? AND svc.status='active'
		LEFT JOIN environment_service_override ov ON ov.project_environment_id=env.id AND ov.service_id=svc.id
		WHERE env.id=?`, serviceID, environmentID).Scan(
		&target.ProjectID, &target.EnvironmentID, &target.EnvironmentCode, &target.EnvironmentType,
		&target.BindingStatus, &target.EnvironmentStatus, &target.ClusterID, &target.Namespace,
		&target.ServiceID, &target.ServiceCode, &target.DeliveryType, &currentRevisionID, &overrideRevisionID,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	target.ServiceRevisionID = requestedRevisionID
	if target.ServiceRevisionID == 0 {
		target.ServiceRevisionID = overrideRevisionID
	}
	if target.ServiceRevisionID == 0 {
		target.ServiceRevisionID = currentRevisionID
	}
	if target.ServiceRevisionID <= 0 {
		return nil, nil
	}
	var exists int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM service_revision WHERE id=? AND service_id=?`, target.ServiceRevisionID, serviceID).Scan(&exists); err != nil {
		return nil, err
	}
	if exists != 1 {
		return nil, nil
	}
	return &target, nil
}

func (r *Repository) LoadLatestHelmInventory(ctx context.Context, environmentID, serviceID int64, releaseName string) ([]byte, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("preview database is not configured")
	}
	var inventory []byte
	var expectedDigest string
	err := r.db.QueryRowContext(ctx, `SELECT rendered_manifest,manifest_digest FROM deployment_snapshot
		WHERE environment_id=? AND service_id=? AND provider='helm_deploy' AND status='applied' AND helm_release_name=?
		ORDER BY applied_at DESC,id DESC LIMIT 1`, environmentID, serviceID, releaseName).Scan(&inventory, &expectedDigest)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(inventory)
	if !strings.EqualFold(expectedDigest, fmt.Sprintf("%x", digest[:])) {
		return nil, fmt.Errorf("latest Helm deployment snapshot digest mismatch")
	}
	return inventory, nil
}
