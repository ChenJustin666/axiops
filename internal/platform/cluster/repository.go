package cluster

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
)

// Repository 集群数据访问层
type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindByID(ctx context.Context, id int64) (*Cluster, error) {
	query := `SELECT id, name, code, description, provider, region, version, api_server, kubeconfig, status, node_count, cpu_total, memory_total, last_heartbeat, created_by, created_at, updated_at FROM cluster WHERE id = ?`
	var c Cluster
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&c.ID, &c.Name, &c.Code, &c.Description, &c.Provider, &c.Region,
		&c.Version, &c.APIServer, &c.Kubeconfig, &c.Status, &c.NodeCount,
		&c.CPUTotal, &c.MemoryTotal, &c.LastHeartbeat, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *Repository) FindByCode(ctx context.Context, code string) (*Cluster, error) {
	query := `SELECT id, name, code, description, provider, region, version, api_server, kubeconfig, status, node_count, cpu_total, memory_total, last_heartbeat, created_by, created_at, updated_at FROM cluster WHERE code = ?`
	var c Cluster
	err := r.db.QueryRowContext(ctx, query, code).Scan(
		&c.ID, &c.Name, &c.Code, &c.Description, &c.Provider, &c.Region,
		&c.Version, &c.APIServer, &c.Kubeconfig, &c.Status, &c.NodeCount,
		&c.CPUTotal, &c.MemoryTotal, &c.LastHeartbeat, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *Repository) List(ctx context.Context, scope authz.ScopeSet, projectID ...int64) ([]Cluster, error) {
	predicate, args := scope.SQLIDPredicate(authz.ScopeCluster, "id")
	requestedProject := int64(0)
	if len(projectID) > 0 {
		requestedProject = projectID[0]
	}
	// A project-scoped grant may see clusters already attached to that project,
	// while global cluster management remains limited to cluster/platform grants.
	if requestedProject > 0 && !scope.AllowAll && !scope.DenyAll && !containsID(scope.DeniedProjects, requestedProject) && containsID(scope.ProjectIDs, requestedProject) {
		predicate = "(" + predicate + " OR id IN (SELECT cluster_id FROM project_environment WHERE project_id = ?))"
		args = append(args, requestedProject)
		if len(scope.DeniedClusters) > 0 {
			placeholders := make([]string, len(scope.DeniedClusters))
			for i, id := range scope.DeniedClusters {
				placeholders[i] = "?"
				args = append(args, id)
			}
			predicate += " AND NOT (id IN (" + strings.Join(placeholders, ",") + "))"
		}
	}
	query := `SELECT id, name, code, description, provider, region, version, api_server, '', status, node_count, cpu_total, memory_total, last_heartbeat, created_by, created_at, updated_at FROM cluster WHERE ` + predicate + ` ORDER BY id ASC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Return an initialized slice so JSON callers receive [] for an empty scope.
	clusters := make([]Cluster, 0)
	for rows.Next() {
		var c Cluster
		if err := rows.Scan(
			&c.ID, &c.Name, &c.Code, &c.Description, &c.Provider, &c.Region,
			&c.Version, &c.APIServer, &c.Kubeconfig, &c.Status, &c.NodeCount,
			&c.CPUTotal, &c.MemoryTotal, &c.LastHeartbeat, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		clusters = append(clusters, c)
	}
	return clusters, nil
}

func containsID(ids []int64, target int64) bool {
	for _, id := range ids {
		if id == target {
			return true
		}
	}
	return false
}

func (r *Repository) Create(ctx context.Context, c *Cluster) (int64, error) {
	query := `INSERT INTO cluster (name, code, description, provider, region, api_server, kubeconfig, status, created_by) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query, c.Name, c.Code, c.Description, c.Provider, c.Region, c.APIServer, c.Kubeconfig, c.Status, c.CreatedBy)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *Repository) Update(ctx context.Context, c *Cluster) error {
	query := `UPDATE cluster SET name=?, description=?, api_server=?, kubeconfig=?, updated_at=? WHERE id=?`
	_, err := r.db.ExecContext(ctx, query, c.Name, c.Description, c.APIServer, c.Kubeconfig, time.Now(), c.ID)
	return err
}

func (r *Repository) UpdateStatus(ctx context.Context, id int64, status, version string, nodeCount int, cpuTotal, memoryTotal string) error {
	query := `UPDATE cluster SET status=?, version=?, node_count=?, cpu_total=?, memory_total=?, last_heartbeat=?, updated_at=? WHERE id=?`
	now := time.Now()
	_, err := r.db.ExecContext(ctx, query, status, version, nodeCount, cpuTotal, memoryTotal, now, now, id)
	return err
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM cluster WHERE id = ?", id)
	return err
}

// HasDeliveryReferences prevents deleting a cluster that remains part of a
// deployment environment or an immutable release record.
func (r *Repository) HasDeliveryReferences(ctx context.Context, id int64) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM (
			SELECT id FROM project_environment WHERE cluster_id = ?
			UNION ALL
			SELECT id FROM release_record WHERE cluster_id = ?
		) refs`, id, id).Scan(&count)
	return count > 0, err
}
