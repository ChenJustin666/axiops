package codesource

import (
	"context"
	"fmt"
	"github.com/ChenJustin666/axiops/pkg/gitprovider"
	"testing"
)

type chartProvider struct {
	gitprovider.Provider
	entries map[string][]gitprovider.RepoFile
	content map[string]string
	reads   []string
}

func (p *chartProvider) ListRepoFiles(_ context.Context, _, commit, dir string) ([]gitprovider.RepoFile, error) {
	if commit != "frozen" {
		return nil, fmt.Errorf("mutable listing: %s", commit)
	}
	return p.entries[dir], nil
}
func (p *chartProvider) GetFileContent(_ context.Context, _, name, commit string) ([]byte, error) {
	if commit != "frozen" {
		return nil, fmt.Errorf("mutable read: %s", commit)
	}
	p.reads = append(p.reads, name)
	return []byte(p.content[name]), nil
}
func TestReadChartTreeFreezesFilesAndRespectsHelmIgnore(t *testing.T) {
	p := &chartProvider{entries: map[string][]gitprovider.RepoFile{
		"deploy/chart":           {{Name: "Chart.yaml", Path: "deploy/chart/Chart.yaml", Type: "file"}, {Name: ".helmignore", Path: "deploy/chart/.helmignore", Type: "file"}, {Name: "docs", Path: "deploy/chart/docs", Type: "dir"}, {Name: "templates", Path: "deploy/chart/templates", Type: "dir"}},
		"deploy/chart/templates": {{Name: "_helpers.tpl", Path: "deploy/chart/templates/_helpers.tpl", Type: "file"}, {Name: ".scratch", Path: "deploy/chart/templates/.scratch", Type: "file"}},
	}, content: map[string]string{"deploy/chart/.helmignore": "docs/\n", "deploy/chart/Chart.yaml": "native", "deploy/chart/templates/_helpers.tpl": "{{ define \"app\" }}app{{ end }}"}}
	files, err := readChartTree(context.Background(), p, "team/chart", "frozen", "deploy/chart")
	if err != nil || len(files) != 3 {
		t.Fatalf("files=%v error=%v", files, err)
	}
	for _, name := range p.reads {
		if name == "deploy/chart/templates/.scratch" {
			t.Fatal("ignored file was read")
		}
	}
	if files[2].Path != "deploy/chart/templates/_helpers.tpl" {
		t.Fatal("native helper omitted")
	}
}
func TestReadChartTreeRejectsEscapesSymlinksAndLargeFiles(t *testing.T) {
	for _, entry := range []gitprovider.RepoFile{{Name: "escape", Path: "../other", Type: "file"}, {Name: "nested", Path: "chart/deeper/file", Type: "file"}, {Name: "link", Path: "chart/link", Type: "symlink"}, {Name: "large", Path: "chart/large", Type: "file", Size: 2 << 20}} {
		p := &chartProvider{entries: map[string][]gitprovider.RepoFile{"chart": {entry}}}
		if _, err := readChartTree(context.Background(), p, "team/chart", "frozen", "chart"); err == nil {
			t.Fatalf("unsafe entry accepted: %+v", entry)
		}
	}
}
func TestReadChartTreeAtRepositoryRoot(t *testing.T) {
	p := &chartProvider{entries: map[string][]gitprovider.RepoFile{"": {{Name: "Chart.yaml", Path: "Chart.yaml", Type: "file"}}}}
	if files, err := readChartTree(context.Background(), p, "team/chart", "frozen", "."); err != nil || len(files) != 1 {
		t.Fatalf("root Chart: %v %v", files, err)
	}
}
