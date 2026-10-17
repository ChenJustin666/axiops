package helmconfig

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
)

// SameFiles ignores JSON formatting and file order, but never content changes.
func SameFiles(left, right string) bool {
	type file struct{ Path, Content string }
	var a, b struct {
		Files []file `json:"files"`
	}
	if json.Unmarshal([]byte(left), &a) != nil || json.Unmarshal([]byte(right), &b) != nil || len(a.Files) == 0 || len(a.Files) != len(b.Files) {
		return false
	}
	files := map[string]string{}
	for _, f := range a.Files {
		if _, ok := files[f.Path]; ok {
			return false
		}
		files[f.Path] = f.Content
	}
	for _, f := range b.Files {
		if content, ok := files[f.Path]; !ok || content != f.Content {
			return false
		}
		delete(files, f.Path)
	}
	return len(files) == 0
}

// Shared by service revision writes and deployment reads: direct API writes
// must receive the same integrity/dependency checks as template/Git imports.
func LoadChart(raw []byte) (*chart.Chart, error) {
	var definition struct {
		SchemaVersion int `json:"schema_version"`
		Files         []struct {
			Path    string `json:"path"`
			Content string `json:"content"`
			SHA256  string `json:"sha256"`
		} `json:"files"`
	}
	if len(raw) > 4<<20 || json.Unmarshal(raw, &definition) != nil || definition.SchemaVersion != 1 || len(definition.Files) == 0 || len(definition.Files) > 500 {
		return nil, fmt.Errorf("invalid immutable Helm Chart definition")
	}
	files := make([]*loader.BufferedFile, 0, len(definition.Files))
	seen := map[string]bool{}
	for _, f := range definition.Files {
		name := path.Clean(strings.TrimSpace(f.Path))
		if name != f.Path || name == "." || name == ".." || strings.HasPrefix(name, "/") || strings.HasPrefix(name, "../") || strings.ContainsAny(name, "\\\x00:") || seen[name] {
			return nil, fmt.Errorf("invalid or duplicate Helm Chart file path")
		}
		seen[name] = true
		digest := sha256.Sum256([]byte(f.Content))
		if !strings.EqualFold(f.SHA256, hex.EncodeToString(digest[:])) {
			return nil, fmt.Errorf("Helm Chart file digest mismatch")
		}
		files = append(files, &loader.BufferedFile{Name: name, Data: []byte(f.Content)})
	}
	c, err := loader.LoadFiles(files)
	if err != nil {
		return nil, fmt.Errorf("load Helm Chart: %w", err)
	}
	if c.Metadata == nil || c.Metadata.APIVersion != "v2" || c.Metadata.Type != "application" && c.Metadata.Type != "" {
		return nil, fmt.Errorf("only Helm v2 application charts are deployable")
	}
	if err := action.CheckDependencies(c, c.Metadata.Dependencies); err != nil {
		return nil, fmt.Errorf("Helm Chart dependencies are incomplete: %w", err)
	}
	return c, nil
}
