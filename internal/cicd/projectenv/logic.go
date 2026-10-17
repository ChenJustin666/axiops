package projectenv

import (
	"context"
	"database/sql"
	"strings"

	"github.com/ChenJustin666/axiops/internal/cicd/codesource"
	"github.com/ChenJustin666/axiops/internal/cicd/workflow"
	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"sigs.k8s.io/yaml"
)

type Logic struct {
	repo        environmentRepository
	authorizer  authz.Checker
	codeSources valuesCodeSource
}

type valuesCodeSource interface {
	Get(context.Context, int64, int64) (*codesource.CodeSource, error)
	ReadFiles(context.Context, int64, int64, string, string, []string) (*codesource.RepositorySnapshot, error)
}

func (l *Logic) WithCodeSources(s valuesCodeSource) *Logic { l.codeSources = s; return l }

func (l *Logic) WithDecryptor(decryptor workflow.CredentialDecryptor) *Logic {
	if repo, ok := l.repo.(*Repository); ok {
		repo.decryptor = decryptor
	}
	return l
}

type environmentRepository interface {
	FindProjectDeliveryProfile(context.Context, int64) (*ProjectDeliveryProfile, error)
	FindCluster(context.Context, int64) (*ClusterTarget, error)
	FindByID(context.Context, int64) (*ProjectEnvironment, error)
	List(context.Context, authz.ScopeSet, int64) ([]ProjectEnvironment, error)
	Create(context.Context, *ProjectEnvironment) (int64, error)
	FindVariableSetByEnvironment(context.Context, int64) (*VariableSet, error)
	FindVariableSetByID(context.Context, int64) (*VariableSet, error)
	CredentialBelongsToProject(context.Context, int64, int64) (bool, error)
	ReplaceVariableItems(context.Context, int64, int64, int64, []VariableItemInput) (bool, error)
	FindService(context.Context, int64) (*ServiceTarget, error)
	ServiceRevisionExists(context.Context, int64, int64) (bool, error)
	FindServiceOverride(context.Context, int64, int64) (*EnvironmentServiceOverride, error)
	SaveServiceOverride(context.Context, *EnvironmentServiceOverride, int64) (bool, error)
	Clone(context.Context, *ProjectEnvironment, *ProjectEnvironment, int64) (int64, error)
	ListResources(context.Context, int64) ([]EnvironmentResource, error)
	FindResource(context.Context, int64, int64) (*EnvironmentResource, error)
	CreateResource(context.Context, *EnvironmentResource) (int64, error)
	UpdateResource(context.Context, *EnvironmentResource, int64) (bool, error)
	DeleteResource(context.Context, int64, int64) (bool, error)
}

var environmentResourceKinds = map[string]struct{}{
	"Ingress": {}, "ConfigMap": {}, "Secret": {}, "PersistentVolumeClaim": {},
}

func (l *Logic) ListResources(ctx context.Context, userID, environmentID int64) ([]EnvironmentResource, int) {
	environment, code := l.Get(ctx, userID, environmentID)
	if code != errcode.CodeSuccess || environment == nil {
		return nil, code
	}
	items, err := l.repo.ListResources(ctx, environment.ID)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	return items, errcode.CodeSuccess
}

