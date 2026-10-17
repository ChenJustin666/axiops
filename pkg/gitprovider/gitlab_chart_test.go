package gitprovider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGitLabChartTreeReadsAllPagesAndPreservesUnsafeTypes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("ref") != "frozen" || r.URL.Query().Get("path") != "deploy/chart" {
			t.Error("tree request lost immutable commit or directory")
		}
		switch r.URL.Query().Get("page") {
		case "1":
			fmt.Fprint(w, `[{"name":"Chart.yaml","path":"deploy/chart/Chart.yaml","type":"blob","mode":"100644"}]`)
		case "2":
			fmt.Fprint(w, `[{"name":"link","path":"deploy/chart/link","type":"blob","mode":"120000"},{"name":"dependency","path":"deploy/chart/dependency","type":"commit","mode":"160000"}]`)
		default:
			fmt.Fprint(w, `[]`)
		}
	}))
	defer server.Close()
	p, _ := newGitLabProvider(&Config{BaseURL: server.URL})
	files, err := p.ListRepoFiles(context.Background(), "team/chart", "frozen", "deploy/chart")
	if err != nil || len(files) != 3 || files[0].Type != "file" || files[1].Type != "symlink" || files[2].Type != "commit" {
		t.Fatalf("incomplete or unsafe Chart tree: %+v / %v", files, err)
	}
}
