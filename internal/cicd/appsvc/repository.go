package appsvc

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/ChenJustin666/axiops/internal/cicd/servicestate"
	"github.com/ChenJustin666/axiops/internal/shared/authz"
)

// Repository 应用服务数据访问层
type Repository struct {
	db *sql.DB
}

var errRevisionChanged = errors.New("service revision changed")

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindByID(ctx context.Context, id int64) (*AppService, error) {
	query := `SELECT id, name, code, description, project_id, language, framework, repo_url, repo_branch, 
		code_source_id, registry_id, image_name, dockerfile_path, build_context, deploy_type,
		delivery_type, source_type, COALESCE(source_ref, JSON_OBJECT()), current_revision_id, component_count, status,
		owner_user_id, created_by, created_at, updated_at FROM app_service WHERE id = ?`
	var s AppService
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&s.ID, &s.Name, &s.Code, &s.Description, &s.ProjectID, &s.Language, &s.Framework,
		&s.RepoURL, &s.RepoBranch, &s.CodeSourceID, &s.RegistryID, &s.ImageName,
		&s.DockerfilePath, &s.BuildContext, &s.DeployType, &s.DeliveryType, &s.SourceType,
		&s.SourceRef, &s.CurrentRevisionID, &s.ComponentCount, &s.Status,
		&s.OwnerUserID, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *Repository) FindProjectDeliveryProfile(ctx context.Context, projectID int64) (*ProjectDeliveryProfile, error) {
	var profile ProjectDeliveryProfile
	err := r.db.QueryRowContext(ctx,
		`SELECT delivery_type, delivery_status FROM project WHERE id = ?`, projectID,
	).Scan(&profile.DeliveryType, &profile.DeliveryStatus)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &profile, nil
}

func (r *Repository) FindByProjectAndCode(ctx context.Context, projectID int64, code string) (*AppService, error) {
	query := `SELECT id FROM app_service WHERE project_id = ? AND code = ?`
	var id int64
	err := r.db.QueryRowContext(ctx, query, projectID, code).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return r.FindByID(ctx, id)
}

// ResolveImageRepository derives the push repository from the service's
// bound image registry and image name. It returns an empty value when the
// service has no registry binding, allowing callers to fail with a clear
// configuration error instead of producing a malformed image reference.
func (r *Repository) ResolveImageRepository(ctx context.Context, serviceID int64) (string, error) {
	var registryURL, imageName string
	err := r.db.QueryRowContext(ctx, `SELECT COALESCE(TRIM(TRAILING '/' FROM registry.url),''), service.image_name
		FROM app_service service LEFT JOIN image_registry registry ON registry.id=service.registry_id
			AND registry.status='active' AND registry.project_id IN (0, service.project_id)
		WHERE service.id=?`, serviceID).Scan(&registryURL, &imageName)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	registryURL = strings.TrimRight(strings.TrimSpace(registryURL), "/")
	imageName = strings.Trim(strings.TrimSpace(imageName), "/")
	if registryURL == "" || imageName == "" {
		return "", nil
	}
	if !strings.Contains(registryURL, "://") {
		registryURL = "https://" + registryURL
	}
	parsed, err := url.Parse(registryURL)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return "", fmt.Errorf("invalid image registry address")
	}
	registryURL = strings.TrimRight(parsed.Host+parsed.Path, "/")
	// Explicit references may use another configured namespace on this host.
	// Build execution validates the registry's namespace scope before using credentials.
	if strings.HasPrefix(imageName, parsed.Host+"/") {
		return imageName, nil
	}
	return registryURL + "/" + imageName, nil
}

// CodeSourceBelongsToProject allows a project to consume a platform source
// while preventing references to another project's private source.
func (r *Repository) CodeSourceBelongsToProject(ctx context.Context, sourceID, projectID int64) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM code_source WHERE id=? AND (project_id=0 OR project_id=?)`,
		sourceID, projectID).Scan(&count)
	return count == 1, err
}

func (r *Repository) RegistryBelongsToProject(ctx context.Context, registryID, projectID int64) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM image_registry WHERE id=? AND status='active' AND project_id IN (0,?)`, registryID, projectID).Scan(&count)
	return count == 1, err
}

