package projectenv

import (
	"context"
	"fmt"
	"testing"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
)

type policyStore []authz.Policy

func (p policyStore) ListUserPolicies(context.Context, int64) ([]authz.Policy, error) { return p, nil }

func TestEnvironmentResourceWritesHonorEnvironmentAndClusterDenies(t *testing.T) {
	for _, scope := range []string{authz.ScopeEnvironment, authz.ScopeCluster} {
		for _, denied := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/denied=%v", scope, denied), func(t *testing.T) {
				env := &ProjectEnvironment{ID: 22, ProjectID: 10, ClusterID: 2, Code: "qa"}
				id := env.ID
				if scope == authz.ScopeCluster {
					id = env.ClusterID
				}
				policies := policyStore{{Resource: "environment", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10}, {Resource: "environment", Action: "update", Effect: "allow", ScopeType: scope, ScopeID: id, Environment: "qa"}}
				if denied {
					policies = append(policies, authz.Policy{Resource: "environment", Action: "update", Effect: "deny", ScopeType: scope, ScopeID: id, Environment: "qa"})
				}
				repo := &fakeRepository{environment: env}
				l := &Logic{repo: repo, authorizer: authz.New(policies)}
				_, code := l.SaveResource(context.Background(), 7, 22, &SaveEnvironmentResourceRequest{Name: "app", Kind: "ConfigMap", Manifest: "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: app\ndata: {}"})
				want := errcode.CodeSuccess
				if denied {
					want = errcode.CodeNoPermission
				}
				if code != want || l.DeleteResource(context.Background(), 7, 22, 1) != want {
					t.Fatalf("write scope ignored: save=%d want=%d", code, want)
				}
				if denied && len(repo.resources) != 0 {
					t.Fatal("denied write persisted")
				}
			})
		}
	}
}

type fakeRepository struct {
	profile               *ProjectDeliveryProfile
	cluster               *ClusterTarget
	environment           *ProjectEnvironment
	variableSet           *VariableSet
	credentialBelongs     bool
	created               *ProjectEnvironment
	replaced              bool
	service               *ServiceTarget
	serviceRevisionExists bool
	serviceOverride       *EnvironmentServiceOverride
	overrideSaved         bool
	cloned                *ProjectEnvironment
	resources             []EnvironmentResource
}

func (f *fakeRepository) ListResources(context.Context, int64) ([]EnvironmentResource, error) {
	return f.resources, nil
}
func (f *fakeRepository) FindResource(_ context.Context, _, id int64) (*EnvironmentResource, error) {
	for i := range f.resources {
		if f.resources[i].ID == id {
			item := f.resources[i]
			return &item, nil
		}
	}
	return nil, nil
}
func (f *fakeRepository) CreateResource(_ context.Context, item *EnvironmentResource) (int64, error) {
	item.ID = 1
	f.resources = append(f.resources, *item)
	return 1, nil
}
func (f *fakeRepository) UpdateResource(_ context.Context, item *EnvironmentResource, _ int64) (bool, error) {
	return true, nil
}
func (f *fakeRepository) DeleteResource(_ context.Context, _, _ int64) (bool, error) {
	return true, nil
}

