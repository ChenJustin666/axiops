package workflow

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/ChenJustin666/axiops/internal/cicd/helmconfig"
	"github.com/ChenJustin666/axiops/internal/cicd/helmimport"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/chartutil"
)

var helmValuePathPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+(?:\.[A-Za-z0-9_-]+)*$`)

type HelmSource struct {
	IsProduction                                                      bool                 `json:"is_production,omitempty"`
	Settings                                                          *helmconfig.Settings `json:"settings,omitempty"`
	ProjectID, ServiceID, ServiceRevisionID, EnvironmentID, ClusterID int64
	DeliveryType, DefinitionFormat, EnvironmentType                   string
	BindingStatus, EnvironmentStatus, Namespace                       string
	Definition                                                        []byte
}

type HelmVariable struct {
	Key, Value, CredentialData, CredentialType string
	CredentialID, CredentialProjectID          int64
	Secret                                     bool
}

type HelmSourceStore interface {
	LoadHelmSource(context.Context, int64, int64, int64) (*HelmSource, error)
	LoadHelmVariables(context.Context, int64, int64, int64) ([]HelmVariable, error)
}

type DefaultHelmResolver struct {
	store     HelmSourceStore
	decryptor CredentialDecryptor
}

func NewDefaultHelmResolver(store HelmSourceStore, decryptor CredentialDecryptor) *DefaultHelmResolver {
	return &DefaultHelmResolver{store: store, decryptor: decryptor}
}

func (r *DefaultHelmResolver) RenderHelm(ctx context.Context, run WorkflowRun, config HelmDeployConfig) (HelmTarget, error) {
	return r.render(ctx, run, config, false)
}

// RenderHelmConfig validates configuration before a future build exists.
func (r *DefaultHelmResolver) RenderHelmConfig(ctx context.Context, run WorkflowRun, config HelmDeployConfig) (HelmTarget, error) {
	return r.render(ctx, run, config, true)
}

func (r *DefaultHelmResolver) render(ctx context.Context, run WorkflowRun, config HelmDeployConfig, configOnly bool) (HelmTarget, error) {
	if r == nil || r.store == nil {
		return HelmTarget{}, fmt.Errorf("helm source resolver is not configured")
	}
	source, err := r.store.LoadHelmSource(ctx, run.EnvironmentID, config.ServiceID, config.ServiceRevisionID)
	if err != nil || source == nil {
		if err == nil {
			err = fmt.Errorf("helm deployment source was not found")
		}
		return HelmTarget{}, err
	}
	if frozen := config.FrozenHelm; frozen != nil {
		if frozen.ProjectID != source.ProjectID || frozen.ServiceID != source.ServiceID || frozen.ServiceRevisionID != source.ServiceRevisionID || frozen.EnvironmentID != source.EnvironmentID || frozen.ClusterID != source.ClusterID || frozen.Namespace != source.Namespace || frozen.IsProduction != source.IsProduction || source.EnvironmentStatus != "active" || source.BindingStatus != "validated" {
			return HelmTarget{}, fmt.Errorf("environment target changed after submission; submit a new run")
		}
		source = frozen
	}
	if source.ProjectID != run.ProjectID || source.ServiceID != config.ServiceID || source.ServiceRevisionID != config.ServiceRevisionID || source.EnvironmentID != run.EnvironmentID {
		return HelmTarget{}, fmt.Errorf("helm deployment source does not belong to the workflow project")
	}
	if source.DeliveryType != DeliveryTypeHelm || source.DefinitionFormat != "helm_chart" || source.EnvironmentType != "k8s_namespace" || source.BindingStatus != "validated" || source.EnvironmentStatus != "active" {
		return HelmTarget{}, fmt.Errorf("helm environment or service revision is not deployable")
	}
	if source.ClusterID <= 0 || strings.TrimSpace(source.Namespace) == "" || chartutil.ValidateReleaseName(config.ReleaseName) != nil {
		return HelmTarget{}, fmt.Errorf("helm cluster, namespace, or release name is invalid")
	}
	var definition helmimport.Definition
	if json.Unmarshal(source.Definition, &definition) != nil || definition.SchemaVersion != 1 || len(definition.Files) == 0 {
		return HelmTarget{}, fmt.Errorf("invalid immutable Helm Chart definition")
	}
	variables, err := r.store.LoadHelmVariables(ctx, source.ProjectID, source.EnvironmentID, source.ServiceID)
	if err != nil {
		return HelmTarget{}, err
	}
	if config.FrozenHelm != nil {
		variables = append([]HelmVariable(nil), config.FrozenHelmVariables...)
		for i := range variables {
			if variables[i].Secret {
				loader, ok := r.store.(interface {
					LoadFrozenCredential(context.Context, *HelmVariable) error
				})
				if !ok {
					return HelmTarget{}, fmt.Errorf("frozen credential loader unavailable")
				}
				if err = loader.LoadFrozenCredential(ctx, &variables[i]); err != nil {
					return HelmTarget{}, err
				}
			}
		}
	}
	// Resolve precedence before decrypting: an overridden credential must not be
	// read using the selected field of its replacement.
	effective := map[string]HelmVariable{}
	keys := []string{}
	for _, v := range variables {
		if _, exists := effective[v.Key]; !exists {
			keys = append(keys, v.Key)
		}
		effective[v.Key] = v
	}
	variables = nil
	for _, key := range keys {
		variables = append(variables, effective[key])
	}
	settings := helmconfig.Default()
	if source.Settings != nil {
		settings = *source.Settings
	}
	if err := settings.Validate(); err != nil {
		return HelmTarget{}, err
	}
	runtimeValues := map[string]interface{}{}
	snapshotValues := map[string]interface{}{}
	for _, variable := range variables {
		if !helmValuePathPattern.MatchString(variable.Key) {
			return HelmTarget{}, fmt.Errorf("invalid Helm value path %q", variable.Key)
		}
		value := interface{}(parseHelmScalar(variable.Value))
		snapshotValue := value
		if variable.Secret {
			if r.decryptor == nil {
				return HelmTarget{}, fmt.Errorf("Helm credential decryptor is not configured")
			}
			if variable.CredentialID <= 0 || variable.CredentialType != "secret" || variable.CredentialProjectID != 0 && variable.CredentialProjectID != source.ProjectID {
				return HelmTarget{}, fmt.Errorf("Helm secret value %q has an invalid credential", variable.Key)
			}
			plaintext, err := r.decryptor.Decrypt(variable.CredentialData)
			if err != nil {
				return HelmTarget{}, fmt.Errorf("decrypt Helm secret value %q: credential authentication failed", variable.Key)
			}
			key := settings.CredentialKeys[variable.Key]
			if key == "" {
				key = "value"
			}
			secret, err := helmSecretValue(plaintext, key)
			if _, placeholder := r.decryptor.(helmPreviewDecryptor); placeholder {
				secret = "axiops-credential-redacted"
				err = nil
			}
			if err != nil {
				return HelmTarget{}, fmt.Errorf("Helm secret value %q is invalid", variable.Key)
			}
			value = secret
			snapshotValue = map[string]interface{}{"credential_id": variable.CredentialID, "key": key}
		}
		if err := setHelmValue(runtimeValues, variable.Key, value); err != nil {
			return HelmTarget{}, err
		}
		if err := setHelmValue(snapshotValues, variable.Key, snapshotValue); err != nil {
			return HelmTarget{}, err
		}
	}
	overrides, err := helmconfig.ParseValues(settings.Values)
	if err != nil {
		return HelmTarget{}, err
	}
	shared, err := helmconfig.ParseValues(settings.SharedValues)
	if err != nil {
		return HelmTarget{}, err
	}
	overrides = MergeHelmValues(shared, overrides)
	for _, v := range variables {
		if v.Secret && helmValueOverridden(overrides, v.Key) {
			return HelmTarget{}, fmt.Errorf("Helm values 路径 %s 已绑定凭据，请通过凭据管理修改", v.Key)
		}
		if v.Secret {
			for _, path := range []string{settings.Image.RefPath, settings.Image.RepositoryPath, settings.Image.TagPath} {
				if path != "" && helmconfig.PathsOverlap(path, v.Key) {
					return HelmTarget{}, fmt.Errorf("Helm 镜像映射不能覆盖凭据路径 %s", v.Key)
				}
			}
		}
	}
	runtimeValues = MergeHelmValues(runtimeValues, overrides)
	snapshotValues = MergeHelmValues(snapshotValues, overrides)
	imageRef := config.ImageRef
	buildID, err := matchingBuildID(run.Tasks, config.ServiceID)
	if err != nil {
		return HelmTarget{}, err
	}
	if buildID > 0 {
		if imageRef != "" {
			return HelmTarget{}, fmt.Errorf("不能同时指定构建输出和已有镜像")
		}
		loader, ok := r.store.(interface {
			LoadBuildArtifact(context.Context, int64) (*BuildArtifact, error)
		})
		if !ok {
			return HelmTarget{}, fmt.Errorf("Helm build artifact loader unavailable")
		}
		artifact, e := loader.LoadBuildArtifact(ctx, buildID)
		if e != nil {
			return HelmTarget{}, e
		}
		if artifact == nil || artifact.ProjectID != run.ProjectID || artifact.ServiceID != config.ServiceID || !validPromotionImage(&PromotionBuild{ImageURL: artifact.ImageURL, ImageDigest: artifact.ImageDigest}) {
			return HelmTarget{}, fmt.Errorf("构建没有生成有效的镜像摘要")
		}
		imageRef = artifact.ImageURL + "@" + artifact.ImageDigest
	}
	if imageRef != "" {
		if err := injectHelmImage(runtimeValues, settings.Image, imageRef); err != nil {
			return HelmTarget{}, err
		}
		if err := injectHelmImage(snapshotValues, settings.Image, imageRef); err != nil {
			return HelmTarget{}, err
		}
	}
	if source.IsProduction && !configOnly && imageRef == "" {
		return HelmTarget{}, fmt.Errorf("生产 Helm 发布必须使用成功构建的固定摘要镜像，请选择已有镜像或构建并部署")
	}
	c, err := helmconfig.LoadChart(source.Definition)
	if err != nil {
		return HelmTarget{}, err
	}
	if err = helmconfig.ValidateWorkloadValues(c, runtimeValues, overrides); err != nil {
		return HelmTarget{}, err
	}
	return HelmTarget{
		ImageRef:  imageRef,
		ClusterID: source.ClusterID, Namespace: source.Namespace, ReleaseName: config.ReleaseName,
		ChartPath: definition.ChartPath, Chart: append([]byte(nil), source.Definition...), Values: runtimeValues,
		SnapshotValues: snapshotValues, TimeoutSeconds: 600,
	}, nil
}

func helmValueOverridden(values map[string]interface{}, path string) bool {
	parts := strings.Split(path, ".")
	for i, part := range parts {
		value, exists := values[part]
		if !exists {
			return false
		}
		if i == len(parts)-1 {
			return true
		}
		nested, ok := value.(map[string]interface{})
		if !ok {
			return true
		}
		values = nested
	}
	return false
}

func injectHelmImage(values map[string]interface{}, binding helmconfig.ImageBinding, imageRef string) error {
	image, digest, ok := strings.Cut(imageRef, "@")
	if !ok || !validPromotionImage(&PromotionBuild{ImageURL: image, ImageDigest: digest}) {
		return fmt.Errorf("Helm 部署镜像必须固定为有效 sha256 摘要")
	}
	if binding.RefPath != "" {
		return setHelmValue(values, binding.RefPath, imageRef)
	}
	slash, colon := strings.LastIndex(image, "/"), strings.LastIndex(image, ":")
	tag := "latest"
	if colon > slash {
		tag, image = image[colon+1:], image[:colon]
	}
	if err := setHelmValue(values, binding.RepositoryPath, image); err != nil {
		return err
	}
	// repo:tag@digest works with ordinary repository:tag Helm templates too.
	return setHelmValue(values, binding.TagPath, tag+"@"+digest)
}

func parseHelmScalar(value string) interface{} {
	var parsed interface{}
	if yaml.Unmarshal([]byte(value), &parsed) == nil && parsed != nil {
		switch parsed.(type) {
		case string, bool, int, int64, uint64, float64:
			return parsed
		}
	}
	return value
}

func helmSecretValue(plaintext string, selected ...string) (string, error) {
	key := "value"
	if len(selected) > 0 {
		key = selected[0]
	}
	var fields map[string]interface{}
	if json.Unmarshal([]byte(plaintext), &fields) != nil {
		return "", fmt.Errorf("invalid credential fields")
	}
	value, ok := fields[key].(string)
	if !ok {
		return "", fmt.Errorf("selected credential field is missing or not a string")
	}
	return value, nil
}

func setHelmValue(target map[string]interface{}, path string, value interface{}) error {
	parts := strings.Split(path, ".")
	cursor := target
	for _, part := range parts[:len(parts)-1] {
		next, exists := cursor[part]
		if !exists {
			child := map[string]interface{}{}
			cursor[part] = child
			cursor = child
			continue
		}
		child, ok := next.(map[string]interface{})
		if !ok {
			return fmt.Errorf("Helm value path %q conflicts with another value", path)
		}
		cursor = child
	}
	leaf := parts[len(parts)-1]
	cursor[leaf] = value
	return nil
}

type SQLHelmSourceStore struct{ db sourceQuery }

func NewSQLHelmSourceStore(db *sql.DB) *SQLHelmSourceStore { return &SQLHelmSourceStore{db: db} }

func ValidateHelmServiceConfig(ctx context.Context, tx *sql.Tx, decryptor CredentialDecryptor, projectID, environmentID, serviceID, revisionID int64) error {
	_, err := RenderHelmServiceConfig(ctx, tx, decryptor, projectID, environmentID, serviceID, revisionID)
	return err
}

func RenderHelmServiceConfig(ctx context.Context, tx *sql.Tx, decryptor CredentialDecryptor, projectID, environmentID, serviceID, revisionID int64) ([]byte, error) {
	manifest, _, err := RenderHelmServicePreview(ctx, tx, decryptor, projectID, environmentID, serviceID, revisionID)
	return manifest, err
}

func RenderHelmServicePreview(ctx context.Context, tx *sql.Tx, decryptor CredentialDecryptor, projectID, environmentID, serviceID, revisionID int64) ([]byte, string, error) {
	var name string
	if err := tx.QueryRowContext(ctx, `SELECT code FROM app_service WHERE id=?`, serviceID).Scan(&name); err != nil {
		return nil, "", err
	}
	target, err := NewDefaultHelmResolver(&SQLHelmSourceStore{db: tx}, decryptor).RenderHelmConfig(ctx, WorkflowRun{ProjectID: projectID, EnvironmentID: environmentID}, HelmDeployConfig{ServiceID: serviceID, ServiceRevisionID: revisionID, ReleaseName: name})
	if err != nil {
		return nil, "", err
	}
	manifest, err := renderHelmManifest(ctx, HelmRequest{RunID: 1, NodeKey: "config-preview", ClusterID: target.ClusterID, Namespace: target.Namespace, ReleaseName: target.ReleaseName, Chart: target.Chart, ChartPath: target.ChartPath, Values: target.Values, TimeoutSeconds: 600, IdempotencyKey: "config-preview"}, false)
	if err != nil {
		return nil, "", err
	}
	if _, err = helmManifestInventory(manifest, target.Namespace); err != nil {
		return nil, "", err
	}
	// Render the public preview with placeholders, so template transforms such
	// as b64enc cannot expose credential payloads outside Secret resources.
	previewValues := redactedHelmValues(target.Values, target.SnapshotValues)
	manifest, err = renderHelmManifest(ctx, HelmRequest{Namespace: target.Namespace, ReleaseName: target.ReleaseName, Chart: target.Chart, Values: previewValues}, false)
	if err != nil {
		return nil, "", fmt.Errorf("Helm 预览脱敏渲染失败，请检查凭据引用方式")
	}
	manifest, err = redactedHelmPreview(manifest, target.Namespace, previewValues, target.SnapshotValues)
	if err != nil {
		return nil, "", err
	}
	c, err := helmconfig.LoadChart(target.Chart)
	if err != nil {
		return nil, "", err
	}
	values, err := chartutil.CoalesceValues(c, previewValues)
	if err != nil {
		return nil, "", err
	}
	raw, err := yaml.Marshal(values)
	return manifest, string(raw), err
}

func (s *SQLHelmSourceStore) LoadHelmSource(ctx context.Context, environmentID, serviceID, revisionID int64) (*HelmSource, error) {
	var source HelmSource
	err := s.db.QueryRowContext(ctx, `SELECT env.project_id,svc.id,rev.id,env.id,env.cluster_id,svc.delivery_type,rev.definition_format,env.environment_type,env.binding_status,env.status,env.namespace,rev.definition,env.is_production
		FROM project_environment env JOIN app_service svc ON svc.project_id=env.project_id AND svc.id=? AND svc.status='active'
		JOIN service_revision rev ON rev.service_id=svc.id AND rev.id=?
		JOIN environment_service_override ov ON ov.service_id=svc.id AND ov.project_environment_id=env.id AND ov.service_revision_id=rev.id WHERE env.id=?`, serviceID, revisionID, environmentID).Scan(
		&source.ProjectID, &source.ServiceID, &source.ServiceRevisionID, &source.EnvironmentID, &source.ClusterID,
		&source.DeliveryType, &source.DefinitionFormat, &source.EnvironmentType, &source.BindingStatus, &source.EnvironmentStatus,
		&source.Namespace, &source.Definition, &source.IsProduction,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err == nil {
		settings, e := helmconfig.Load(ctx, s.db, environmentID, serviceID)
		if e != nil {
			return nil, e
		}
		source.Settings = &settings
	}
	return &source, err
}

func (s *SQLHelmSourceStore) LoadHelmVariables(ctx context.Context, projectID, environmentID, serviceID int64) ([]HelmVariable, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT vi.variable_key,COALESCE(vi.value_text,''),vi.credential_id,vi.is_secret,
		COALESCE(c.data,''),COALESCE(c.type,''),COALESCE(c.project_id,0)
		FROM variable_set vs JOIN variable_item vi ON vi.variable_set_id=vs.id
		LEFT JOIN credential c ON c.id=vi.credential_id
		WHERE (vs.scope='project' AND vs.project_id=? AND vs.environment_id=0 AND vs.service_id=0)
		   OR (vs.scope='environment' AND vs.project_id=? AND vs.environment_id=? AND vs.service_id IN (0, ?))
		   OR (vs.scope='service' AND vs.project_id=? AND vs.service_id=? AND (vs.environment_id=0 OR vs.environment_id=?))
		ORDER BY FIELD(vs.scope,'project','service','environment'),vs.service_id,(vs.name='default'),vs.id,vi.variable_key`, projectID, projectID, environmentID, serviceID, projectID, serviceID, environmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var variables []HelmVariable
	for rows.Next() {
		var item HelmVariable
		if err := rows.Scan(&item.Key, &item.Value, &item.CredentialID, &item.Secret, &item.CredentialData, &item.CredentialType, &item.CredentialProjectID); err != nil {
			return nil, err
		}
		variables = append(variables, item)
	}
	return variables, rows.Err()
}

func (s *SQLHelmSourceStore) LoadFrozenCredential(ctx context.Context, v *HelmVariable) error {
	return s.db.QueryRowContext(ctx, `SELECT data,type,project_id FROM credential WHERE id=?`, v.CredentialID).Scan(&v.CredentialData, &v.CredentialType, &v.CredentialProjectID)
}

func (s *SQLHelmSourceStore) LoadBuildArtifact(ctx context.Context, id int64) (*BuildArtifact, error) {
	return (&SQLYAMLSourceStore{db: s.db}).LoadBuildArtifact(ctx, id)
}
