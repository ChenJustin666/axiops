package workflow

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/ChenJustin666/axiops/internal/cicd/serviceassets"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

type YAMLSource struct {
	PreservedImages     map[string]string `json:"preserved_images,omitempty"`
	ProjectID           int64
	ServiceID           int64
	ServiceRevisionID   int64
	DeliveryType        string
	SourceType          string
	DefinitionFormat    string
	EnvironmentID       int64
	EnvironmentType     string
	BindingStatus       string
	EnvironmentStatus   string
	ClusterID           int64
	Namespace           string
	Manifest            []byte
	ImageName           string
	ImageRef            string
	Replicas            *int
	ProductName         string
	ServiceName         string
	EnvironmentName     string
	ClusterName         string
	EnvironmentManifest []byte
	Variables           map[string]string
	Assets              serviceassets.Settings
}

type BuildArtifact struct {
	ProjectID   int64
	ServiceID   int64
	ImageURL    string
	ImageDigest string
}

type YAMLSourceStore interface {
	LoadYAMLSource(context.Context, int64, int64, int64) (*YAMLSource, error)
	LoadBuildArtifact(context.Context, int64) (*BuildArtifact, error)
}

type DefaultYAMLResolver struct{ store YAMLSourceStore }

func NewDefaultYAMLResolver(store YAMLSourceStore) *DefaultYAMLResolver {
	return &DefaultYAMLResolver{store: store}
}

func (r *DefaultYAMLResolver) RenderYAML(ctx context.Context, run WorkflowRun, config YAMLDeployConfig) (YAMLTarget, error) {
	return r.renderYAML(ctx, run, config, false)
}

// RenderYAMLConfig validates configuration before a build has supplied Image.
// Its temporary image exists only in the returned validation manifest; callers
// must never persist or deploy this manifest.
func (r *DefaultYAMLResolver) RenderYAMLConfig(ctx context.Context, run WorkflowRun, config YAMLDeployConfig) (YAMLTarget, error) {
	return r.renderYAML(ctx, run, config, true)
}

