package codesource

import (
	"context"
	"testing"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/secretbox"
)

type testPolicyStore []authz.Policy

func (p testPolicyStore) ListUserPolicies(context.Context, int64) ([]authz.Policy, error) {
	return p, nil
}

type fakeStore struct {
	source     *CodeSource
	credential *CredentialRef
	created    *CodeSource
}

func (s *fakeStore) Create(_ context.Context, source *CodeSource) (int64, error) {
	s.created = source
	return 11, nil
}
func (s *fakeStore) FindByID(context.Context, int64) (*CodeSource, error) { return s.source, nil }
func (s *fakeStore) List(context.Context, int64, authz.ScopeSet) ([]CodeSource, error) {
	if s.source == nil {
		return nil, nil
	}
	return []CodeSource{*s.source}, nil
}
func (s *fakeStore) Update(context.Context, *CodeSource) error { return nil }
func (s *fakeStore) Delete(context.Context, int64) error       { return nil }
func (s *fakeStore) FindCredential(context.Context, int64) (*CredentialRef, error) {
	return s.credential, nil
}

func TestCreateRequiresProjectPermissionAndMatchingCredential(t *testing.T) {
	box, _ := secretbox.New("test-key")
	encrypted, _ := box.Encrypt(`{"token":"secret"}`)
	store := &fakeStore{credential: &CredentialRef{ID: 8, Type: "git_token", Data: encrypted, ProjectID: 10}}
	logic := &Logic{repo: store, authorizer: authz.New(testPolicyStore{{
		Resource: "code_source", Action: "create", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10,
	}, {
		Resource: "credential", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10,
	}}), box: box}
	id, err := logic.Create(context.Background(), 7, 10, CreateRequest{Name: "GitLab", Type: "gitlab", URL: "https://git.example.com", CredentialID: 8})
	if err != nil || id != 11 || store.created == nil || store.created.ProjectID != 10 {
		t.Fatalf("id=%d created=%+v err=%v", id, store.created, err)
	}
}

func TestCreateAcceptsInternalHTTPSource(t *testing.T) {
	box, _ := secretbox.New("test-key")
	encrypted, _ := box.Encrypt(`{"token":"secret"}`)
	store := &fakeStore{credential: &CredentialRef{ID: 8, Type: "git_token", Data: encrypted, ProjectID: 0}}
	logic := &Logic{repo: store, authorizer: authz.New(testPolicyStore{{
		Resource: "code_source", Action: "create", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10,
	}, {
		Resource: "credential", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10,
	}}), box: box}
	if _, err := logic.Create(context.Background(), 7, 10, CreateRequest{Name: "Internal Gitea", Type: "gitea", URL: "http://gitea.intra", CredentialID: 8}); err != nil {
		t.Fatalf("internal HTTP source rejected: %v", err)
	}
}

func TestCreateRejectsCredentialFromAnotherProject(t *testing.T) {
	box, _ := secretbox.New("test-key")
	encrypted, _ := box.Encrypt(`{"token":"secret"}`)
	store := &fakeStore{credential: &CredentialRef{ID: 8, Type: "git_token", Data: encrypted, ProjectID: 99}}
	logic := &Logic{repo: store, authorizer: authz.New(testPolicyStore{{
		Resource: "code_source", Action: "create", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10,
	}, {
		Resource: "credential", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10,
	}}), box: box}
	if _, err := logic.Create(context.Background(), 7, 10, CreateRequest{Name: "GitLab", Type: "gitlab", URL: "https://git.example.com", CredentialID: 8}); err == nil || store.created != nil {
		t.Fatalf("created=%+v err=%v", store.created, err)
	}
}

func TestGetRejectsDifferentProjectScope(t *testing.T) {
	store := &fakeStore{source: &CodeSource{ID: 5, ProjectID: 10}}
	logic := &Logic{repo: store, authorizer: authz.New(testPolicyStore{{
		Resource: "code_source", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 99,
	}})}
	if _, err := logic.Get(context.Background(), 7, 5); err != ErrNoPermission {
		t.Fatalf("err=%v", err)
	}
}

func TestGlobalSourceCanBeConsumedByProjectScopedUser(t *testing.T) {
	store := &fakeStore{source: &CodeSource{ID: 5, ProjectID: 0}}
	logic := &Logic{repo: store, authorizer: authz.New(testPolicyStore{{
		Resource: "code_source", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10,
	}})}
	if _, err := logic.Get(context.Background(), 7, 5); err != nil {
		t.Fatalf("global source should be readable from a project scope: %v", err)
	}
}

func TestGlobalSourceMutationStillRequiresPlatformPermission(t *testing.T) {
	box, _ := secretbox.New("test-key")
	encrypted, _ := box.Encrypt(`{"token":"secret"}`)
	store := &fakeStore{credential: &CredentialRef{ID: 8, Type: "git_token", Data: encrypted, ProjectID: 0}}
	logic := &Logic{repo: store, authorizer: authz.New(testPolicyStore{{
		Resource: "code_source", Action: "create", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10,
	}, {
		Resource: "credential", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10,
	}}), box: box}
	if _, err := logic.Create(context.Background(), 7, 0, CreateRequest{Name: "Gitea", Type: "gitea", URL: "http://gitea.local", CredentialID: 8}); err != ErrNoPermission {
		t.Fatalf("global create err=%v, want permission denied", err)
	}
}

func TestUpdateAllowsChangingProviderTypeAndCredential(t *testing.T) {
	box, _ := secretbox.New("test-key")
	encrypted, _ := box.Encrypt(`{"token":"new-token"}`)
	store := &fakeStore{
		source:     &CodeSource{ID: 5, ProjectID: 0, Name: "old", Type: "gitea", URL: "http://git.local", CredentialID: 8, Status: "active"},
		credential: &CredentialRef{ID: 8, Type: "git_token", Data: encrypted, ProjectID: 0},
	}
	logic := &Logic{repo: store, authorizer: authz.New(testPolicyStore{
		{Resource: "code_source", Action: "update", Effect: "allow", ScopeType: authz.ScopePlatform},
		{Resource: "credential", Action: "view", Effect: "allow", ScopeType: authz.ScopePlatform},
	}), box: box}
	err := logic.Update(context.Background(), 1, 5, UpdateRequest{
		Name:         "company-gitlab",
		Type:         "gitlab",
		URL:          "https://git.example.com",
		CredentialID: 8,
		Status:       "active",
	})
	if err != nil {
		t.Fatal(err)
	}
	if store.source.Type != "gitlab" || store.source.Name != "company-gitlab" || store.source.CredentialID != 8 {
		t.Fatalf("updated source=%+v", store.source)
	}
}