func (l *Logic) SaveResource(ctx context.Context, userID, environmentID int64, req *SaveEnvironmentResourceRequest) (*EnvironmentResource, int) {
	if req == nil {
		return nil, errcode.CodeParamInvalid
	}
	req.Name, req.Kind, req.Manifest = strings.TrimSpace(req.Name), strings.TrimSpace(req.Kind), strings.TrimSpace(req.Manifest)
	if req.Name == "" || len(req.Name) > 128 || len(req.Manifest) == 0 || len(req.Manifest) > 1024*1024 {
		return nil, errcode.CodeParamInvalid
	}
	if _, ok := environmentResourceKinds[req.Kind]; !ok {
		return nil, errcode.CodeParamInvalid
	}
	var document struct {
		APIVersion string `yaml:"apiVersion"`
		Kind       string `yaml:"kind"`
		Metadata   struct {
			Name string `yaml:"name"`
		} `yaml:"metadata"`
	}
	if err := yaml.Unmarshal([]byte(req.Manifest), &document); err != nil || document.APIVersion == "" || document.Kind != req.Kind || strings.TrimSpace(document.Metadata.Name) == "" {
		return nil, errcode.CodeParamInvalid
	}
	environment, code := l.Get(ctx, userID, environmentID)
	if code != errcode.CodeSuccess || environment == nil {
		return nil, code
	}
	if code := l.authorizeEnvironment(ctx, userID, environment, "update"); code != errcode.CodeSuccess {
		return nil, code
	}
	resource := &EnvironmentResource{ID: req.ID, EnvironmentID: environmentID, Name: req.Name, Kind: req.Kind, Manifest: req.Manifest, Revision: req.ExpectedRevision, CreatedBy: userID}
	if req.ID == 0 {
		id, err := l.repo.CreateResource(ctx, resource)
		if err != nil {
			return nil, errcode.CodeDBError
		}
		resource.ID, resource.Revision = id, 1
		return resource, errcode.CodeSuccess
	}
	current, err := l.repo.FindResource(ctx, environmentID, req.ID)
	if err != nil || current == nil || current.Revision != req.ExpectedRevision {
		return nil, errcode.CodeParamInvalid
	}
	if ok, err := l.repo.UpdateResource(ctx, resource, req.ExpectedRevision); err != nil || !ok {
		return nil, errcode.CodeDBError
	}
	resource.Revision++
	return resource, errcode.CodeSuccess
}

func (l *Logic) DeleteResource(ctx context.Context, userID, environmentID, resourceID int64) int {
	environment, code := l.Get(ctx, userID, environmentID)
	if code != errcode.CodeSuccess || environment == nil {
		return code
	}
	if code := l.authorizeEnvironment(ctx, userID, environment, "update"); code != errcode.CodeSuccess {
		return code
	}
	ok, err := l.repo.DeleteResource(ctx, environmentID, resourceID)
	if err != nil {
		return errcode.CodeDBError
	}
	if !ok {
		return errcode.CodeParamInvalid
	}
	return errcode.CodeSuccess
}