func (r *DefaultYAMLResolver) renderYAML(ctx context.Context, run WorkflowRun, config YAMLDeployConfig, configOnly bool) (YAMLTarget, error) {
	if r == nil || r.store == nil {
		return YAMLTarget{}, fmt.Errorf("yaml source store is not configured")
	}
	source, err := r.store.LoadYAMLSource(ctx, run.EnvironmentID, config.ServiceID, config.ServiceRevisionID)
	if err != nil {
		return YAMLTarget{}, err
	}
	if source == nil {
		return YAMLTarget{}, fmt.Errorf("yaml deployment source was not found")
	}
	if frozen := config.FrozenYAML; frozen != nil {
		if frozen.ProjectID != source.ProjectID || frozen.ServiceID != source.ServiceID || frozen.ServiceRevisionID != source.ServiceRevisionID || frozen.EnvironmentID != source.EnvironmentID || frozen.ClusterID != source.ClusterID || frozen.Namespace != source.Namespace || source.EnvironmentStatus != "active" || source.BindingStatus != "validated" {
			return YAMLTarget{}, fmt.Errorf("environment target changed after submission; submit a new run")
		}
		source = frozen
	}
	// Recheck ownership here even if the SQL query is constrained. This keeps
	// future non-SQL source stores from bypassing the same isolation boundary.
	if source.ProjectID != run.ProjectID || source.ServiceID != config.ServiceID || source.ServiceRevisionID != config.ServiceRevisionID {
		return YAMLTarget{}, fmt.Errorf("yaml deployment source does not belong to the workflow project")
	}
	if source.DeliveryType != DeliveryTypeK8sYAML || source.DefinitionFormat != DeliveryTypeK8sYAML {
		return YAMLTarget{}, fmt.Errorf("service revision is not a k8s yaml definition")
	}
	if source.EnvironmentID != run.EnvironmentID || source.EnvironmentType != "k8s_namespace" || source.BindingStatus != "validated" || source.EnvironmentStatus != "active" {
		return YAMLTarget{}, fmt.Errorf("target environment is not deployable")
	}
	if source.ClusterID <= 0 || source.Namespace == "" {
		return YAMLTarget{}, fmt.Errorf("target cluster and namespace are required")
	}
	if configOnly && strings.TrimSpace(source.ImageRef) == "" && strings.TrimSpace(source.ImageName) == "" {
		// Copy even frozen sources: validation must not alter the saved run input.
		copy := *source
		copy.ImageName = "axiops-config-validation.invalid/pending-image:validation"
		source = &copy
	}
	imageRef := strings.TrimSpace(source.ImageRef)
	if imageRef == "" {
		buildID, err := matchingBuildID(run.Tasks, config.ServiceID)
		if err != nil {
			return YAMLTarget{}, err
		}
		if buildID > 0 {
			artifact, err := r.store.LoadBuildArtifact(ctx, buildID)
			if err != nil {
				return YAMLTarget{}, err
			}
			if artifact == nil || artifact.ImageURL == "" || artifact.ImageDigest == "" {
				return YAMLTarget{}, fmt.Errorf("successful build has no immutable image digest")
			}
			imageRef = artifact.ImageURL + "@" + artifact.ImageDigest
		}
	}
	definition := source.Manifest
	if len(source.EnvironmentManifest) > 0 {
		definition = append(append([]byte{}, definition...), append([]byte("\n---\n"), source.EnvironmentManifest...)...)
	}
	manifest, err := renderYAMLDefinition(definition, source, imageRef)
	if err != nil {
		return YAMLTarget{}, err
	}
	if imageRef != "" {
		rendered, err := injectYAMLImage(manifest, source.ImageName, imageRef)
		if err != nil {
			return YAMLTarget{}, err
		}
		manifest = rendered
	}
	var refs []serviceassets.SecretRef
	if source.Assets.Active() {
		resources, err := ValidateYAMLManifest(manifest, source.Namespace)
		if err != nil {
			return YAMLTarget{}, err
		}
		resources, refs, err = serviceassets.Apply(resources, source.Assets, source.ServiceID, source.Namespace)
		if err != nil {
			return YAMLTarget{}, err
		}
		var out strings.Builder
		for _, item := range resources {
			data, e := yaml.Marshal(item.Object)
			if e != nil {
				return YAMLTarget{}, e
			}
			if out.Len() > 0 {
				out.WriteString("\n---\n")
			}
			out.Write(data)
		}
		manifest = []byte(out.String())
	}
	manifest, err = preserveDeploymentImages(manifest, source.Namespace, source.PreservedImages)
	if err != nil {
		return YAMLTarget{}, err
	}
	manifest, err = stampManagedConfigMaps(manifest, source.Namespace)
	if err != nil {
		return YAMLTarget{}, err
	}
	return YAMLTarget{ClusterID: source.ClusterID, Namespace: source.Namespace, Manifest: manifest, AssetSecrets: refs}, nil
}

type SQLYAMLSourceStore struct{ db sourceQuery }

func NewSQLYAMLSourceStore(db *sql.DB) *SQLYAMLSourceStore   { return &SQLYAMLSourceStore{db: db} }
func NewSQLYAMLSourceStoreTx(tx *sql.Tx) *SQLYAMLSourceStore { return &SQLYAMLSourceStore{db: tx} }

// ValidateServiceConfig uses the same renderer as execution inside the caller's
// transaction, without contacting Kubernetes or applying any resources.
func ValidateServiceConfig(ctx context.Context, tx *sql.Tx, projectID, environmentID, serviceID, revisionID int64) error {
	resolver := NewDefaultYAMLResolver(&SQLYAMLSourceStore{db: tx})
	target, err := resolver.RenderYAMLConfig(ctx, WorkflowRun{ProjectID: projectID, EnvironmentID: environmentID}, YAMLDeployConfig{ServiceID: serviceID, ServiceRevisionID: revisionID})
	if err != nil {
		return err
	}
	_, err = ValidateYAMLManifest(target.Manifest, target.Namespace)
	return err
}

