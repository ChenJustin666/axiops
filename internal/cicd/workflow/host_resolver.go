package workflow

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type HostSource struct {
	ProjectID, ServiceID, ServiceRevisionID, EnvironmentID int64
	DeliveryType, DefinitionFormat, EnvironmentType        string
	BindingStatus, EnvironmentStatus, HostGroupRef         string
	Definition                                             []byte
}

type HostArtifact struct {
	URL, SHA256 string
	Size        int64
}

type HostSourceStore interface {
	LoadHostSource(context.Context, int64, int64, int64) (*HostSource, error)
	LoadHostBuildArtifact(context.Context, int64) (*HostArtifact, error)
	ListOnlineHostAgentIDs(context.Context, int64, string, string) ([]int64, error)
}

type DefaultHostResolver struct{ store HostSourceStore }

func NewDefaultHostResolver(store HostSourceStore) *DefaultHostResolver {
	return &DefaultHostResolver{store: store}
}

type HostSpec struct {
	OS             string `json:"os"`
	ArtifactURL    string `json:"artifact_url"`
	ArtifactSHA256 string `json:"artifact_sha256"`
	DeployScript   string `json:"deploy_script"`
	HealthScript   string `json:"health_script"`
	RollbackScript string `json:"rollback_script"`
	TimeoutSeconds int    `json:"timeout_seconds"`
	Rollout        struct {
		BatchSize       int `json:"batch_size,omitempty"`
		BatchPercent    int `json:"batch_percent,omitempty"`
		IntervalSeconds int `json:"interval_seconds,omitempty"`
	} `json:"rollout,omitempty"`
}

func (r *DefaultHostResolver) RenderHost(ctx context.Context, run WorkflowRun, config HostDeployConfig) (HostTarget, error) {
	if r == nil || r.store == nil {
		return HostTarget{}, fmt.Errorf("host source store is not configured")
	}
	source, err := r.store.LoadHostSource(ctx, run.EnvironmentID, config.ServiceID, config.ServiceRevisionID)
	if err != nil || source == nil {
		if err == nil {
			err = fmt.Errorf("host deployment source was not found")
		}
		return HostTarget{}, err
	}
	if frozen := config.FrozenHost; frozen != nil {
		if frozen.ProjectID != source.ProjectID || frozen.ServiceID != source.ServiceID || frozen.ServiceRevisionID != source.ServiceRevisionID || frozen.EnvironmentID != source.EnvironmentID || frozen.HostGroupRef != source.HostGroupRef {
			return HostTarget{}, fmt.Errorf("host deployment target changed after submission")
		}
		copy := *frozen
		copy.EnvironmentStatus = source.EnvironmentStatus
		copy.BindingStatus = source.BindingStatus
		source = &copy
	}
	if source.ProjectID != run.ProjectID || source.ServiceID != config.ServiceID || source.ServiceRevisionID != config.ServiceRevisionID || source.EnvironmentID != run.EnvironmentID {
		return HostTarget{}, fmt.Errorf("host deployment source does not belong to the workflow project")
	}
	if source.DeliveryType != DeliveryTypeHost || source.DefinitionFormat != "host_spec" || source.EnvironmentType != "host_group" || source.BindingStatus != "validated" || source.EnvironmentStatus != "active" {
		return HostTarget{}, fmt.Errorf("host environment or service revision is not deployable")
	}
	var spec HostSpec
	if json.Unmarshal(source.Definition, &spec) != nil || (spec.OS != "linux" && spec.OS != "windows") || strings.TrimSpace(spec.DeployScript) == "" || len(spec.DeployScript) > 256<<10 || len(spec.HealthScript) > 64<<10 || len(spec.RollbackScript) > 256<<10 {
		return HostTarget{}, fmt.Errorf("invalid host service specification")
	}
	artifact := HostArtifact{URL: strings.TrimSpace(spec.ArtifactURL), SHA256: strings.ToLower(strings.TrimSpace(spec.ArtifactSHA256))}
	if artifact.URL == "" {
		buildID, err := matchingBuildID(run.Tasks, config.ServiceID)
		if err != nil {
			return HostTarget{}, err
		}
		if buildID <= 0 {
			return HostTarget{}, fmt.Errorf("host deployment requires a successful build artifact")
		}
		resolved, err := r.store.LoadHostBuildArtifact(ctx, buildID)
		if err != nil || resolved == nil {
			if err == nil {
				err = fmt.Errorf("successful build has no host artifact")
			}
			return HostTarget{}, err
		}
		artifact = *resolved
	}
	if err := validateHostArtifact(artifact); err != nil {
		return HostTarget{}, err
	}
	agents, err := r.store.ListOnlineHostAgentIDs(ctx, run.ProjectID, source.HostGroupRef, spec.OS)
	if err != nil {
		return HostTarget{}, err
	}
	if len(agents) == 0 {
		return HostTarget{}, fmt.Errorf("host group %q has no online %s Agent", source.HostGroupRef, spec.OS)
	}
	if config.FrozenHost != nil {
		if len(config.FrozenAgentIDs) == 0 {
			return HostTarget{}, fmt.Errorf("host target snapshot has no agents")
		}
		online := make(map[int64]bool, len(agents))
		for _, id := range agents {
			online[id] = true
		}
		for _, id := range config.FrozenAgentIDs {
			if !online[id] {
				return HostTarget{}, fmt.Errorf("目标 Agent %d 已离线或移出主机组，已阻止替换目标", id)
			}
		}
		agents = append([]int64(nil), config.FrozenAgentIDs...)
	}
	timeout := spec.TimeoutSeconds
	if timeout <= 0 {
		timeout = 900
	}
	if timeout > 86400 {
		return HostTarget{}, fmt.Errorf("host deployment timeout exceeds 86400 seconds")
	}
	batchSize, err := resolveHostBatchSize(len(agents), spec.Rollout.BatchSize, spec.Rollout.BatchPercent)
	if err != nil {
		return HostTarget{}, err
	}
	if spec.Rollout.IntervalSeconds < 0 || spec.Rollout.IntervalSeconds > 3600 {
		return HostTarget{}, fmt.Errorf("host rollout interval must be between 0 and 3600 seconds")
	}
	return HostTarget{
		AgentIDs: agents, ArtifactURL: artifact.URL, ArtifactSHA256: artifact.SHA256,
		DeployScript: spec.DeployScript, HealthScript: spec.HealthScript, RollbackScript: spec.RollbackScript,
		TimeoutSeconds: timeout, BatchSize: batchSize, BatchIntervalSeconds: spec.Rollout.IntervalSeconds,
	}, nil
}

