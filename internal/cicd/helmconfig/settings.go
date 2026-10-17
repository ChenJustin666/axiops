package helmconfig

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const Kind = "AxiOpsHelmValues"

type ImageBinding struct {
	RepositoryPath string `json:"repository_path"`
	TagPath        string `json:"tag_path"`
	RefPath        string `json:"ref_path"`
}

type Settings struct {
	Values string        `json:"values"`
	Image  ImageBinding  `json:"image"`
	Source *ValuesSource `json:"source,omitempty"`
	// Loaded from the environment-owned shared row, never accepted from a client.
	SharedValues   string `json:"shared_values,omitempty"`
	SharedRevision int64  `json:"shared_revision,omitempty"`
	// Selected fields are frozen with the environment configuration. Credential
	// identities remain in variable_item and payloads are never stored here.
	CredentialKeys map[string]string `json:"credential_keys,omitempty"`
}

type ValuesSource struct {
	CodeSourceID int64  `json:"code_source_id"`
	RepoPath     string `json:"repo_path"`
	Ref          string `json:"ref"`
	Path         string `json:"path"`
	CommitID     string `json:"commit_id"`
	Base         string `json:"base"`
}

type CredentialBinding struct {
	CredentialID int64  `json:"credential_id"`
	Key          string `json:"key"`
}

func Default() Settings {
	return Settings{Values: "{}\n", Image: ImageBinding{RepositoryPath: "image.repository", TagPath: "image.tag"}}
}

func RowName(serviceID int64) string { return fmt.Sprintf("axiops-s%d-helm-values", serviceID) }

var PathPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+(?:\.[A-Za-z0-9_-]+)*$`)

func (s Settings) Validate() error {
	if _, err := ParseValues(s.Values); err != nil {
		return err
	}
	if s.Source != nil {
		v := s.Source
		if v.CodeSourceID <= 0 || len(v.RepoPath) > 512 || strings.TrimSpace(v.RepoPath) == "" || len(v.Ref) > 256 || strings.TrimSpace(v.Ref) == "" || len(v.CommitID) > 128 || strings.TrimSpace(v.CommitID) == "" || len(v.Path) > 512 || !ValidValuesFilePath(v.Path) {
			return fmt.Errorf("values 来源信息无效，请重新读取文件")
		}
		if _, err := ParseValues(v.Base); err != nil {
			return err
		}
	}
	i := s.Image
	if i.RefPath != "" {
		if !PathPattern.MatchString(i.RefPath) || i.RepositoryPath != "" || i.TagPath != "" {
			return fmt.Errorf("完整镜像路径与 repository/tag 映射只能选择一种")
		}
	} else if !PathPattern.MatchString(i.RepositoryPath) || !PathPattern.MatchString(i.TagPath) || PathsOverlap(i.RepositoryPath, i.TagPath) {
		return fmt.Errorf("请填写不同的 Helm 镜像 repository/tag values 路径")
	}
	if len(s.CredentialKeys) > 200 {
		return fmt.Errorf("凭据引用不能超过 200 项")
	}
	for path, key := range s.CredentialKeys {
		if !PathPattern.MatchString(path) || len(path) > 128 || !CredentialKeyPattern.MatchString(key) {
			return fmt.Errorf("凭据 values 路径或字段无效")
		}
	}
	return nil
}

var CredentialKeyPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,128}$`)

func PathsOverlap(a, b string) bool {
	return a == b || strings.HasPrefix(a, b+".") || strings.HasPrefix(b, a+".")
}

// Values retain YAML scalar types and arrays. Reject duplicate keys, multiple
// documents and non-JSON values before they can reach the Helm engine.
func ParseValues(raw string) (map[string]interface{}, error) {
	if len(raw) > 512*1024 {
		return nil, fmt.Errorf("Helm values 不能超过 512 KiB")
	}
	d := yaml.NewDecoder(strings.NewReader(raw))
	var values map[string]interface{}
	if err := d.Decode(&values); err != nil && err != io.EOF {
		return nil, fmt.Errorf("Helm values YAML 格式无效（请检查缩进、重复键和类型）")
	}
	var extra interface{}
	if err := d.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("Helm values 只能包含一个 YAML 对象")
	}
	if values == nil {
		values = map[string]interface{}{}
	}
	if _, err := json.Marshal(values); err != nil {
		return nil, fmt.Errorf("Helm values 的所有键必须为字符串，值必须兼容 JSON")
	}
	return values, nil
}

type Query interface {
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

func Load(ctx context.Context, q Query, environmentID, serviceID int64) (Settings, error) {
	s := Default()
	var raw string
	err := q.QueryRowContext(ctx, `SELECT manifest FROM environment_resource WHERE environment_id=? AND name=? AND kind=?`, environmentID, RowName(serviceID), Kind).Scan(&raw)
	if err != nil && err != sql.ErrNoRows {
		return s, err
	}
	if err == nil && json.Unmarshal([]byte(raw), &s) != nil {
		return s, fmt.Errorf("Helm 环境配置已损坏，请重新配置")
	}
	shared, err := LoadShared(ctx, q, environmentID)
	if err != nil {
		return s, err
	}
	s.SharedValues, s.SharedRevision = shared.Values, shared.Revision
	return s, s.Validate()
}