func (f *fakeRepository) FindProjectDeliveryProfile(context.Context, int64) (*ProjectDeliveryProfile, error) {
	return f.profile, nil
}
func (f *fakeRepository) FindCluster(context.Context, int64) (*ClusterTarget, error) {
	return f.cluster, nil
}
func (f *fakeRepository) FindByID(context.Context, int64) (*ProjectEnvironment, error) {
	return f.environment, nil
}
func (f *fakeRepository) List(context.Context, authz.ScopeSet, int64) ([]ProjectEnvironment, error) {
	return nil, nil
}
func (f *fakeRepository) Create(_ context.Context, environment *ProjectEnvironment) (int64, error) {
	f.created = environment
	return 42, nil
}
func (f *fakeRepository) FindVariableSetByEnvironment(context.Context, int64) (*VariableSet, error) {
	return f.variableSet, nil
}
func (f *fakeRepository) FindVariableSetByID(context.Context, int64) (*VariableSet, error) {
	return f.variableSet, nil
}
func (f *fakeRepository) CredentialBelongsToProject(context.Context, int64, int64) (bool, error) {
	return f.credentialBelongs, nil
}
func (f *fakeRepository) ReplaceVariableItems(context.Context, int64, int64, int64, []VariableItemInput) (bool, error) {
	f.replaced = true
	return true, nil
}
func (f *fakeRepository) FindService(context.Context, int64) (*ServiceTarget, error) {
	return f.service, nil
}
func (f *fakeRepository) ServiceRevisionExists(context.Context, int64, int64) (bool, error) {
	return f.serviceRevisionExists, nil
}
func (f *fakeRepository) FindServiceOverride(context.Context, int64, int64) (*EnvironmentServiceOverride, error) {
	return f.serviceOverride, nil
}
func (f *fakeRepository) SaveServiceOverride(_ context.Context, override *EnvironmentServiceOverride, expectedRevision int64) (bool, error) {
	if !f.overrideSaved {
		return false, nil
	}
	copy := *override
	copy.ID = 8
	copy.Revision = expectedRevision + 1
	f.serviceOverride = &copy
	return true, nil
}
func (f *fakeRepository) Clone(_ context.Context, _ *ProjectEnvironment, target *ProjectEnvironment, _ int64) (int64, error) {
	f.cloned = target
	return 84, nil
}

func TestCreateProductionEnvironmentRequiresDedicatedPermission(t *testing.T) {
	repo := &fakeRepository{}
	logic := &Logic{repo: repo, authorizer: authz.New(policyStore{{
		Resource: "environment", Action: "create", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10, Environment: "*",
	}})}
	_, code := logic.Create(context.Background(), 7, 10, &CreateEnvironmentRequest{
		Name: "Production", Code: "prod", ClusterID: 2, Namespace: "payment-prod", IsProduction: true,
	})
	if code != errcode.CodeNoPermission {
		t.Fatalf("code=%d want no permission", code)
	}
	if repo.created != nil {
		t.Fatal("production environment was persisted without dedicated permission")
	}
}

func TestStandardProductionCodesCannotBypassClassificationOrPermission(t *testing.T) {
	for _, code := range []string{"prod", "production"} {
		for _, permitted := range []bool{false, true} {
			t.Run(code+fmt.Sprint(permitted), func(t *testing.T) {
				repo := &fakeRepository{
					profile:     &ProjectDeliveryProfile{DeliveryType: DeliveryTypeK8sYAML, DeliveryStatus: "active"},
					cluster:     &ClusterTarget{ID: 2, Status: "connected"},
					environment: &ProjectEnvironment{ID: 3, ProjectID: 10, Code: "qa", Status: "active"},
				}
				policies := policyStore{
					{Resource: "environment", Action: "create", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10, Environment: "*"},
					{Resource: "environment", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10, Environment: "*"},
					{Resource: "cluster", Action: "view", Effect: "allow", ScopeType: authz.ScopeCluster, ScopeID: 2, Environment: "*"},
				}
				if permitted {
					policies = append(policies, authz.Policy{Resource: "environment", Action: "manage_production", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10, Environment: "*"})
				}
				logic := &Logic{repo: repo, authorizer: authz.New(policies)}
				_, result := logic.Create(context.Background(), 7, 10, &CreateEnvironmentRequest{Name: "Production", Code: " " + code + " ", ClusterID: 2, Namespace: "app-prod"})
				if !permitted {
					if result != errcode.CodeNoPermission || repo.created != nil {
						t.Fatal("production creation bypassed permission")
					}
				} else if result != errcode.CodeSuccess || repo.created == nil || !repo.created.IsProduction || !repo.created.DeployApproval {
					t.Fatalf("production classification lost: code=%d env=%+v", result, repo.created)
				}
				_, result = logic.Clone(context.Background(), 7, 3, &CloneEnvironmentRequest{Name: "Production", Code: code, ClusterID: 2, Namespace: "app-prod"})
				if !permitted {
					if result != errcode.CodeNoPermission || repo.cloned != nil {
						t.Fatal("production clone bypassed permission")
					}
				} else if result != errcode.CodeSuccess || repo.cloned == nil || !repo.cloned.IsProduction || !repo.cloned.DeployApproval {
					t.Fatalf("clone classification lost: code=%d env=%+v", result, repo.cloned)
				}
			})
		}
	}
}