func (s *SQLYAMLSourceStore) LoadYAMLSource(ctx context.Context, environmentID, serviceID, revisionID int64) (*YAMLSource, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("yaml source database is not configured")
	}
	var source YAMLSource
	var replicas sql.NullInt64
	err := s.db.QueryRowContext(ctx, `SELECT env.project_id, svc.id, rev.id, svc.delivery_type, svc.source_type, rev.definition_format,
		env.id, env.environment_type, env.binding_status, env.status, env.cluster_id, env.namespace, COALESCE(ey.definition,rev.definition), svc.image_name, COALESCE(ov.image_ref,''), ov.replicas,
		COALESCE(project.code,''), COALESCE(svc.code,''), COALESCE(env.code,''), COALESCE(cluster.name,'')
		FROM project_environment env
		JOIN app_service svc ON svc.project_id=env.project_id AND svc.id=? AND svc.status='active'
		JOIN service_revision rev ON rev.service_id=svc.id AND rev.id=?
		LEFT JOIN environment_service_override ov ON ov.project_environment_id=env.id AND ov.service_id=svc.id
		LEFT JOIN environment_service_yaml_revision ey ON ey.id=ov.environment_yaml_revision_id AND ey.environment_id=env.id AND ey.service_id=svc.id AND ey.base_revision_id=rev.id
		JOIN project ON project.id=env.project_id
		LEFT JOIN cluster ON cluster.id=env.cluster_id
		WHERE env.id=?`, serviceID, revisionID, environmentID).Scan(
		&source.ProjectID, &source.ServiceID, &source.ServiceRevisionID, &source.DeliveryType, &source.SourceType, &source.DefinitionFormat,
		&source.EnvironmentID, &source.EnvironmentType, &source.BindingStatus, &source.EnvironmentStatus,
		&source.ClusterID, &source.Namespace, &source.Manifest, &source.ImageName, &source.ImageRef, &replicas,
		&source.ProductName, &source.ServiceName, &source.EnvironmentName, &source.ClusterName)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err == nil && replicas.Valid {
		value := int(replicas.Int64)
		source.Replicas = &value
	}
	if err == nil {
		source.Assets, err = serviceassets.Load(ctx, s.db, environmentID, serviceID)
		if err != nil {
			return nil, err
		}
		rows, queryErr := s.db.QueryContext(ctx, `SELECT manifest FROM environment_resource WHERE environment_id=? AND kind<>? AND kind NOT IN ('AxiOpsHelmValues','AxiOpsHelmSharedValues') ORDER BY id`, environmentID, serviceassets.ResourceKind)
		if queryErr != nil {
			return nil, queryErr
		}
		defer rows.Close()
		var manifests []string
		for rows.Next() {
			var manifest string
			if scanErr := rows.Scan(&manifest); scanErr != nil {
				return nil, scanErr
			}
			manifests = append(manifests, manifest)
		}
		if rowsErr := rows.Err(); rowsErr != nil {
			return nil, rowsErr
		}
		if len(manifests) > 0 {
			source.EnvironmentManifest = []byte(strings.Join(manifests, "\n---\n"))
		}
		varItems, variableErr := s.db.QueryContext(ctx, `SELECT vi.variable_key, COALESCE(vi.value_text,'')
			FROM variable_set vs JOIN variable_item vi ON vi.variable_set_id=vs.id
			WHERE vs.project_id=? AND vs.environment_id=? AND vs.scope='environment'
			AND vs.name='default' AND vs.service_id IN (0, ?) AND vi.is_secret=0
			ORDER BY vs.service_id, vi.id`, source.ProjectID, environmentID, serviceID)
		if variableErr != nil {
			return nil, variableErr
		}
		defer varItems.Close()
		source.Variables = map[string]string{}
		for varItems.Next() {
			var key, value string
			if scanErr := varItems.Scan(&key, &value); scanErr != nil {
				return nil, scanErr
			}
			source.Variables[key] = value
		}
		if rowsErr := varItems.Err(); rowsErr != nil {
			return nil, rowsErr
		}
	}
	return &source, err
}

var axiopsVariablePattern = regexp.MustCompile(`\{\{\s*\.?([A-Za-z][A-Za-z0-9_]*)\s*\}\}|\$([A-Za-z][A-Za-z0-9_-]*)\$`)