func (r *Repository) List(ctx context.Context, scope authz.ScopeSet, projectID int64, keyword, status string, page, pageSize int) ([]AppService, int64, error) {
	var conditions []string
	var args []interface{}

	scopePredicate, scopeArgs := scope.SQLPredicate("project_id", "id", "")
	conditions = append(conditions, scopePredicate)
	args = append(args, scopeArgs...)

	if projectID > 0 {
		conditions = append(conditions, "project_id = ?")
		args = append(args, projectID)
	}
	if keyword != "" {
		conditions = append(conditions, "(name LIKE ? OR code LIKE ?)")
		like := "%" + keyword + "%"
		args = append(args, like, like)
	}
	if status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, status)
	} else {
		conditions = append(conditions, "status <> 'deleted'")
	}

	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}

	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM app_service"+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	dataQuery := `SELECT id, name, code, description, project_id, language, framework, repo_url, repo_branch, 
		code_source_id, registry_id, image_name, dockerfile_path, build_context, deploy_type,
		delivery_type, source_type, COALESCE(source_ref, JSON_OBJECT()), current_revision_id, component_count, status,
		owner_user_id, created_by, created_at, updated_at FROM app_service` + where + ` ORDER BY id DESC LIMIT ? OFFSET ?`
	args = append(args, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var services []AppService
	for rows.Next() {
		var s AppService
		if err := rows.Scan(
			&s.ID, &s.Name, &s.Code, &s.Description, &s.ProjectID, &s.Language, &s.Framework,
			&s.RepoURL, &s.RepoBranch, &s.CodeSourceID, &s.RegistryID, &s.ImageName,
			&s.DockerfilePath, &s.BuildContext, &s.DeployType, &s.DeliveryType, &s.SourceType,
			&s.SourceRef, &s.CurrentRevisionID, &s.ComponentCount, &s.Status,
			&s.OwnerUserID, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		services = append(services, s)
	}
	return services, total, nil
}

func (r *Repository) Create(ctx context.Context, s *AppService) (int64, error) {
	query := `INSERT INTO app_service (name, code, description, project_id, language, framework, repo_url, repo_branch, 
		code_source_id, registry_id, image_name, dockerfile_path, build_context, deploy_type,
		delivery_type, source_type, source_ref, status, owner_user_id, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query,
		s.Name, s.Code, s.Description, s.ProjectID, s.Language, s.Framework,
		s.RepoURL, s.RepoBranch, s.CodeSourceID, s.RegistryID, s.ImageName,
		s.DockerfilePath, s.BuildContext, s.DeployType, s.DeliveryType, s.SourceType, s.SourceRef,
		s.Status, s.OwnerUserID, s.CreatedBy,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *Repository) CreateRevision(ctx context.Context, revision *ServiceRevision) (int64, int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()
	id, number, err := CreateRevisionTx(ctx, tx, revision)
	if err != nil {
		return 0, 0, err
	}
	if err = tx.Commit(); err != nil {
		return 0, 0, err
	}
	return id, number, nil
}

