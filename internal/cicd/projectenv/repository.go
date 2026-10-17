package projectenv

import (
	"context"
	"database/sql"
	"github.com/ChenJustin666/axiops/internal/cicd/servicestate"
	"github.com/ChenJustin666/axiops/internal/cicd/workflow"
	"strings"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
)

type Repository struct {
	db        *sql.DB
	decryptor workflow.CredentialDecryptor
}

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) FindProjectDeliveryProfile(ctx context.Context, projectID int64) (*ProjectDeliveryProfile, error) {
	var profile ProjectDeliveryProfile
	err := r.db.QueryRowContext(ctx, `SELECT delivery_type, delivery_status FROM project WHERE id=?`, projectID).
		Scan(&profile.DeliveryType, &profile.DeliveryStatus)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &profile, err
}

func (r *Repository) FindCluster(ctx context.Context, clusterID int64) (*ClusterTarget, error) {
	var cluster ClusterTarget
	err := r.db.QueryRowContext(ctx, `SELECT id, status FROM cluster WHERE id=?`, clusterID).Scan(&cluster.ID, &cluster.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &cluster, err
}

func (r *Repository) FindByID(ctx context.Context, id int64) (*ProjectEnvironment, error) {
	var environment ProjectEnvironment
	err := r.db.QueryRowContext(ctx, `SELECT id, project_id, name, code, environment_type, cluster_id, namespace,
		host_group_ref, binding_status, is_production, deploy_approval, status, created_by, created_at, updated_at
		FROM project_environment WHERE id=?`, id).Scan(
		&environment.ID, &environment.ProjectID, &environment.Name, &environment.Code, &environment.EnvironmentType,
		&environment.ClusterID, &environment.Namespace, &environment.HostGroupRef, &environment.BindingStatus,
		&environment.IsProduction, &environment.DeployApproval, &environment.Status, &environment.CreatedBy,
		&environment.CreatedAt, &environment.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &environment, err
}

func (r *Repository) List(ctx context.Context, scope authz.ScopeSet, projectID int64) ([]ProjectEnvironment, error) {
	predicate, args := scope.EnvironmentSQLPredicate("project_id", "id", "code", "cluster_id")
	conditions := []string{predicate, "status='active'"}
	if projectID > 0 {
		conditions = append(conditions, "project_id=?")
		args = append(args, projectID)
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, project_id, name, code, environment_type, cluster_id, namespace,
		host_group_ref, binding_status, is_production, deploy_approval, status, created_by, created_at, updated_at
		FROM project_environment WHERE `+strings.Join(conditions, " AND ")+` ORDER BY is_production ASC, id ASC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var environments []ProjectEnvironment
	for rows.Next() {
		var environment ProjectEnvironment
		if err := rows.Scan(&environment.ID, &environment.ProjectID, &environment.Name, &environment.Code,
			&environment.EnvironmentType, &environment.ClusterID, &environment.Namespace, &environment.HostGroupRef,
			&environment.BindingStatus, &environment.IsProduction, &environment.DeployApproval, &environment.Status,
			&environment.CreatedBy, &environment.CreatedAt, &environment.UpdatedAt); err != nil {
			return nil, err
		}
		environments = append(environments, environment)
	}
	return environments, rows.Err()
}

func (r *Repository) Create(ctx context.Context, environment *ProjectEnvironment) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `INSERT INTO project_environment
		(project_id, name, code, environment_type, cluster_id, namespace, host_group_ref, binding_status,
		 is_production, deploy_approval, status, created_by) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		environment.ProjectID, environment.Name, environment.Code, environment.EnvironmentType, environment.ClusterID,
		environment.Namespace, environment.HostGroupRef, environment.BindingStatus, environment.IsProduction,
		environment.DeployApproval, environment.Status, environment.CreatedBy)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO variable_set
		(project_id, environment_id, service_id, name, scope, revision, created_by)
		VALUES (?, ?, 0, 'default', 'environment', 1, ?)`, environment.ProjectID, id, environment.CreatedBy); err != nil {
		return 0, err
	}
	for _, s := range environment.initialHelmServices {
		c, e := readServiceConfig(ctx, tx, id, s.ServiceID, true)
		if e != nil {
			return 0, e
		}
		request := SaveConfigRequest{Targets: []ConfigTarget{{EnvironmentID: id, Token: c.Token}}, RevisionID: s.RevisionID, Helm: &s.Helm}
		if e = r.saveServiceConfigTx(ctx, tx, environment.CreatedBy, environment.ProjectID, s.ServiceID, &request); e != nil {
			return 0, e
		}
		environment.initialConfigs = append(environment.initialConfigs, request.Result...)
	}
	if environment.preview {
		return id, nil
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) Clone(ctx context.Context, source, target *ProjectEnvironment, userID int64) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `INSERT INTO project_environment
		(project_id, name, code, environment_type, cluster_id, namespace, host_group_ref, binding_status,
		 is_production, deploy_approval, status, created_by) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'active', ?)`,
		target.ProjectID, target.Name, target.Code, target.EnvironmentType, target.ClusterID,
		target.Namespace, target.HostGroupRef, target.BindingStatus, target.IsProduction, target.DeployApproval, userID)
	if err != nil {
		return 0, err
	}
	targetID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	// Clone every environment-scoped variable set (the project-level set is
	// intentionally excluded). This preserves service-specific values such as
	// port, replicas, CPU/memory and probe settings. Secret values are omitted.
	if _, err = tx.ExecContext(ctx, `INSERT INTO variable_set
		(project_id, environment_id, service_id, name, scope, revision, created_by)
		SELECT project_id, ?, service_id, name, scope, 1, ?
		FROM variable_set
		WHERE environment_id=? AND scope='environment'`, targetID, userID, source.ID); err != nil {
		return 0, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO variable_item
		(variable_set_id, variable_key, value_text, credential_id, is_secret, created_by)
		SELECT target.id, source_item.variable_key, source_item.value_text, 0, 0, ?
		FROM variable_set source_set
		JOIN variable_set target
		  ON target.project_id=source_set.project_id
		 AND target.environment_id=?
		 AND target.service_id=source_set.service_id
		 AND target.name=source_set.name
		 AND target.scope=source_set.scope
		JOIN variable_item source_item ON source_item.variable_set_id=source_set.id
		WHERE source_set.environment_id=? AND source_set.scope='environment' AND source_item.is_secret=0`, userID, targetID, source.ID); err != nil {
		return 0, err
	}
	// Helm values and image mappings are environment-owned. A cloned binding
	// without its settings would silently deploy Chart defaults. Require an
	// explicit preview/save in the new environment instead.
	if _, err = tx.ExecContext(ctx, `INSERT INTO environment_service_override
		(project_environment_id, service_id, service_revision_id, image_ref, replicas, revision, updated_by)
		SELECT ?, o.service_id, o.service_revision_id, o.image_ref, o.replicas, 1, ?
		FROM environment_service_override o
		JOIN app_service s ON s.id=o.service_id
		WHERE o.project_environment_id=? AND s.delivery_type<>'helm'`, targetID, userID, source.ID); err != nil {
		return 0, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO environment_service_yaml_revision(environment_id,service_id,base_revision_id,definition,created_by) SELECT ?,y.service_id,y.base_revision_id,y.definition,? FROM environment_service_override o JOIN environment_service_yaml_revision y ON y.id=o.environment_yaml_revision_id AND y.environment_id=o.project_environment_id AND y.service_id=o.service_id WHERE o.project_environment_id=?`, targetID, userID, source.ID); err != nil {
		return 0, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE environment_service_override o JOIN environment_service_yaml_revision y ON y.environment_id=o.project_environment_id AND y.service_id=o.service_id SET o.environment_yaml_revision_id=y.id WHERE o.project_environment_id=?`, targetID); err != nil {
		return 0, err
	}
	// Carry environment-owned non-secret resources (Ingress, ConfigMap and
	// PVC) to the new environment. Secret manifests are intentionally excluded;
	// credentials must be bound explicitly for the target environment.
	if _, err = tx.ExecContext(ctx, `INSERT INTO environment_resource
		(environment_id, name, kind, manifest, revision, created_by)
		SELECT ?, name, kind, manifest, 1, ? FROM environment_resource
		WHERE environment_id=? AND kind NOT IN ('AxiOpsServiceSettings', 'AxiOpsHelmValues','AxiOpsHelmSharedValues', 'Secret')`, targetID, userID, source.ID); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return targetID, nil
}

func (r *Repository) FindVariableSetByEnvironment(ctx context.Context, environmentID int64) (*VariableSet, error) {
	var set VariableSet
	err := r.db.QueryRowContext(ctx, `SELECT id, project_id, environment_id, service_id, name, scope, revision,
		created_by, created_at, updated_at FROM variable_set
		WHERE environment_id=? AND service_id=0 AND scope='environment' AND name='default'`, environmentID).Scan(
		&set.ID, &set.ProjectID, &set.EnvironmentID, &set.ServiceID, &set.Name, &set.Scope, &set.Revision,
		&set.CreatedBy, &set.CreatedAt, &set.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	items, err := r.listVariableItems(ctx, set.ID)
	if err != nil {
		return nil, err
	}
	set.Items = items
	return &set, nil
}

func (r *Repository) FindVariableSetByEnvironmentService(ctx context.Context, environmentID, serviceID int64) (*VariableSet, error) {
	var set VariableSet
	err := r.db.QueryRowContext(ctx, `SELECT id, project_id, environment_id, service_id, name, scope, revision,
		created_by, created_at, updated_at FROM variable_set
		WHERE environment_id=? AND service_id=? AND scope='environment' AND name='default'`, environmentID, serviceID).Scan(
		&set.ID, &set.ProjectID, &set.EnvironmentID, &set.ServiceID, &set.Name, &set.Scope, &set.Revision,
		&set.CreatedBy, &set.CreatedAt, &set.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	items, err := r.listVariableItems(ctx, set.ID)
	if err != nil {
		return nil, err
	}
	set.Items = items
	return &set, nil
}

func (r *Repository) EnsureServiceVariableSet(ctx context.Context, projectID, environmentID, serviceID int64) error {
	_, err := r.db.ExecContext(ctx, `INSERT IGNORE INTO variable_set
		(project_id, environment_id, service_id, name, scope, revision, created_by)
		VALUES (?, ?, ?, 'default', 'environment', 1, 0)`, projectID, environmentID, serviceID)
	return err
}

func (r *Repository) ListResources(ctx context.Context, environmentID int64) ([]EnvironmentResource, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, environment_id, name, kind, manifest, revision, created_by, created_at, updated_at
		FROM environment_resource WHERE environment_id=? AND kind NOT IN ('AxiOpsServiceSettings','AxiOpsHelmValues','AxiOpsHelmSharedValues') ORDER BY name`, environmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []EnvironmentResource
	for rows.Next() {
		var item EnvironmentResource
		if err := rows.Scan(&item.ID, &item.EnvironmentID, &item.Name, &item.Kind, &item.Manifest, &item.Revision, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) FindResource(ctx context.Context, environmentID, resourceID int64) (*EnvironmentResource, error) {
	var item EnvironmentResource
	err := r.db.QueryRowContext(ctx, `SELECT id, environment_id, name, kind, manifest, revision, created_by, created_at, updated_at
		FROM environment_resource WHERE environment_id=? AND id=? AND kind NOT IN ('AxiOpsServiceSettings','AxiOpsHelmValues','AxiOpsHelmSharedValues')`, environmentID, resourceID).Scan(
		&item.ID, &item.EnvironmentID, &item.Name, &item.Kind, &item.Manifest, &item.Revision, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) CreateResource(ctx context.Context, item *EnvironmentResource) (int64, error) {
	result, err := r.db.ExecContext(ctx, `INSERT INTO environment_resource(environment_id, name, kind, manifest, revision, created_by) VALUES (?, ?, ?, ?, 1, ?)`, item.EnvironmentID, item.Name, item.Kind, item.Manifest, item.CreatedBy)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *Repository) UpdateResource(ctx context.Context, item *EnvironmentResource, expectedRevision int64) (bool, error) {
	result, err := r.db.ExecContext(ctx, `UPDATE environment_resource SET name=?, kind=?, manifest=?, revision=revision+1, updated_at=CURRENT_TIMESTAMP WHERE id=? AND environment_id=? AND revision=?`, item.Name, item.Kind, item.Manifest, item.ID, item.EnvironmentID, expectedRevision)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	return affected == 1, err
}

func (r *Repository) DeleteResource(ctx context.Context, environmentID, resourceID int64) (bool, error) {
	result, err := r.db.ExecContext(ctx, `DELETE FROM environment_resource WHERE id=? AND environment_id=? AND kind NOT IN ('AxiOpsServiceSettings','AxiOpsHelmValues','AxiOpsHelmSharedValues')`, resourceID, environmentID)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	return affected == 1, err
}

func (r *Repository) FindVariableSetByID(ctx context.Context, id int64) (*VariableSet, error) {
	var set VariableSet
	err := r.db.QueryRowContext(ctx, `SELECT id, project_id, environment_id, service_id, name, scope, revision,
		created_by, created_at, updated_at FROM variable_set WHERE id=?`, id).Scan(
		&set.ID, &set.ProjectID, &set.EnvironmentID, &set.ServiceID, &set.Name, &set.Scope, &set.Revision,
		&set.CreatedBy, &set.CreatedAt, &set.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &set, err
}

func (r *Repository) listVariableItems(ctx context.Context, setID int64) ([]VariableItem, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, variable_key,
		CASE WHEN is_secret=1 THEN '' ELSE COALESCE(value_text, '') END, credential_id, is_secret
		FROM variable_item WHERE variable_set_id=? ORDER BY variable_key ASC`, setID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []VariableItem
	for rows.Next() {
		var item VariableItem
		if err := rows.Scan(&item.ID, &item.VariableKey, &item.Value, &item.CredentialID, &item.IsSecret); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) CredentialBelongsToProject(ctx context.Context, credentialID, projectID int64) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM credential WHERE id=? AND (project_id=0 OR project_id=?)`, credentialID, projectID).Scan(&count)
	return count == 1, err
}

func (r *Repository) ReplaceVariableItems(ctx context.Context, setID, expectedRevision, userID int64, items []VariableItemInput) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE variable_set SET revision=revision+1, updated_at=CURRENT_TIMESTAMP
		WHERE id=? AND revision=?`, setID, expectedRevision)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil || affected != 1 {
		return false, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM variable_item WHERE variable_set_id=?`, setID); err != nil {
		return false, err
	}
	for _, item := range items {
		if _, err := tx.ExecContext(ctx, `INSERT INTO variable_item
			(variable_set_id, variable_key, value_text, credential_id, is_secret, created_by)
			VALUES (?, ?, ?, ?, ?, ?)`, setID, strings.TrimSpace(item.VariableKey), item.Value,
			item.CredentialID, item.IsSecret, userID); err != nil {
			return false, err
		}
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

func (r *Repository) FindService(ctx context.Context, serviceID int64) (*ServiceTarget, error) {
	var service ServiceTarget
	err := r.db.QueryRowContext(ctx, `SELECT id, project_id, delivery_type FROM app_service WHERE id=? AND status='active'`, serviceID).
		Scan(&service.ID, &service.ProjectID, &service.DeliveryType)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &service, err
}

func (r *Repository) ServiceRevisionExists(ctx context.Context, serviceID, revisionID int64) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM service_revision WHERE id=? AND service_id=?`, revisionID, serviceID).Scan(&count)
	return count == 1, err
}

func (r *Repository) FindServiceOverride(ctx context.Context, environmentID, serviceID int64) (*EnvironmentServiceOverride, error) {
	var override EnvironmentServiceOverride
	var replicas sql.NullInt64
	err := r.db.QueryRowContext(ctx, `SELECT id, project_environment_id, service_id, service_revision_id,
		image_ref, replicas, revision, updated_by, created_at, updated_at
		FROM environment_service_override WHERE project_environment_id=? AND service_id=?`, environmentID, serviceID).Scan(
		&override.ID, &override.ProjectEnvironmentID, &override.ServiceID, &override.ServiceRevisionID,
		&override.ImageRef, &replicas, &override.Revision, &override.UpdatedBy, &override.CreatedAt, &override.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if replicas.Valid {
		value := int(replicas.Int64)
		override.Replicas = &value
	}
	return &override, nil
}

func (r *Repository) SaveServiceOverride(ctx context.Context, override *EnvironmentServiceOverride, expectedRevision int64) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	if err = servicestate.LockActive(ctx, tx, 0, override.ServiceID); err != nil {
		return false, err
	}
	var result sql.Result
	if expectedRevision == 0 {
		result, err = tx.ExecContext(ctx, `INSERT IGNORE INTO environment_service_override
			(project_environment_id, service_id, service_revision_id, image_ref, replicas, revision, updated_by)
			VALUES (?, ?, ?, ?, ?, 1, ?)`, override.ProjectEnvironmentID, override.ServiceID,
			override.ServiceRevisionID, override.ImageRef, override.Replicas, override.UpdatedBy)
	} else {
		result, err = tx.ExecContext(ctx, `UPDATE environment_service_override
			SET service_revision_id=?, image_ref=?, replicas=?, revision=revision+1, updated_by=?, updated_at=CURRENT_TIMESTAMP
			WHERE project_environment_id=? AND service_id=? AND revision=? AND (environment_yaml_revision_id=0 OR service_revision_id=?)`, override.ServiceRevisionID,
			override.ImageRef, override.Replicas, override.UpdatedBy, override.ProjectEnvironmentID,
			override.ServiceID, expectedRevision, override.ServiceRevisionID)
	}
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected == 1, tx.Commit()
}
