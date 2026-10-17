package workflow

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type DeploymentSnapshot struct {
	ID                int64           `json:"id"`
	WorkflowRunID     int64           `json:"workflow_run_id"`
	WorkflowTaskRunID int64           `json:"workflow_task_run_id"`
	NodeKey           string          `json:"node_key"`
	ProjectID         int64           `json:"project_id"`
	EnvironmentID     int64           `json:"environment_id"`
	ServiceID         int64           `json:"service_id"`
	ServiceRevisionID int64           `json:"service_revision_id"`
	ClusterID         int64           `json:"cluster_id"`
	Namespace         string          `json:"namespace"`
	Provider          string          `json:"provider"`
	ManifestDigest    string          `json:"manifest_digest"`
	RenderedManifest  []byte          `json:"rendered_manifest,omitempty"`
	ValuesSnapshot    json.RawMessage `json:"values_snapshot,omitempty"`
	HelmReleaseName   string          `json:"helm_release_name,omitempty"`
	HelmRevision      int             `json:"helm_revision,omitempty"`
	Status            string          `json:"status"`
	CreatedAt         time.Time       `json:"created_at"`
}

type DeploymentSnapshotStore interface {
	Prepare(context.Context, *DeploymentSnapshot) error
	MarkApplied(context.Context, int64, string, string) error
}

type HelmDeploymentSnapshotStore interface {
	Prepare(context.Context, *DeploymentSnapshot) error
	MarkHelmApplied(context.Context, int64, string, string, int) error
}

type DeploymentSnapshotReader interface {
	FindApplied(context.Context, int64) (*DeploymentSnapshot, error)
}

// NewYAMLDeploymentSnapshot stores the exact validated manifest used by apply.
// Secret resources are rejected before this function, so rollback does not
// turn an audit snapshot into a second credential store.
func NewYAMLDeploymentSnapshot(run WorkflowRun, task TaskRun, config YAMLDeployConfig, target YAMLTarget) (*DeploymentSnapshot, error) {
	if run.ID <= 0 || task.ID <= 0 || strings.TrimSpace(task.NodeKey) == "" || run.ProjectID <= 0 || run.EnvironmentID <= 0 || config.ServiceID <= 0 || config.ServiceRevisionID <= 0 {
		return nil, fmt.Errorf("snapshot identity is incomplete")
	}
	if target.ClusterID <= 0 || strings.TrimSpace(target.Namespace) == "" {
		return nil, fmt.Errorf("snapshot target is incomplete")
	}
	if _, err := ValidateYAMLManifest(target.Manifest, target.Namespace); err != nil {
		return nil, err
	}
	hash := sha256.Sum256(target.Manifest)
	return &DeploymentSnapshot{
		WorkflowRunID: run.ID, WorkflowTaskRunID: task.ID, NodeKey: task.NodeKey,
		ProjectID: run.ProjectID, EnvironmentID: run.EnvironmentID,
		ServiceID: config.ServiceID, ServiceRevisionID: config.ServiceRevisionID,
		ClusterID: target.ClusterID, Namespace: strings.TrimSpace(target.Namespace),
		Provider: NodeYAMLDeploy, ManifestDigest: hex.EncodeToString(hash[:]), RenderedManifest: append([]byte(nil), target.Manifest...), Status: "prepared",
	}, nil
}

// NewHelmDeploymentSnapshot stores only resource identities and redacted value
// references. Helm revision history, not the rendered objects, is the rollback source.
func NewHelmDeploymentSnapshot(run WorkflowRun, task TaskRun, config HelmDeployConfig, target HelmTarget, manifest []byte) (*DeploymentSnapshot, error) {
	if run.ID <= 0 || task.ID <= 0 || strings.TrimSpace(task.NodeKey) == "" || run.ProjectID <= 0 || run.EnvironmentID <= 0 || config.ServiceID <= 0 || config.ServiceRevisionID <= 0 {
		return nil, fmt.Errorf("snapshot identity is incomplete")
	}
	if target.ClusterID <= 0 || strings.TrimSpace(target.Namespace) == "" || strings.TrimSpace(target.ReleaseName) == "" {
		return nil, fmt.Errorf("Helm snapshot target is incomplete")
	}
	inventory, err := helmManifestInventory(manifest, target.Namespace)
	if err != nil {
		return nil, err
	}
	snapshotValues := MergeHelmValues(target.SnapshotValues)
	if target.ImageRef != "" {
		snapshotValues["_axiops_image"] = target.ImageRef
	}
	values, err := json.Marshal(snapshotValues)
	if err != nil {
		return nil, fmt.Errorf("encode redacted Helm values: %w", err)
	}
	hash := sha256.Sum256(inventory)
	return &DeploymentSnapshot{
		WorkflowRunID: run.ID, WorkflowTaskRunID: task.ID, NodeKey: task.NodeKey,
		ProjectID: run.ProjectID, EnvironmentID: run.EnvironmentID, ServiceID: config.ServiceID, ServiceRevisionID: config.ServiceRevisionID,
		ClusterID: target.ClusterID, Namespace: target.Namespace, Provider: NodeHelmDeploy,
		ManifestDigest: hex.EncodeToString(hash[:]), RenderedManifest: inventory, ValuesSnapshot: values,
		HelmReleaseName: target.ReleaseName, Status: "prepared",
	}, nil
}

func (s *DeploymentSnapshot) VerifyManifestDigest() bool {
	if s == nil || s.ManifestDigest == "" || len(s.RenderedManifest) == 0 {
		return false
	}
	hash := sha256.Sum256(s.RenderedManifest)
	return strings.EqualFold(s.ManifestDigest, hex.EncodeToString(hash[:]))
}