// renderYAMLDefinition converts the revision's template placeholders into a
// concrete manifest immediately before Kubernetes validation. Keeping this
// at execution time means the same revision can be reused across environments.
func renderYAMLDefinition(definition []byte, source *YAMLSource, imageRef string) ([]byte, error) {
	text := string(definition)
	values := map[string]string{}
	// The editor stores custom variables in a single metadata comment so the
	// YAML remains portable and can still be edited as plain text.
	metadataPresent := false
	for _, line := range strings.Split(text, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "# AxiOps-Variables:") {
			continue
		}
		metadataPresent = true
		payload := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "# AxiOps-Variables:"))
		var items []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		if err := json.Unmarshal([]byte(payload), &items); err != nil {
			return nil, fmt.Errorf("invalid AxiOps variable metadata: %w", err)
		}
		for _, item := range items {
			key := strings.TrimSpace(item.Key)
			if key != "" {
				values[key] = item.Value
			}
		}
	}
	// Revision metadata supplies defaults; environment/service overrides win.
	for key, value := range source.Variables {
		values[key] = value
	}
	// Keep older manually-created service revisions usable when they predate the
	// standard resource variables. Explicit values still win. Custom YAML keeps
	// strict undefined-variable validation.
	standardDefaults := map[string]string{
		"replicas": "1", "port": "8080", "cpuRequest": "100m", "memoryRequest": "128Mi",
		"cpuLimit": "500m", "memoryLimit": "512Mi", "healthCheckType": "none", "healthPath": "/health",
		"healthPort": "0", "probePeriodSeconds": "10", "probeTimeoutSeconds": "3",
		"probeFailureThreshold": "3", "startupFailureThreshold": "30",
	}
	if source.SourceType == "manual" && !metadataPresent {
		for key, value := range standardDefaults {
			if _, ok := values[key]; !ok {
				values[key] = value
			}
		}
	} else if metadataPresent {
		for _, key := range []string{"cpuRequest", "memoryRequest", "cpuLimit", "memoryLimit", "healthCheckType", "healthPath", "healthPort", "probePeriodSeconds", "probeTimeoutSeconds", "probeFailureThreshold", "startupFailureThreshold"} {
			if _, ok := values[key]; !ok {
				values[key] = standardDefaults[key]
			}
		}
	}
	if source.Replicas != nil {
		values["replicas"] = fmt.Sprintf("%d", *source.Replicas)
	}
	// Legacy manually-created K8s services used the standard starter
	// placeholders before variable metadata was persisted. Keep those revisions
	// deployable with the same defaults as the current starter template.
	if source.SourceType == "manual" && !metadataPresent {
		if _, ok := values["replicas"]; !ok {
			values["replicas"] = "1"
		}
		if _, ok := values["port"]; !ok {
			values["port"] = "8080"
		}
	}
	// Identity and build output are system-owned, never editor defaults.
	values["Product"] = source.ProductName
	values["Service"] = source.ServiceName
	values["Namespace"] = source.Namespace
	values["EnvName"] = source.EnvironmentName
	values["ClusterName"] = source.ClusterName
	values["Image"] = source.ImageName
	if imageRef != "" {
		values["Image"] = imageRef
	}
	values[source.ServiceName+"-image"] = values["Image"]
	text = regexp.MustCompile(`(?m)^[\t ]*# AxiOps-Variables:.*\r?\n?`).ReplaceAllString(text, "")
	missing := make(map[string]struct{})
	text = axiopsVariablePattern.ReplaceAllStringFunc(text, func(match string) string {
		parts := axiopsVariablePattern.FindStringSubmatch(match)
		key := parts[1]
		if key == "" {
			key = parts[2]
		}
		value, ok := values[key]
		if !ok || strings.TrimSpace(value) == "" {
			missing[key] = struct{}{}
			return match
		}
		return expandYAMLVariable(value, values, missing, 0)
	})
	if len(missing) > 0 {
		keys := make([]string, 0, len(missing))
		for key := range missing {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		if _, missingImage := missing["Image"]; missingImage {
			return nil, fmt.Errorf("yaml contains undefined variables: %s；Image 为系统镜像变量，请先构建服务或为目标环境选择镜像，无需添加同名普通变量", strings.Join(keys, ", "))
		}
		return nil, fmt.Errorf("yaml contains undefined variables: %s；请在服务定义中保存变量默认值，或在目标环境的服务变量中配置", strings.Join(keys, ", "))
	}
	renderedText, err := applyStandardResources([]byte(strings.TrimSpace(text)), values)
	if err != nil {
		return nil, err
	}
	renderedText, err = applyStandardProbes(renderedText, values)
	if err != nil {
		return nil, err
	}
	return renderedText, nil
}