func TestProductionCodeDoesNotGuessFromSubstrings(t *testing.T) {
	for _, code := range []string{"qa", "preprod", "production-test", "prod-copy", "staging"} {
		if IsProductionCode(code) {
			t.Fatalf("custom code %s was classified as production", code)
		}
	}
}

func TestCreateK8sEnvironmentRequiresClusterPermission(t *testing.T) {
	repo := &fakeRepository{
		profile: &ProjectDeliveryProfile{DeliveryType: DeliveryTypeK8sYAML, DeliveryStatus: "active"},
		cluster: &ClusterTarget{ID: 2, Status: "connected"},
	}
	logic := &Logic{repo: repo, authorizer: authz.New(policyStore{{
		Resource: "environment", Action: "create", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10, Environment: "*",
	}})}
	_, code := logic.Create(context.Background(), 7, 10, &CreateEnvironmentRequest{
		Name: "Test", Code: "test", ClusterID: 2, Namespace: "payment-test",
	})
	if code != errcode.CodeNoPermission {
		t.Fatalf("code=%d want no permission", code)
	}
}

func TestCreateHostEnvironmentStaysPendingValidation(t *testing.T) {
	repo := &fakeRepository{profile: &ProjectDeliveryProfile{DeliveryType: DeliveryTypeHost, DeliveryStatus: "active"}}
	logic := &Logic{repo: repo, authorizer: authz.New(policyStore{{
		Resource: "environment", Action: "create", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10, Environment: "*",
	}})}
	id, code := logic.Create(context.Background(), 7, 10, &CreateEnvironmentRequest{
		Name: "Host Test", Code: "test", HostGroupRef: "legacy-binding-1",
	})
	if code != errcode.CodeSuccess || id != 42 {
		t.Fatalf("id=%d code=%d", id, code)
	}
	if repo.created == nil || repo.created.BindingStatus != BindingStatusPendingValidation {
		t.Fatalf("unexpected persisted environment: %+v", repo.created)
	}
}

func TestSecretVariableRequiresCredentialPermission(t *testing.T) {
	repo := &fakeRepository{
		environment:       &ProjectEnvironment{ID: 3, ProjectID: 10, Code: "test"},
		variableSet:       &VariableSet{ID: 5, ProjectID: 10, EnvironmentID: 3, Scope: "environment", Revision: 1},
		credentialBelongs: true,
	}
	logic := &Logic{repo: repo, authorizer: authz.New(policyStore{{
		Resource: "environment", Action: "update", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10, Environment: "*",
	}})}
	_, code := logic.UpdateVariableSet(context.Background(), 7, 5, &UpdateVariableSetRequest{
		Revision: 1,
		Items:    []VariableItemInput{{VariableKey: "TOKEN", IsSecret: true, CredentialID: 9}},
	})
	if code != errcode.CodeNoPermission {
		t.Fatalf("code=%d want no permission", code)
	}
	if repo.replaced {
		t.Fatal("variables were replaced without credential permission")
	}
}

type serviceVariableRepository struct {
	*fakeRepository
	serviceSet *VariableSet
}

