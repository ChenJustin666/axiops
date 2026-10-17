package projectenv

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"github.com/ChenJustin666/axiops/internal/shared/middleware"
)

func TestServiceConfigRejectsUnscopedWrites(t *testing.T) {
	for _, crossProject := range []bool{false, true} {
		svcProject := int64(19)
		if crossProject {
			svcProject = 20
		}
		repo := &fakeRepository{environment: &ProjectEnvironment{ID: 1, ProjectID: 19, Code: "prod"}, service: &ServiceTarget{ID: 17, ProjectID: svcProject}}
		policies := policyStore{{Resource: "environment", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 19, Environment: "*"}, {Resource: "service", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 19, Environment: "*"}}
		if crossProject {
			policies = append(policies, authz.Policy{Resource: "environment", Action: "update", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 19, Environment: "*"})
		}
		handler := NewHandler(&Logic{repo: repo, authorizer: authz.New(policies)})
		req := httptest.NewRequest("POST", "/services/17/environment-config", strings.NewReader(`{"targets":[{"environment_id":1,"token":"`+strings.Repeat("a", 64)+`"}],"revision_id":3,"variables":{}}`))
		req.SetPathValue("service_id", "17")
		req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyUserID, int64(7)))
		w := httptest.NewRecorder()
		handler.SaveServiceConfig(w, req)
		var body struct {
			Code int `json:"code"`
		}
		if json.Unmarshal(w.Body.Bytes(), &body) != nil {
			t.Fatal(w.Body.String())
		}
		want := errcode.CodeNoPermission
		if crossProject {
			want = errcode.CodeServiceNotFound
		}
		if body.Code != want {
			t.Fatalf("crossProject=%v code=%d want=%d", crossProject, body.Code, want)
		}
	}
}

func TestConfigRequestRejectsDuplicateTargetsAndSystemVariables(t *testing.T) {
	req := SaveConfigRequest{Targets: []ConfigTarget{{EnvironmentID: 1, Token: strings.Repeat("a", 64)}}, RevisionID: 1}
	if !validConfigRequest(&req) {
		t.Fatal("valid request rejected")
	}
	req.Targets = append(req.Targets, req.Targets[0])
	if validConfigRequest(&req) {
		t.Fatal("duplicate accepted")
	}
	req.Targets = req.Targets[:1]
	value := "other"
	req.Variables = map[string]*string{"Namespace": &value}
	if validConfigRequest(&req) {
		t.Fatal("system variable accepted")
	}
}

func TestYAMLSyncRequiresDistinctSourceAndSingleTarget(t *testing.T) {
	definition := "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: app"
	req := SaveConfigRequest{
		Source:     &ConfigTarget{EnvironmentID: 1, Token: strings.Repeat("a", 64)},
		Targets:    []ConfigTarget{{EnvironmentID: 2, Token: strings.Repeat("b", 64)}},
		Definition: &definition,
	}
	if !validConfigRequest(&req) {
		t.Fatal("source-guarded YAML sync rejected")
	}
	req.Targets = append(req.Targets, ConfigTarget{EnvironmentID: 3, Token: strings.Repeat("c", 64)})
	if validConfigRequest(&req) {
		t.Fatal("multiple YAML targets accepted")
	}
	req.Targets = req.Targets[:1]
	req.Source.EnvironmentID = 2
	if validConfigRequest(&req) {
		t.Fatal("source equals target accepted")
	}
	req.Definition = nil
	req.RevisionID = 10
	if validConfigRequest(&req) {
		t.Fatal("self sync without YAML accepted")
	}
}