// applyStandardResources fills only missing workload fields. This lets old
// service revisions receive safe defaults while preserving YAML explicitly
// written by an operator.
func applyStandardResources(manifest []byte, values map[string]string) ([]byte, error) {
	// Non-workload documents and legacy validation fixtures may not contain a
	// Kubernetes metadata block. Leave them untouched so normal YAML validation
	// and undefined-variable errors retain their original behavior.
	if !strings.Contains(string(manifest), "metadata:") {
		return manifest, nil
	}
	resources, err := ParseYAMLManifest(manifest)
	if err != nil {
		return nil, err
	}
	changed := false
	workloadKinds := map[string]bool{"Deployment": true, "StatefulSet": true, "ReplicaSet": true}
	for i := range resources {
		kind, _, _ := unstructured.NestedString(resources[i].Object, "kind")
		containerPath := []string{"spec", "template", "spec", "containers"}
		if kind == "CronJob" {
			containerPath = []string{"spec", "jobTemplate", "spec", "template", "spec", "containers"}
		}
		containers, found, _ := unstructured.NestedSlice(resources[i].Object, containerPath...)
		if !found {
			continue
		}
		if workloadKinds[kind] {
			if _, exists, _ := unstructured.NestedFieldNoCopy(resources[i].Object, "spec", "replicas"); !exists {
				if replicas, parseErr := strconv.ParseInt(strings.TrimSpace(values["replicas"]), 10, 64); parseErr == nil && replicas >= 0 {
					_ = unstructured.SetNestedField(resources[i].Object, replicas, "spec", "replicas")
					changed = true
				}
			}
		}
		for j, raw := range containers {
			container, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			if _, exists := container["resources"]; !exists {
				container["resources"] = map[string]interface{}{
					"requests": map[string]interface{}{"cpu": values["cpuRequest"], "memory": values["memoryRequest"]},
					"limits":   map[string]interface{}{"cpu": values["cpuLimit"], "memory": values["memoryLimit"]},
				}
				containers[j] = container
				changed = true
			}
		}
		if changed {
			_ = unstructured.SetNestedSlice(resources[i].Object, containers, containerPath...)
		}
	}
	if !changed {
		return manifest, nil
	}
	var out strings.Builder
	for i, item := range resources {
		data, marshalErr := yaml.Marshal(item.Object)
		if marshalErr != nil {
			return nil, marshalErr
		}
		if i > 0 {
			out.WriteString("---\n")
		}
		out.Write(data)
	}
	return []byte(out.String()), nil
}