func (r *serviceVariableRepository) FindVariableSetByEnvironment(context.Context, int64) (*VariableSet, error) {
	return &VariableSet{ID: 25, ProjectID: 19, EnvironmentID: 17, ServiceID: 0}, nil
}

func (r *serviceVariableRepository) FindVariableSetByEnvironmentService(context.Context, int64, int64) (*VariableSet, error) {
	return r.serviceSet, nil
}

func TestUpdateServiceVariablesReturnsServiceSetInsteadOfEnvironmentDefaults(t *testing.T) {
	serviceSet := &VariableSet{ID: 26, ProjectID: 19, EnvironmentID: 17, ServiceID: 17, Scope: "environment", Revision: 2}
	repo := &serviceVariableRepository{fakeRepository: &fakeRepository{
		environment: &ProjectEnvironment{ID: 17, ProjectID: 19, Code: "qa"}, variableSet: serviceSet,
	}, serviceSet: serviceSet}
	logic := &Logic{repo: repo, authorizer: authz.New(policyStore{
		{Resource: "environment", Action: "update", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 19, Environment: "*"},
		{Resource: "environment", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 19, Environment: "*"},
	})}
	set, code := logic.UpdateVariableSet(context.Background(), 1, 26, &UpdateVariableSetRequest{Revision: 1, Items: []VariableItemInput{{VariableKey: "port", Value: "8001"}}})
	if code != errcode.CodeSuccess || set != serviceSet || !repo.replaced {
		t.Fatalf("set=%+v code=%d", set, code)
	}
}

func TestSaveServiceOverrideRejectsPendingHostEnvironment(t *testing.T) {
	repo := &fakeRepository{
		environment: &ProjectEnvironment{ID: 3, ProjectID: 10, Code: "test", EnvironmentType: EnvironmentTypeHostGroup, BindingStatus: BindingStatusPendingValidation},
		service:     &ServiceTarget{ID: 4, ProjectID: 10, DeliveryType: DeliveryTypeHost},
	}
	logic := &Logic{repo: repo, authorizer: authz.New(policyStore{
		{Resource: "environment", Action: "update", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10, Environment: "*"},
		{Resource: "service", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10, Environment: "*"},
	})}
	_, code := logic.SaveServiceOverride(context.Background(), 7, 3, 4, &SaveServiceOverrideRequest{ServiceRevisionID: 9})
	if code != errcode.CodeParamInvalid {
		t.Fatalf("code=%d want invalid parameter", code)
	}
}

func TestSaveServiceOverrideUsesOptimisticRevision(t *testing.T) {
	replicas := 2
	repo := &fakeRepository{
		environment:           &ProjectEnvironment{ID: 3, ProjectID: 10, Code: "test", EnvironmentType: EnvironmentTypeK8sNamespace, BindingStatus: BindingStatusValidated},
		service:               &ServiceTarget{ID: 4, ProjectID: 10, DeliveryType: DeliveryTypeK8sYAML},
		serviceRevisionExists: true,
		overrideSaved:         true,
	}
	logic := &Logic{repo: repo, authorizer: authz.New(policyStore{
		{Resource: "environment", Action: "update", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10, Environment: "*"},
		{Resource: "environment", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10, Environment: "*"},
		{Resource: "service", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10, Environment: "*"},
	})}
	override, code := logic.SaveServiceOverride(context.Background(), 7, 3, 4, &SaveServiceOverrideRequest{
		ExpectedRevision: 2, ServiceRevisionID: 9, ImageRef: "registry/payment@sha256:abc", Replicas: &replicas,
	})
	if code != errcode.CodeSuccess || override == nil || override.Revision != 3 {
		t.Fatalf("override=%+v code=%d", override, code)
	}
}