type SQLDeploymentSnapshotStore struct{ db *sql.DB }

func NewSQLDeploymentSnapshotStore(db *sql.DB) *SQLDeploymentSnapshotStore {
	return &SQLDeploymentSnapshotStore{db: db}
}

func (s *SQLDeploymentSnapshotStore) Prepare(ctx context.Context, snapshot *DeploymentSnapshot) error {
	if s == nil || s.db == nil || snapshot == nil || !snapshot.VerifyManifestDigest() {
		return fmt.Errorf("valid deployment snapshot and database are required")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO deployment_snapshot
		(workflow_run_id, workflow_task_run_id, node_key, project_id, environment_id, service_id, service_revision_id,
		 cluster_id, namespace, provider,
		 manifest_digest, rendered_manifest, values_snapshot, helm_release_name, helm_revision, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'prepared')
		ON DUPLICATE KEY UPDATE
		manifest_digest=IF(status='applied',manifest_digest,VALUES(manifest_digest)),
		rendered_manifest=IF(status='applied',rendered_manifest,VALUES(rendered_manifest)),
		values_snapshot=IF(status='applied',values_snapshot,VALUES(values_snapshot)),
		helm_release_name=IF(status='applied',helm_release_name,VALUES(helm_release_name)),
		status=IF(status='applied',status,'prepared')`,
		snapshot.WorkflowRunID, snapshot.WorkflowTaskRunID, snapshot.NodeKey, snapshot.ProjectID, snapshot.EnvironmentID,
		snapshot.ServiceID, snapshot.ServiceRevisionID, snapshot.ClusterID, snapshot.Namespace, snapshot.Provider,
		snapshot.ManifestDigest, snapshot.RenderedManifest, snapshot.ValuesSnapshot, snapshot.HelmReleaseName, snapshot.HelmRevision)
	return err
}

func (s *SQLDeploymentSnapshotStore) MarkHelmApplied(ctx context.Context, runID int64, nodeKey, releaseName string, revision int) error {
	if s == nil || s.db == nil || runID <= 0 || strings.TrimSpace(nodeKey) == "" || strings.TrimSpace(releaseName) == "" || revision <= 0 {
		return fmt.Errorf("Helm snapshot identity, release, revision, and database are required")
	}
	result, err := s.db.ExecContext(ctx, `UPDATE deployment_snapshot SET status='applied',helm_release_name=?,helm_revision=?,applied_at=CURRENT_TIMESTAMP
		WHERE workflow_run_id=? AND node_key=? AND provider=? AND status='prepared'`, releaseName, revision, runID, nodeKey, NodeHelmDeploy)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		var count int
		if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM deployment_snapshot WHERE workflow_run_id=? AND node_key=? AND provider=? AND status='applied' AND helm_release_name=? AND helm_revision=?`, runID, nodeKey, NodeHelmDeploy, releaseName, revision).Scan(&count); err != nil || count != 1 {
			return fmt.Errorf("prepared Helm deployment snapshot was not found")
		}
	}
	return nil
}

func (s *SQLDeploymentSnapshotStore) MarkApplied(ctx context.Context, runID int64, nodeKey, provider string) error {
	if s == nil || s.db == nil || runID <= 0 || strings.TrimSpace(nodeKey) == "" || strings.TrimSpace(provider) == "" {
		return fmt.Errorf("snapshot identity and database are required")
	}
	result, err := s.db.ExecContext(ctx, `UPDATE deployment_snapshot SET status='applied', applied_at=CURRENT_TIMESTAMP
		WHERE workflow_run_id=? AND node_key=? AND provider=? AND status='prepared'`, runID, nodeKey, provider)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		var count int
		if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM deployment_snapshot WHERE workflow_run_id=? AND node_key=? AND provider=? AND status='applied'`, runID, nodeKey, provider).Scan(&count); err != nil || count != 1 {
			return fmt.Errorf("prepared deployment snapshot was not found")
		}
	}
	return nil
}

func (s *SQLDeploymentSnapshotStore) FindApplied(ctx context.Context, id int64) (*DeploymentSnapshot, error) {
	if s == nil || s.db == nil || id <= 0 {
		return nil, fmt.Errorf("snapshot id and database are required")
	}
	var snapshot DeploymentSnapshot
	var values []byte
	err := s.db.QueryRowContext(ctx, `SELECT id, workflow_run_id, workflow_task_run_id, node_key,
		project_id, environment_id, service_id, service_revision_id, cluster_id, namespace, provider,
		manifest_digest, rendered_manifest, values_snapshot,helm_release_name,helm_revision,status, created_at
		FROM deployment_snapshot WHERE id=? AND status='applied'`, id).Scan(
		&snapshot.ID, &snapshot.WorkflowRunID, &snapshot.WorkflowTaskRunID, &snapshot.NodeKey,
		&snapshot.ProjectID, &snapshot.EnvironmentID, &snapshot.ServiceID, &snapshot.ServiceRevisionID,
		&snapshot.ClusterID, &snapshot.Namespace, &snapshot.Provider, &snapshot.ManifestDigest,
		&snapshot.RenderedManifest, &values, &snapshot.HelmReleaseName, &snapshot.HelmRevision, &snapshot.Status, &snapshot.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	snapshot.ValuesSnapshot = values
	if !snapshot.VerifyManifestDigest() {
		return nil, fmt.Errorf("deployment snapshot manifest digest mismatch")
	}
	return &snapshot, nil
}