// applyStandardProbes turns the service-level probe settings into native
// Kubernetes probe fields. The default "none" mode deliberately leaves the
// workload untouched, because a /health endpoint cannot be assumed.
func applyStandardProbes(manifest []byte, values map[string]string) ([]byte, error) {
	type probeSettings struct {
		Mode string
		Path string
		Port int64
	}
	mode := strings.ToLower(strings.TrimSpace(values["healthCheckType"]))
	if mode == "" || mode == "none" {
		return manifest, nil
	}
	if mode != "http" && mode != "tcp" {
		return nil, fmt.Errorf("healthCheckType must be none, http, or tcp")
	}
	enabled := map[string]bool{}
	for _, key := range []string{"probeReadinessEnabled", "probeStartupEnabled", "probeLivenessEnabled"} {
		// Missing flags retain the historical three-probe behavior. New UI
		// selections explicitly opt into readiness/startup and leave liveness off.
		text := strings.TrimSpace(values[key])
		enabled[key] = text != "false"
		if text != "" && text != "true" && text != "false" {
			return nil, fmt.Errorf("%s must be true or false", key)
		}
	}
	if !enabled["probeReadinessEnabled"] && !enabled["probeStartupEnabled"] && !enabled["probeLivenessEnabled"] {
		return manifest, nil
	}
	portText := strings.TrimSpace(values["healthPort"])
	port := int64(0)
	if portText != "" {
		if parsed, err := strconv.ParseInt(portText, 10, 64); err == nil {
			port = parsed
		} else {
			return nil, fmt.Errorf("healthPort must be a numeric port")
		}
	}
	if port == 0 {
		var err error
		port, err = strconv.ParseInt(strings.TrimSpace(values["port"]), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("port must be numeric when probes are enabled")
		}
	}
	if port < 1 || port > 65535 {
		return nil, fmt.Errorf("health probe port must be between 1 and 65535")
	}
	path := strings.TrimSpace(values["healthPath"])
	if mode == "http" && (path == "" || !strings.HasPrefix(path, "/")) {
		return nil, fmt.Errorf("healthPath must start with /")
	}
	settings := probeSettings{Mode: mode, Path: path, Port: port}
	period, err := probeInt(values, "probePeriodSeconds", 10, 1, 3600)
	if err != nil {
		return nil, err
	}
	timeout, err := probeInt(values, "probeTimeoutSeconds", 3, 1, 60)
	if err != nil {
		return nil, err
	}
	failureThreshold, err := probeInt(values, "probeFailureThreshold", 3, 1, 100)
	if err != nil {
		return nil, err
	}
	startupFailureThreshold, err := probeInt(values, "startupFailureThreshold", 30, 1, 1000)
	if err != nil {
		return nil, err
	}
	resources, err := ParseYAMLManifest(manifest)
	if err != nil {
		return nil, err
	}
	for i := range resources {
		paths := [][]string{{"spec", "template", "spec", "containers"}, {"spec", "jobTemplate", "spec", "template", "spec", "containers"}}
		for _, path := range paths {
			containers, found, _ := unstructured.NestedSlice(resources[i].Object, path...)
			if !found {
				continue
			}
			for j, raw := range containers {
				container, ok := raw.(map[string]interface{})
				if !ok {
					continue
				}
				probe := map[string]interface{}{}
				startupProbe := map[string]interface{}{"failureThreshold": startupFailureThreshold, "periodSeconds": period, "timeoutSeconds": timeout}
				if settings.Mode == "http" {
					probe["httpGet"] = map[string]interface{}{"path": settings.Path, "port": settings.Port}
					startupProbe["httpGet"] = probe["httpGet"]
				} else {
					probe["tcpSocket"] = map[string]interface{}{"port": settings.Port}
					startupProbe["tcpSocket"] = probe["tcpSocket"]
				}
				probe["periodSeconds"] = period
				probe["timeoutSeconds"] = timeout
				probe["failureThreshold"] = failureThreshold
				// Explicit probe fields in the YAML editor are authoritative. The
				// compact parameter panel only fills fields that are absent.
				if _, exists := container["startupProbe"]; !exists && enabled["probeStartupEnabled"] {
					container["startupProbe"] = startupProbe
				}
				if _, exists := container["readinessProbe"]; !exists && enabled["probeReadinessEnabled"] {
					container["readinessProbe"] = probe
				}
				if _, exists := container["livenessProbe"]; !exists && enabled["probeLivenessEnabled"] {
					container["livenessProbe"] = probe
				}
				containers[j] = container
			}
			_ = unstructured.SetNestedSlice(resources[i].Object, containers, path...)
		}
	}
	var out strings.Builder
	for i, item := range resources {
		data, err := yaml.Marshal(item.Object)
		if err != nil {
			return nil, err
		}
		if i > 0 {
			out.WriteString("---\n")
		}
		out.Write(data)
	}
	return []byte(out.String()), nil
}