func TestGetServiceOverrideAllowsInvalidBindingForAudit(t *testing.T) {
	repo := &fakeRepository{
		environment:     &ProjectEnvironment{ID: 3, ProjectID: 10, Code: "test", EnvironmentType: EnvironmentTypeHostGroup, BindingStatus: BindingStatusPendingValidation},
		service:         &ServiceTarget{ID: 4, ProjectID: 10, DeliveryType: DeliveryTypeHost},
		serviceOverride: &EnvironmentServiceOverride{ID: 8, ProjectEnvironmentID: 3, ServiceID: 4, ServiceRevisionID: 9, Revision: 2},
	}
	logic := &Logic{repo: repo, authorizer: authz.New(policyStore{
		{Resource: "environment", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10, Environment: "*"},
		{Resource: "service", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10, Environment: "*"},
	})}
	override, code := logic.GetServiceOverride(context.Background(), 7, 3, 4)
	if code != errcode.CodeSuccess || override == nil || override.ID != 8 {
		t.Fatalf("override=%+v code=%d", override, code)
	}
}

func TestCloneRejectsProductionSource(t *testing.T) {
	repo := &fakeRepository{environment: &ProjectEnvironment{ID: 3, ProjectID: 10, Code: "prod", Status: "active", IsProduction: true}}
	logic := &Logic{repo: repo, authorizer: authz.New(policyStore{{
		Resource: "environment", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10, Environment: "*",
	}})}
	_, code := logic.Clone(context.Background(), 7, 3, &CloneEnvironmentRequest{Name: "Test", Code: "test", ClusterID: 2, Namespace: "payment-test"})
	if code != errcode.CodeParamInvalid {
		t.Fatalf("code=%d want invalid parameter", code)
	}
	if repo.cloned != nil {
		t.Fatal("production source was cloned")
	}
}

func TestCloneRechecksTargetClusterPermission(t *testing.T) {
	repo := &fakeRepository{
		environment: &ProjectEnvironment{ID: 3, ProjectID: 10, Code: "test", Status: "active"},
		profile:     &ProjectDeliveryProfile{DeliveryType: DeliveryTypeK8sYAML, DeliveryStatus: "active"},
		cluster:     &ClusterTarget{ID: 2, Status: "connected"},
	}
	logic := &Logic{repo: repo, authorizer: authz.New(policyStore{{
		Resource: "environment", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10, Environment: "*",
	}, {
		Resource: "environment", Action: "create", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10, Environment: "*",
	}})}
	_, code := logic.Clone(context.Background(), 7, 3, &CloneEnvironmentRequest{Name: "Test 2", Code: "test-2", ClusterID: 2, Namespace: "payment-test-2"})
	if code != errcode.CodeNoPermission {
		t.Fatalf("code=%d want no permission", code)
	}
	if repo.cloned != nil {
		t.Fatal("clone persisted without target cluster permission")
	}
}

func TestCloneBuildsNonProductionTarget(t *testing.T) {
	repo := &fakeRepository{
		environment: &ProjectEnvironment{ID: 3, ProjectID: 10, Code: "test", Status: "active"},
		profile:     &ProjectDeliveryProfile{DeliveryType: DeliveryTypeK8sYAML, DeliveryStatus: "active"},
		cluster:     &ClusterTarget{ID: 2, Status: "connected"},
	}
	logic := &Logic{repo: repo, authorizer: authz.New(policyStore{
		{Resource: "environment", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10, Environment: "*"},
		{Resource: "environment", Action: "create", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10, Environment: "*"},
		{Resource: "cluster", Action: "view", Effect: "allow", ScopeType: authz.ScopeCluster, ScopeID: 2, Environment: "*"},
	})}
	id, code := logic.Clone(context.Background(), 7, 3, &CloneEnvironmentRequest{Name: "Test 2", Code: "test-2", ClusterID: 2, Namespace: "payment-test-2", DeployApproval: true})
	if code != errcode.CodeSuccess || id != 84 {
		t.Fatalf("id=%d code=%d", id, code)
	}
	if repo.cloned == nil || repo.cloned.IsProduction || !repo.cloned.DeployApproval {
		t.Fatalf("unexpected target: %+v", repo.cloned)
	}
}