// Clone creates an environment from another non-production environment.
// Secret values are intentionally omitted by the repository layer.
func (l *Logic) Clone(ctx context.Context, userID, sourceID int64, req *CloneEnvironmentRequest) (int64, int) {
	if req == nil {
		return 0, errcode.CodeParamInvalid
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Code = strings.TrimSpace(req.Code)
	req.Namespace = strings.TrimSpace(req.Namespace)
	req.HostGroupRef = strings.TrimSpace(req.HostGroupRef)
	req.IsProduction = req.IsProduction || IsProductionCode(req.Code)
	if !ValidateEnvironmentIdentity(req.Name, req.Code) {
		return 0, errcode.CodeParamInvalid
	}
	source, err := l.repo.FindByID(ctx, sourceID)
	if err != nil {
		return 0, errcode.CodeDBError
	}
	if source == nil {
		return 0, errcode.CodeEnvironmentNotFound
	}
	if source.IsProduction || source.Status != "active" {
		return 0, errcode.CodeParamInvalid
	}
	if code := l.authorizeEnvironment(ctx, userID, source, "view"); code != errcode.CodeSuccess {
		return 0, code
	}
	profile, err := l.repo.FindProjectDeliveryProfile(ctx, source.ProjectID)
	if err != nil {
		return 0, errcode.CodeDBError
	}
	if profile == nil || profile.DeliveryStatus != "active" {
		return 0, errcode.CodeParamInvalid
	}
	allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{
		Resource: "environment", Action: "create", ProjectID: source.ProjectID, Environment: req.Code,
	})
	if err != nil {
		return 0, errcode.CodeDBError
	}
	if !allowed {
		return 0, errcode.CodeNoPermission
	}
	if req.IsProduction {
		allowed, err = l.authorizer.Authorize(ctx, userID, authz.Request{
			Resource: "environment", Action: "manage_production", ProjectID: source.ProjectID, Environment: req.Code,
		})
		if err != nil {
			return 0, errcode.CodeDBError
		}
		if !allowed {
			return 0, errcode.CodeNoPermission
		}
	}
	if req.ClusterID == source.ClusterID && req.Namespace == source.Namespace && req.HostGroupRef == source.HostGroupRef {
		// Equal bindings are valid; this branch documents that no special case
		// bypasses the normal target authorization below.
	}
	environmentType, bindingStatus, valid := ResolveEnvironmentTarget(profile.DeliveryType, CreateEnvironmentRequest{
		ClusterID: req.ClusterID, Namespace: req.Namespace, HostGroupRef: req.HostGroupRef,
	})
	if !valid {
		return 0, errcode.CodeParamInvalid
	}
	if environmentType == EnvironmentTypeK8sNamespace {
		cluster, err := l.repo.FindCluster(ctx, req.ClusterID)
		if err != nil {
			return 0, errcode.CodeDBError
		}
		if cluster == nil || cluster.Status != "connected" {
			return 0, errcode.CodeClusterNotFound
		}
		allowed, err = l.authorizer.Authorize(ctx, userID, authz.Request{
			Resource: "cluster", Action: "view", ProjectID: source.ProjectID, ClusterID: req.ClusterID, Environment: req.Code,
		})
		if err != nil {
			return 0, errcode.CodeDBError
		}
		if !allowed {
			return 0, errcode.CodeNoPermission
		}
	}
	target := &ProjectEnvironment{
		ProjectID: source.ProjectID, Name: req.Name, Code: req.Code,
		EnvironmentType: environmentType, ClusterID: req.ClusterID, Namespace: req.Namespace,
		HostGroupRef: req.HostGroupRef, BindingStatus: bindingStatus, IsProduction: req.IsProduction,
		DeployApproval: req.DeployApproval || req.IsProduction, Status: "active", CreatedBy: userID,
	}
	id, err := l.repo.Clone(ctx, source, target, userID)
	if err != nil {
		return 0, errcode.CodeDBError
	}
	return id, errcode.CodeSuccess
}

func NewLogic(db *sql.DB, authorizer authz.Checker) *Logic {
	return &Logic{repo: NewRepository(db), authorizer: authorizer}
}

func (l *Logic) List(ctx context.Context, userID, projectID int64) ([]ProjectEnvironment, int) {
	scope, err := l.authorizer.Scopes(ctx, userID, "environment", "view")
	if err != nil {
		return nil, errcode.CodeDBError
	}
	environments, err := l.repo.List(ctx, scope, projectID)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	return environments, errcode.CodeSuccess
}

func (l *Logic) Get(ctx context.Context, userID, id int64) (*ProjectEnvironment, int) {
	environment, err := l.repo.FindByID(ctx, id)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if environment == nil {
		return nil, errcode.CodeEnvironmentNotFound
	}
	if code := l.authorizeEnvironment(ctx, userID, environment, "view"); code != errcode.CodeSuccess {
		return nil, code
	}
	return environment, errcode.CodeSuccess
}

// ResolveK8sTarget authorizes the environment "view" permission and returns
// the bound cluster id + namespace so the read-only environment workload view
// queries the exact cluster the environment deploys to. Only k8s_namespace
// environments are resolvable.
func (l *Logic) ResolveK8sTarget(ctx context.Context, userID, environmentID int64) (int64, string, int) {
	environment, code := l.Get(ctx, userID, environmentID)
	if code != errcode.CodeSuccess {
		return 0, "", code
	}
	if environment.EnvironmentType != EnvironmentTypeK8sNamespace {
		return 0, "", errcode.CodeParamInvalid
	}
	if environment.ClusterID <= 0 || strings.TrimSpace(environment.Namespace) == "" {
		return 0, "", errcode.CodeParamInvalid
	}
	return environment.ClusterID, environment.Namespace, errcode.CodeSuccess
}