// CreateRevisionTx allows an environment template upgrade and its binding to
// commit atomically. The caller owns the transaction and its rollback.
func CreateRevisionTx(ctx context.Context, tx *sql.Tx, revision *ServiceRevision) (int64, int64, error) {
	if err := servicestate.LockActive(ctx, tx, 0, revision.ServiceID); err != nil {
		return 0, 0, err
	}
	if revision.ExpectedRevisionID != nil {
		var current int64
		if err := tx.QueryRowContext(ctx, `SELECT COALESCE(current_revision_id,0) FROM app_service WHERE id=? FOR UPDATE`, revision.ServiceID).Scan(&current); err != nil {
			return 0, 0, err
		}
		if current != *revision.ExpectedRevisionID {
			return 0, 0, errRevisionChanged
		}
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO service_revision_sequence (service_id, next_revision)
		 SELECT ?,LAST_INSERT_ID(COALESCE(MAX(revision_number),0)+1) FROM service_revision WHERE service_id=?
		 ON DUPLICATE KEY UPDATE next_revision=LAST_INSERT_ID(next_revision + 1)`, revision.ServiceID, revision.ServiceID); err != nil {
		return 0, 0, err
	}
	var revisionNumber int64
	if err := tx.QueryRowContext(ctx, `SELECT LAST_INSERT_ID()`).Scan(&revisionNumber); err != nil {
		return 0, 0, err
	}
	result, err := tx.ExecContext(ctx,
		`INSERT INTO service_revision (service_id, revision_number, source_commit, definition, definition_format, content_hash, created_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		revision.ServiceID, revisionNumber, revision.SourceCommit, revision.Definition,
		revision.DefinitionFormat, revision.ContentHash, revision.CreatedBy)
	if err != nil {
		return 0, 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, 0, err
	}
	if revision.RepositoryBinding != nil {
		binding := *revision.RepositoryBinding
		binding.RevisionID = id
		raw, err := json.Marshal(binding)
		if err != nil {
			return 0, 0, err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE app_service SET source_ref=JSON_SET(COALESCE(source_ref,JSON_OBJECT()),'$.helm_repository',CAST(? AS JSON)) WHERE id=?`, string(raw), revision.ServiceID); err != nil {
			return 0, 0, err
		}
	}
	updateResult, err := tx.ExecContext(ctx, `UPDATE app_service SET current_revision_id=?, component_count=CASE WHEN ?>0 THEN ? ELSE component_count END, updated_at=? WHERE id=?`, id, revision.ComponentCount, revision.ComponentCount, time.Now(), revision.ServiceID)
	if err != nil {
		return 0, 0, err
	}
	affected, err := updateResult.RowsAffected()
	if err != nil || affected != 1 {
		if err == nil {
			err = sql.ErrNoRows
		}
		return 0, 0, err
	}
	return id, revisionNumber, nil
}

func (r *Repository) ListRevisions(ctx context.Context, serviceID int64, latest ...bool) ([]ServiceRevision, error) {
	suffix := ""
	if len(latest) > 0 && latest[0] {
		suffix = " LIMIT 1"
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, service_id, revision_number, source_commit, definition, definition_format, content_hash, created_by, created_at
		 FROM service_revision WHERE service_id=? ORDER BY revision_number DESC`+suffix, serviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var revisions []ServiceRevision
	for rows.Next() {
		var revision ServiceRevision
		if err := rows.Scan(&revision.ID, &revision.ServiceID, &revision.RevisionNumber, &revision.SourceCommit,
			&revision.Definition, &revision.DefinitionFormat, &revision.ContentHash, &revision.CreatedBy, &revision.CreatedAt); err != nil {
			return nil, err
		}
		revisions = append(revisions, revision)
	}
	return revisions, rows.Err()
}

func (r *Repository) Update(ctx context.Context, s *AppService) error {
	query := `UPDATE app_service SET name=?, description=?, language=?, framework=?, repo_url=?, repo_branch=?, 
		code_source_id=?, registry_id=?, image_name=?, dockerfile_path=?, build_context=?,
		status=?, owner_user_id=?, updated_at=? WHERE id=? AND status <> 'deleted'`
	_, err := r.db.ExecContext(ctx, query,
		s.Name, s.Description, s.Language, s.Framework, s.RepoURL, s.RepoBranch,
		s.CodeSourceID, s.RegistryID, s.ImageName, s.DockerfilePath, s.BuildContext,
		s.Status, s.OwnerUserID, time.Now(), s.ID,
	)
	return err
}
