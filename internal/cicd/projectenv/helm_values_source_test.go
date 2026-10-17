package projectenv

import (
	"context"
	"errors"
	"github.com/ChenJustin666/axiops/internal/cicd/codesource"
	"github.com/ChenJustin666/axiops/internal/cicd/helmconfig"
	"strings"
	"testing"
)

type fakeValuesCodeSource struct {
	projectID int64
	denied    bool
	raw       string
	commit    string
	reads     int
}

func (s *fakeValuesCodeSource) Get(context.Context, int64, int64) (*codesource.CodeSource, error) {
	if s.denied {
		return nil, errors.New("sensitive-provider-detail")
	}
	return &codesource.CodeSource{ProjectID: s.projectID}, nil
}
func (s *fakeValuesCodeSource) ReadFiles(_ context.Context, _ int64, _ int64, _ string, _ string, paths []string) (*codesource.RepositorySnapshot, error) {
	s.reads++
	return &codesource.RepositorySnapshot{CommitID: s.commit, Files: []codesource.RepositoryFile{{Path: paths[0], Content: []byte(s.raw)}}}, nil
}

func TestHelmValuesSourceScopeAndSnapshot(t *testing.T) {
	source := &fakeValuesCodeSource{projectID: 7, raw: "replicas: 2\n", commit: "fixed-commit"}
	l := (&Logic{}).WithCodeSources(source)
	input := &helmconfig.ValuesSource{CodeSourceID: 4, RepoPath: "team/config", Ref: "main", Path: "deploy/values-prod.yaml"}
	actual, err := l.readValuesSource(context.Background(), 9, 7, input)
	if err != nil || actual.CommitID != "fixed-commit" || actual.Base != source.raw || input.CommitID != "" {
		t.Fatal("snapshot not fixed or input mutated", err)
	}
	source.projectID = 8
	if _, err = l.readValuesSource(context.Background(), 9, 7, input); err == nil || source.reads != 1 {
		t.Fatal("cross-project values source read")
	}
	source.projectID = 0
	if _, err = l.readValuesSource(context.Background(), 9, 7, input); err != nil {
		t.Fatal("global source unavailable", err)
	}
	source.denied = true
	if _, err = l.readValuesSource(context.Background(), 9, 7, input); err == nil || strings.Contains(err.Error(), "sensitive-provider-detail") {
		t.Fatal("provider error leaked or revocation ignored")
	}
	source.denied = false
	source.raw = "a: 1\na: 2\n"
	if _, err = l.readValuesSource(context.Background(), 9, 7, input); err == nil {
		t.Fatal("duplicate keys imported")
	}
	for _, p := range []string{"../values.yaml", "/values.yaml", "values.json", "dir\\values.yaml"} {
		input.Path = p
		if _, err = l.readValuesSource(context.Background(), 9, 7, input); err == nil {
			t.Fatal("unsafe values path", p)
		}
	}
}