func (l *Logic) Create(ctx context.Context, userID, projectID int64, req *CreateEnvironmentRequest) (int64, int) {
	if req == nil {
		return 0, errcode.CodeParamInvalid
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Code = strings.TrimSpace(req.Code)
	req.Namespace = strings.TrimSpace(req.Namespace)
	req.HostGroupRef = strings.TrimSpace(req.HostGroupRef)
	req.IsProduction = req.IsProduction || IsProductionCode(req.Code)
	if !ValidateEnvironmentIdentity(req.Name, req.Code) {
		return 0, errcode.CodeParamInvalid
	}
	allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{
		Resource: "environment", Action: "create", ProjectID: projectID, Environment: req.Code,
	})
	if err != nil {
		return 0, errcode.CodeDBError
	}
	if !allowed {
		return 0, errcode.CodeNoPermission
	}
	if req.IsProduction {
		allowed, err = l.authorizer.Authorize(ctx, userID, authz.Request{
			Resource: "environment", Action: "manage_production", ProjectID: projectID, Environment: req.Code,
		})
		if err != nil {
			return 0, errcode.CodeDBError
		}
		if !allowed {
			return 0, errcode.CodeNoPermission
		}
	}

	profile, err := l.repo.FindProjectDeliveryProfile(ctx, projectID)
	if err != nil {
		return 0, errcode.CodeDBError
	}
	if profile == nil {
		return 0, errcode.CodeProjectNotFound
	}
	if profile.DeliveryStatus != "active" {
		return 0, errcode.CodeParamInvalid
	}
	environmentType, bindingStatus, valid := ResolveEnvironmentTarget(profile.DeliveryType, *req)
	if !valid {
		return 0, errcode.CodeParamInvalid
	}
	if environmentType == EnvironmentTypeK8sNamespace {
		cluster, err := l.repo.FindCluster(ctx, req.ClusterID)
		if err != nil {
			return 0, errcode.CodeDBError
		}
		if cluster == nil || cluster.Status != "connected" {
			return 0, errcode.CodeClusterNotFound
		}
		allowed, err = l.authorizer.Authorize(ctx, userID, authz.Request{
			Resource: "cluster", Action: "view", ProjectID: projectID, ClusterID: req.ClusterID, Environment: req.Code,
		})
		if err != nil {
			return 0, errcode.CodeDBError
		}
		if !allowed {
			return 0, errcode.CodeNoPermission
		}
	}

	environment := &ProjectEnvironment{
		ProjectID:       projectID,
		Name:            req.Name,
		Code:            req.Code,
		EnvironmentType: environmentType,
		ClusterID:       req.ClusterID,
		Namespace:       req.Namespace,
		HostGroupRef:    req.HostGroupRef,
		BindingStatus:   bindingStatus,
		IsProduction:    req.IsProduction,
		DeployApproval:  req.DeployApproval || req.IsProduction,
		Status:          "active",
		CreatedBy:       userID,
	}
	if len(req.HelmServices) > 50 || len(req.HelmServices) > 0 && profile.DeliveryType != DeliveryTypeHelm {
		return 0, errcode.CodeParamInvalid
	}
	seen := map[int64]bool{}
	for _, s := range req.HelmServices {
		if s.ServiceID <= 0 || s.RevisionID <= 0 || seen[s.ServiceID] || s.Helm.Validate() != nil || len(s.Helm.CredentialKeys) > 0 {
			return 0, errcode.CodeParamInvalid
		}
		seen[s.ServiceID] = true
		svc, e := l.repo.FindService(ctx, s.ServiceID)
		if e != nil || svc == nil || svc.ProjectID != projectID || svc.DeliveryType != DeliveryTypeHelm {
			return 0, errcode.CodeParamInvalid
		}
		allowed, e = l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "service", Action: "view", ProjectID: projectID, ServiceID: s.ServiceID})
		if e != nil || !allowed {
			return 0, errcode.CodeNoPermission
		}
		// Creating environment service settings is governed by environment:create;
		// shared Chart contents are only read, never modified here.
		if s.Helm.Source != nil {
			actual, e := l.readValuesSource(ctx, userID, projectID, s.Helm.Source)
			if e != nil || actual.CommitID != s.Helm.Source.CommitID || actual.Base != s.Helm.Source.Base {
				req.failure = e
				return 0, errcode.CodeParamInvalid
			}
		}
	}
	environment.initialHelmServices, environment.preview = req.HelmServices, req.preview
	id, err := l.repo.Create(ctx, environment)
	if err != nil {
		if len(req.HelmServices) > 0 {
			req.failure = err
			return 0, errcode.CodeParamInvalid
		}
		return 0, errcode.CodeDBError
	}
	req.configs = environment.initialConfigs
	return id, errcode.CodeSuccess
}