func probeInt(values map[string]string, key string, fallback, min, max int64) (int64, error) {
	text := strings.TrimSpace(values[key])
	if text == "" {
		return fallback, nil
	}
	value, err := strconv.ParseInt(text, 10, 64)
	if err != nil || value < min || value > max {
		return 0, fmt.Errorf("%s must be between %d and %d", key, min, max)
	}
	return value, nil
}

func expandYAMLVariable(value string, values map[string]string, missing map[string]struct{}, depth int) string {
	if depth >= 8 || !axiopsVariablePattern.MatchString(value) {
		return value
	}
	return axiopsVariablePattern.ReplaceAllStringFunc(value, func(match string) string {
		parts := axiopsVariablePattern.FindStringSubmatch(match)
		key := parts[1]
		if key == "" {
			key = parts[2]
		}
		nested, ok := values[key]
		if !ok || strings.TrimSpace(nested) == "" {
			missing[key] = struct{}{}
			return match
		}
		return expandYAMLVariable(nested, values, missing, depth+1)
	})
}

func (s *SQLYAMLSourceStore) LoadBuildArtifact(ctx context.Context, buildID int64) (*BuildArtifact, error) {
	var item BuildArtifact
	err := s.db.QueryRowContext(ctx, `SELECT image_url,image_digest,project_id,service_id FROM build_run WHERE id=? AND status='success'`, buildID).Scan(&item.ImageURL, &item.ImageDigest, &item.ProjectID, &item.ServiceID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &item, err
}

func matchingBuildID(tasks []TaskRun, serviceID int64) (int64, error) {
	var found int64
	for _, task := range tasks {
		if task.NodeType != NodeBuild || task.Status != RunSucceeded {
			continue
		}
		cfg, err := buildConfig(task.NodeSnapshot)
		if err != nil {
			return 0, err
		}
		if cfg.ServiceID != serviceID {
			continue
		}
		id, err := parseBuildRunID(task.ProviderRunID)
		if err != nil {
			return 0, err
		}
		if found > 0 && found != id {
			return 0, fmt.Errorf("multiple build outputs match service %d", serviceID)
		}
		found = id
	}
	return found, nil
}

func injectYAMLImage(manifest []byte, imageName, imageRef string) ([]byte, error) {
	resources, err := ParseYAMLManifest(manifest)
	if err != nil {
		return nil, err
	}
	target := normalizeImageName(imageName)
	if target == "" {
		return nil, fmt.Errorf("service image_name is required for image injection")
	}
	matches := 0
	for i := range resources {
		paths := [][]string{{"spec", "template", "spec", "containers"}, {"spec", "template", "spec", "initContainers"}, {"spec", "jobTemplate", "spec", "template", "spec", "containers"}, {"spec", "jobTemplate", "spec", "template", "spec", "initContainers"}}
		for _, path := range paths {
			containers, found, _ := unstructured.NestedSlice(resources[i].Object, path...)
			if !found {
				continue
			}
			changed := false
			for j, raw := range containers {
				container, ok := raw.(map[string]interface{})
				if !ok {
					continue
				}
				current, _ := container["image"].(string)
				// {{.Image}} already expands to the immutable build output. Older
				// services may still store a short image_name such as risk_server.
				if current != imageRef && normalizeImageName(current) != target {
					continue
				}
				container["image"] = imageRef
				containers[j] = container
				matches++
				changed = true
			}
			if changed {
				_ = unstructured.SetNestedSlice(resources[i].Object, containers, path...)
			}
		}
	}
	if matches == 0 {
		return nil, fmt.Errorf("no workload image matches service image_name %q", imageName)
	}
	var out strings.Builder
	for i, item := range resources {
		data, err := yaml.Marshal(item.Object)
		if err != nil {
			return nil, err
		}
		if i > 0 {
			out.WriteString("---\n")
		}
		out.Write(data)
	}
	return []byte(out.String()), nil
}
func normalizeImageName(value string) string {
	value = strings.TrimSpace(value)
	if at := strings.Index(value, "@"); at >= 0 {
		value = value[:at]
	}
	lastSlash := strings.LastIndex(value, "/")
	if colon := strings.LastIndex(value, ":"); colon > lastSlash {
		value = value[:colon]
	}
	return value
}