func resolveHostBatchSize(total, batchSize, batchPercent int) (int, error) {
	if total <= 0 {
		return 0, fmt.Errorf("host rollout has no targets")
	}
	if batchSize > 0 && batchPercent > 0 {
		return 0, fmt.Errorf("host rollout cannot set both batch size and percentage")
	}
	if batchSize < 0 || batchSize > 1000 || batchPercent < 0 || batchPercent > 100 {
		return 0, fmt.Errorf("host rollout batch policy is invalid")
	}
	if batchPercent > 0 {
		batchSize = (total*batchPercent + 99) / 100
	}
	if batchSize == 0 {
		batchSize = 1
	}
	if batchSize > total {
		batchSize = total
	}
	return batchSize, nil
}

func validateHostArtifact(artifact HostArtifact) error {
	parsed, err := url.Parse(artifact.URL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return fmt.Errorf("host artifact must use an absolute HTTPS URL")
	}
	if len(artifact.SHA256) != 64 {
		return fmt.Errorf("host artifact SHA-256 is required")
	}
	if _, err := hex.DecodeString(artifact.SHA256); err != nil {
		return fmt.Errorf("host artifact SHA-256 is invalid")
	}
	if artifact.Size < 0 {
		return fmt.Errorf("host artifact size is invalid")
	}
	return nil
}

type SQLHostSourceStore struct{ db sourceQuery }

func NewSQLHostSourceStore(db *sql.DB) *SQLHostSourceStore { return &SQLHostSourceStore{db: db} }

func (s *SQLHostSourceStore) LoadHostSource(ctx context.Context, environmentID, serviceID, revisionID int64) (*HostSource, error) {
	var source HostSource
	err := s.db.QueryRowContext(ctx, `SELECT env.project_id,svc.id,rev.id,env.id,svc.delivery_type,rev.definition_format,env.environment_type,env.binding_status,env.status,env.host_group_ref,rev.definition FROM project_environment env JOIN app_service svc ON svc.project_id=env.project_id AND svc.id=? AND svc.status='active' JOIN service_revision rev ON rev.service_id=svc.id AND rev.id=? WHERE env.id=?`, serviceID, revisionID, environmentID).Scan(&source.ProjectID, &source.ServiceID, &source.ServiceRevisionID, &source.EnvironmentID, &source.DeliveryType, &source.DefinitionFormat, &source.EnvironmentType, &source.BindingStatus, &source.EnvironmentStatus, &source.HostGroupRef, &source.Definition)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &source, err
}

func (s *SQLHostSourceStore) LoadHostBuildArtifact(ctx context.Context, buildID int64) (*HostArtifact, error) {
	var artifact HostArtifact
	err := s.db.QueryRowContext(ctx, `SELECT artifact_url,artifact_sha256,artifact_size FROM build_run WHERE id=? AND status='success' AND artifact_verification_status='verified'`, buildID).Scan(&artifact.URL, &artifact.SHA256, &artifact.Size)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &artifact, err
}

func (s *SQLHostSourceStore) ListOnlineHostAgentIDs(ctx context.Context, projectID int64, group, osName string) ([]int64, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM host_agent WHERE project_id=? AND host_group_ref=? AND LOWER(os)=? AND status='online' AND last_heartbeat>=DATE_SUB(CURRENT_TIMESTAMP,INTERVAL 3 MINUTE) ORDER BY id`, projectID, group, osName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