func (l *Logic) GetVariableSet(ctx context.Context, userID, environmentID int64) (*VariableSet, int) {
	environment, err := l.repo.FindByID(ctx, environmentID)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if environment == nil {
		return nil, errcode.CodeEnvironmentNotFound
	}
	if code := l.authorizeEnvironment(ctx, userID, environment, "view"); code != errcode.CodeSuccess {
		return nil, code
	}
	set, err := l.repo.FindVariableSetByEnvironment(ctx, environmentID)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if set == nil {
		return nil, errcode.CodeNotFound
	}
	return set, errcode.CodeSuccess
}

func (l *Logic) GetServiceVariableSet(ctx context.Context, userID, environmentID, serviceID int64) (*VariableSet, int) {
	environment, err := l.repo.FindByID(ctx, environmentID)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if environment == nil {
		return nil, errcode.CodeEnvironmentNotFound
	}
	if code := l.authorizeEnvironment(ctx, userID, environment, "view"); code != errcode.CodeSuccess {
		return nil, code
	}
	serviceRepo, ok := l.repo.(interface {
		FindVariableSetByEnvironmentService(context.Context, int64, int64) (*VariableSet, error)
	})
	if !ok {
		return nil, errcode.CodeNotFound
	}
	set, err := serviceRepo.FindVariableSetByEnvironmentService(ctx, environmentID, serviceID)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if set == nil {
		ensurer, ok := l.repo.(interface {
			EnsureServiceVariableSet(context.Context, int64, int64, int64) error
		})
		if !ok {
			return nil, errcode.CodeNotFound
		}
		if err := ensurer.EnsureServiceVariableSet(ctx, environment.ProjectID, environmentID, serviceID); err != nil {
			return nil, errcode.CodeDBError
		}
		set, err = serviceRepo.FindVariableSetByEnvironmentService(ctx, environmentID, serviceID)
		if err != nil {
			return nil, errcode.CodeDBError
		}
	}
	if set == nil {
		return nil, errcode.CodeNotFound
	}
	return set, errcode.CodeSuccess
}

func (l *Logic) UpdateVariableSet(ctx context.Context, userID, setID int64, req *UpdateVariableSetRequest) (*VariableSet, int) {
	if req.Revision <= 0 || !ValidateVariableItems(req.Items) {
		return nil, errcode.CodeParamInvalid
	}
	set, err := l.repo.FindVariableSetByID(ctx, setID)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if set == nil || set.Scope != "environment" {
		return nil, errcode.CodeNotFound
	}
	environment, err := l.repo.FindByID(ctx, set.EnvironmentID)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if environment == nil {
		return nil, errcode.CodeEnvironmentNotFound
	}
	if code := l.authorizeEnvironment(ctx, userID, environment, "update"); code != errcode.CodeSuccess {
		return nil, code
	}
	for _, item := range req.Items {
		if !item.IsSecret {
			continue
		}
		belongs, err := l.repo.CredentialBelongsToProject(ctx, item.CredentialID, set.ProjectID)
		if err != nil {
			return nil, errcode.CodeDBError
		}
		if !belongs {
			return nil, errcode.CodeCredentialNotFound
		}
		allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{
			Resource: "credential", Action: "view", ProjectID: set.ProjectID, EnvironmentID: environment.ID, Environment: environment.Code,
		})
		if err != nil {
			return nil, errcode.CodeDBError
		}
		if !allowed {
			return nil, errcode.CodeNoPermission
		}
	}
	updated, err := l.repo.ReplaceVariableItems(ctx, setID, req.Revision, userID, req.Items)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if !updated {
		return nil, errcode.CodeParamInvalid
	}
	if set.ServiceID > 0 {
		return l.GetServiceVariableSet(ctx, userID, environment.ID, set.ServiceID)
	}
	return l.GetVariableSet(ctx, userID, environment.ID)
}

