package workflow

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"testing"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
)

// Called only inside the disposable workflow schema, never against application
// records. The HTTP handlers use the real repository and policy evaluator.
func testMySQLWorkflowAuthorization(t *testing.T, database *sql.DB) {
	exec := func(query string) {
		t.Helper()
		if _, err := database.Exec(query); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO project(id) VALUES(9013),(9014)`)
	exec(`INSERT INTO project_environment(id,project_id,name,code,environment_type,cluster_id,namespace,binding_status,status) VALUES
		(701,9013,'Test','qa','k8s_namespace',2,'authz-qa','validated','active'),
		(702,9013,'Production','prod','k8s_namespace',3,'authz-prod','validated','active'),
		(703,9013,'Staging','staging','k8s_namespace',2,'authz-staging','validated','active'),
		(704,9014,'Other','qa','k8s_namespace',2,'authz-other','validated','active')`)
	exec(`INSERT INTO workflow(id,project_id,name,code,delivery_type,target_environment_id,status,definition_version) VALUES
		(801,9013,'Test','authz-test','k8s_yaml',701,'active',1),
		(802,9013,'Production','authz-prod','k8s_yaml',702,'active',1),
		(803,9013,'Staging','authz-staging','k8s_yaml',703,'active',1),
		(804,9014,'Other','authz-other','k8s_yaml',704,'active',1),
		(805,9013,'Unbound','authz-build','k8s_yaml',0,'active',1)`)
	exec(`INSERT INTO workflow_run(id,workflow_id,project_id,environment_id,definition_version,status,context_snapshot) VALUES
		(9001,801,9013,701,1,'succeeded','{}'),(9002,802,9013,702,1,'succeeded','{}'),
		(9006,801,9013,702,1,'succeeded','{}')`)
	policies := policyStore{
		{Resource: "workflow", Action: "view", Effect: "allow", ScopeType: authz.ScopePlatform},
		{Resource: "workflow", Action: "view", Effect: "deny", ScopeType: authz.ScopeProject, ScopeID: 9013, Environment: "prod"},
	}
	repo := NewRepository(database)
	logic := &Logic{repo: repo, authorizer: authz.New(policies)}
	ctx := actorContext(7, "scoped-reader")
	items, code := logic.List(ctx, 7, 9013)
	if code != errcode.CodeSuccess {
		t.Fatalf("list code=%d", code)
	}
	ids := []int64{}
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	// The SQL order uses update time, then descending ID.
	if !reflect.DeepEqual(ids, []int64{805, 803, 801}) {
		t.Fatalf("workflow list=%v", ids)
	}
	for _, id := range []int64{801, 803, 805} {
		if _, code := logic.Get(ctx, 7, id); code != errcode.CodeSuccess {
			t.Fatalf("visible detail %d rejected: %d", id, code)
		}
	}
	if _, code := logic.Get(ctx, 7, 802); code != errcode.CodeNoPermission {
		t.Fatalf("production detail leaked: %d", code)
	}
	for page := 1; page <= 2; page++ {
		runs, total, code := logic.ListRuns(ctx, 7, 801, page, 1)
		if code != errcode.CodeSuccess || total != 1 || len(runs) != 2-page {
			t.Fatalf("page=%d code=%d total=%d runs=%v", page, code, total, runs)
		}
		if len(runs) > 0 && runs[0].ID != 9001 {
			t.Fatalf("production history leaked: %d", runs[0].ID)
		}
	}
	if _, code := logic.GetRun(ctx, 7, 9006); code != errcode.CodeNoPermission {
		t.Fatalf("production run detail leaked: %d", code)
	}
	// Changing a workflow's current binding cannot grant access to old runs in
	// a different environment: run authorization uses its own stored target.
	h := NewHandler(logic)
	for _, tc := range []struct {
		id   int64
		run  bool
		want int
	}{
		{801, false, http.StatusOK}, {802, false, http.StatusForbidden}, {9001, true, http.StatusOK}, {9006, true, http.StatusForbidden},
	} {
		r := httptest.NewRequest(http.MethodGet, "/read", nil).WithContext(ctx)
		r.SetPathValue("id", strconv.FormatInt(tc.id, 10))
		w := httptest.NewRecorder()
		if tc.run {
			h.GetRun(w, r)
		} else {
			h.Get(w, r)
		}
		if w.Code != tc.want {
			t.Fatalf("HTTP id=%d status=%d want=%d", tc.id, w.Code, tc.want)
		}
	}
	r := httptest.NewRequest(http.MethodGet, "/runs?page=1&page_size=1", nil).WithContext(ctx)
	r.SetPathValue("id", "801")
	w := httptest.NewRecorder()
	h.ListRuns(w, r)
	var response struct {
		Code int `json:"code"`
		Data struct {
			Total int           `json:"total"`
			List  []WorkflowRun `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusOK || response.Code != 0 || response.Data.Total != 1 || len(response.Data.List) != 1 || response.Data.List[0].ID != 9001 {
		t.Fatalf("filtered HTTP page=%s", w.Body.String())
	}
	// A grant constrained to an environment must work throughout the read chain.
	logic.authorizer = authz.New(policyStore{{Resource: "workflow", Action: "view", Effect: "allow", ScopeType: authz.ScopeEnvironment, ScopeID: 701, Environment: "qa"}})
	items, code = logic.List(ctx, 7, 9013)
	if code != errcode.CodeSuccess || len(items) != 1 || items[0].ID != 801 {
		t.Fatalf("environment-scoped list=%v code=%d", items, code)
	}
	if _, _, code := logic.ListRuns(ctx, 7, 801, 1, 20); code != errcode.CodeSuccess {
		t.Fatalf("environment-scoped runs rejected: %d", code)
	}
	if items, code := logic.List(ctx, 7, 9014); code != errcode.CodeSuccess || len(items) != 0 {
		t.Fatalf("same code in another project leaked: %v code=%d", items, code)
	}
}