func (l *Logic) authorizeEnvironment(ctx context.Context, userID int64, environment *ProjectEnvironment, action string) int {
	allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{
		Resource: "environment", Action: action, ProjectID: environment.ProjectID,
		EnvironmentID: environment.ID, ClusterID: environment.ClusterID, Environment: environment.Code,
	})
	if err != nil {
		return errcode.CodeDBError
	}
	if !allowed {
		return errcode.CodeNoPermission
	}
	return errcode.CodeSuccess
}

func (l *Logic) GetServiceOverride(ctx context.Context, userID, environmentID, serviceID int64) (*EnvironmentServiceOverride, int) {
	_, _, code := l.authorizeEnvironmentService(ctx, userID, environmentID, serviceID, "view")
	if code != errcode.CodeSuccess {
		return nil, code
	}
	override, err := l.repo.FindServiceOverride(ctx, environmentID, serviceID)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if override == nil {
		return nil, errcode.CodeNotFound
	}
	return override, errcode.CodeSuccess
}

func (l *Logic) SaveServiceOverride(ctx context.Context, userID, environmentID, serviceID int64, req *SaveServiceOverrideRequest) (*EnvironmentServiceOverride, int) {
	if !ValidateServiceOverrideRequest(*req) {
		return nil, errcode.CodeParamInvalid
	}
	environment, service, code := l.authorizeEnvironmentService(ctx, userID, environmentID, serviceID, "update")
	if code != errcode.CodeSuccess {
		return nil, code
	}
	if !IsEnvironmentCompatible(environment.EnvironmentType, environment.BindingStatus, service.DeliveryType) {
		return nil, errcode.CodeParamInvalid
	}
	exists, err := l.repo.ServiceRevisionExists(ctx, serviceID, req.ServiceRevisionID)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if !exists {
		return nil, errcode.CodeNotFound
	}
	override := &EnvironmentServiceOverride{
		ProjectEnvironmentID: environmentID,
		ServiceID:            serviceID,
		ServiceRevisionID:    req.ServiceRevisionID,
		ImageRef:             strings.TrimSpace(req.ImageRef),
		Replicas:             req.Replicas,
		UpdatedBy:            userID,
	}
	saved, err := l.repo.SaveServiceOverride(ctx, override, req.ExpectedRevision)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if !saved {
		return nil, errcode.CodeParamInvalid
	}
	return l.GetServiceOverride(ctx, userID, environmentID, serviceID)
}

func (l *Logic) authorizeEnvironmentService(ctx context.Context, userID, environmentID, serviceID int64, environmentAction string) (*ProjectEnvironment, *ServiceTarget, int) {
	environment, err := l.repo.FindByID(ctx, environmentID)
	if err != nil {
		return nil, nil, errcode.CodeDBError
	}
	if environment == nil {
		return nil, nil, errcode.CodeEnvironmentNotFound
	}
	if code := l.authorizeEnvironment(ctx, userID, environment, environmentAction); code != errcode.CodeSuccess {
		return nil, nil, code
	}
	service, err := l.repo.FindService(ctx, serviceID)
	if err != nil {
		return nil, nil, errcode.CodeDBError
	}
	if service == nil || service.ProjectID != environment.ProjectID {
		return nil, nil, errcode.CodeServiceNotFound
	}
	allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{
		Resource: "service", Action: "view", ProjectID: service.ProjectID, ServiceID: service.ID,
		EnvironmentID: environment.ID, Environment: environment.Code, ClusterID: environment.ClusterID,
	})
	if err != nil {
		return nil, nil, errcode.CodeDBError
	}
	if !allowed {
		return nil, nil, errcode.CodeNoPermission
	}
	return environment, service, errcode.CodeSuccess
}
