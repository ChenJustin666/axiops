package serviceassets

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/validation"
)

// Stored separately from deployable environment resources; payload never holds
// credential plaintext. Existing tables preserve transaction/snapshot semantics.
const ResourceKind = "AxiOpsServiceSettings"

type Query interface {
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}
type Settings struct {
	Resources    *RuntimeResources `json:"resources,omitempty"`
	Replicas     *int              `json:"replicas,omitempty"`
	Command      []string          `json:"command,omitempty"`
	Args         []string          `json:"args,omitempty"`
	WorkloadKind string            `json:"workload_kind"`
	WorkloadName string            `json:"workload_name"`
	Container    string            `json:"container"`
	Files        []File            `json:"files"`
	Secrets      []Secret          `json:"secrets"`
	Routes       []Route           `json:"routes"`
	Volumes      []Volume          `json:"volumes"`
}

// Empty fields inherit the YAML value; disabling the override restores YAML.
type RuntimeResources struct {
	CPURequest    string `json:"cpu_request"`
	CPULimit      string `json:"cpu_limit"`
	MemoryRequest string `json:"memory_request"`
	MemoryLimit   string `json:"memory_limit"`
}

func (s Settings) ResourcesValue(key string) string {
	if s.Resources == nil {
		return ""
	}
	switch key {
	case "requests/cpu":
		return s.Resources.CPURequest
	case "limits/cpu":
		return s.Resources.CPULimit
	case "requests/memory":
		return s.Resources.MemoryRequest
	case "limits/memory":
		return s.Resources.MemoryLimit
	default:
		return ""
	}
}

func (r *RuntimeResources) Validate() error {
	if r == nil {
		return nil
	}
	for _, pair := range []struct {
		name, request, limit string
		memory               bool
	}{
		{"CPU", r.CPURequest, r.CPULimit, false}, {"内存", r.MemoryRequest, r.MemoryLimit, true},
	} {
		for _, v := range []string{pair.request, pair.limit} {
			if v == "" {
				continue
			}
			q, e := resource.ParseQuantity(v)
			if e != nil || strings.TrimSpace(v) != v || q.Sign() <= 0 {
				return fmt.Errorf("%s资源量无效：%q", pair.name, v)
			}
			if pair.memory && q.Cmp(resource.MustParse("1Mi")) < 0 {
				return fmt.Errorf("内存需填写单位且至少 1Mi，例如 1Gi，不能填写 1 或 1g")
			}
			if !pair.memory && q.Cmp(*resource.NewMilliQuantity(q.MilliValue(), resource.DecimalSI)) != 0 {
				return fmt.Errorf("CPU 精度不能小于 1m")
			}
		}
		if pair.request != "" && pair.limit != "" {
			a, _ := resource.ParseQuantity(pair.request)
			b, _ := resource.ParseQuantity(pair.limit)
			if a.Cmp(b) > 0 {
				return fmt.Errorf("%s request 不能大于 limit", pair.name)
			}
		}
	}
	return nil
}

type File struct {
	Name              string `json:"name"`
	Filename          string `json:"filename"`
	Content           string `json:"content"`
	MountPath         string `json:"mount_path"`
	ExistingConfigMap string `json:"existing_config_map"`
}
type Secret struct {
	Name           string `json:"name"`
	Group          string `json:"group,omitempty"`
	CredentialID   int64  `json:"credential_id"`
	CredentialKey  string `json:"credential_key"`
	ExistingSecret string `json:"existing_secret"`
	Key            string `json:"key"`
	EnvName        string `json:"env_name"`
	MountPath      string `json:"mount_path"`
}
type Route struct {
	Name        string            `json:"name"`
	Host        string            `json:"host"`
	Path        string            `json:"path"`
	PathType    string            `json:"path_type,omitempty"`
	Service     string            `json:"service"`
	Port        int               `json:"port"`
	Class       string            `json:"class"`
	TLSSecret   string            `json:"tls_secret"`
	TLSEnabled  bool              `json:"tls_enabled,omitempty"`
	TLSHosts    []string          `json:"tls_hosts,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}
type Volume struct {
	Name          string `json:"name"`
	ExistingClaim string `json:"existing_claim"`
	Capacity      string `json:"capacity"`
	StorageClass  string `json:"storage_class"`
	AccessMode    string `json:"access_mode"`
	MountPath     string `json:"mount_path"`
	ReadOnly      bool   `json:"read_only"`
}
type SecretRef struct {
	Name          string
	Key           string
	CredentialID  int64
	CredentialKey string
}

func Empty() Settings {
	return Settings{Files: []File{}, Secrets: []Secret{}, Routes: []Route{}, Volumes: []Volume{}}
}
func RowName(sid int64) string { return fmt.Sprintf("axiops-service-settings-%d", sid) }
func ResourceName(sid int64, prefix, name string) string {
	return fmt.Sprintf("axiops-s%d-%s-%s", sid, prefix, name)
}
func Load(ctx context.Context, q Query, eid, sid int64) (Settings, error) {
	var raw string
	err := q.QueryRowContext(ctx, `SELECT manifest FROM environment_resource WHERE environment_id=? AND kind=? AND name=?`, eid, ResourceKind, RowName(sid)).Scan(&raw)
	if err == sql.ErrNoRows {
		return Empty(), nil
	}
	if err != nil {
		return Settings{}, err
	}
	s := Empty()
	if err = json.Unmarshal([]byte(raw), &s); err != nil {
		return s, fmt.Errorf("服务资源配置损坏，请重新加载")
	}
	return s, Validate(s)
}
func (s Settings) Active() bool {
	return s.Resources != nil || s.Replicas != nil || len(s.Command)+len(s.Args)+len(s.Files)+len(s.Secrets)+len(s.Routes)+len(s.Volumes) > 0
}

var envKey = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func validName(v string) bool { return v != "" && len(validation.IsDNS1123Subdomain(v)) == 0 }
func Validate(s Settings) error {
	if err := s.Resources.Validate(); err != nil {
		return err
	}
	if s.Replicas != nil && (*s.Replicas < 0 || *s.Replicas > 10000) {
		return fmt.Errorf("副本数必须在 0 到 10000 之间")
	}
	if len(s.Command) > 32 || len(s.Args) > 64 {
		return fmt.Errorf("启动命令或参数数量超过限制")
	}
	for _, value := range append(append([]string{}, s.Command...), s.Args...) {
		if len(value) > 4096 || strings.ContainsAny(value, "\x00\r") {
			return fmt.Errorf("启动命令或参数包含无效内容")
		}
	}
	if len(s.Files) > 30 || len(s.Secrets) > 30 || len(s.Routes) > 20 || len(s.Volumes) > 20 {
		return fmt.Errorf("配置项数量超过限制")
	}
	if s.WorkloadName != "" && !validName(s.WorkloadName) {
		return fmt.Errorf("工作负载名称无效")
	}
	if s.Container != "" && len(validation.IsDNS1123Label(s.Container)) > 0 {
		return fmt.Errorf("容器名称无效")
	}
	if s.WorkloadKind != "" && !strings.Contains("|Deployment|StatefulSet|DaemonSet|Job|CronJob|", "|"+s.WorkloadKind+"|") {
		return fmt.Errorf("不支持此工作负载类型")
	}
	names, paths, envs := map[string]bool{}, map[string]bool{}, map[string]bool{}
	checkName := func(v string) bool {
		if len(v) > 30 || len(validation.IsDNS1123Label(v)) > 0 || names[v] {
			return false
		}
		names[v] = true
		return true
	}
	checkPath := func(v string) bool {
		if !strings.HasPrefix(v, "/") || v == "/" || strings.ContainsAny(v, "\\\x00\r\n") || path.Clean(v) != v || paths[v] {
			return false
		}
		for old := range paths {
			if strings.HasPrefix(v, old+"/") || strings.HasPrefix(old, v+"/") {
				return false
			}
		}
		paths[v] = true
		return true
	}
	for _, f := range s.Files {
		if !checkName(f.Name) || f.Filename == "" || len(validation.IsConfigMapKey(f.Filename)) > 0 || f.Filename == "." || f.Filename == ".." || len(f.Content) > 256*1024 || !checkPath(f.MountPath) {
			return fmt.Errorf("配置文件名称、键名、大小或挂载路径无效/冲突")
		}
		if f.ExistingConfigMap != "" && (!validName(f.ExistingConfigMap) || f.Content != "") {
			return fmt.Errorf("引用已有ConfigMap时不能同时填写文件内容")
		}
	}
	groups := map[string]Secret{}
	groupKeys := map[string]bool{}
	for _, s := range s.Secrets {
		if !checkName(s.Name) || s.Key == "" || s.Key == "." || s.Key == ".." || len(validation.IsConfigMapKey(s.Key)) > 0 || (s.CredentialID > 0) == (s.ExistingSecret != "") || s.CredentialID < 0 {
			return fmt.Errorf("请选择一个有效凭据或已有Secret，键名不能为空")
		}
		if s.ExistingSecret != "" && !validName(s.ExistingSecret) {
			return fmt.Errorf("Secret名称无效")
		}
		if s.CredentialID > 0 && (s.CredentialKey == "" || len(s.CredentialKey) > 128) {
			return fmt.Errorf("请选择凭据中的字段")
		}
		if s.Group != "" {
			if len(s.Group) > 30 || len(validation.IsDNS1123Label(s.Group)) > 0 {
				return fmt.Errorf("Secret组标识无效")
			}
			if old, exists := groups[s.Group]; exists && (old.CredentialID != s.CredentialID || old.ExistingSecret != s.ExistingSecret) {
				return fmt.Errorf("同一Secret组必须引用同一凭据或已有Secret")
			}
			groups[s.Group] = s
			key := s.Group + "/" + s.Key
			if groupKeys[key] {
				return fmt.Errorf("同一Secret组的键不能重复")
			}
			groupKeys[key] = true
		}
		if (s.EnvName != "") == (s.MountPath != "") {
			return fmt.Errorf("Secret键请选择环境变量注入或文件挂载其中一种")
		}
		if s.EnvName != "" {
			if !envKey.MatchString(s.EnvName) || envs[s.EnvName] {
				return fmt.Errorf("环境变量名称无效或重复")
			}
			envs[s.EnvName] = true
		} else if !checkPath(s.MountPath) {
			return fmt.Errorf("密码文件挂载路径冲突")
		}
	}
	routes := map[string]bool{}
	for _, r := range s.Routes {
		if !checkName(r.Name) {
			return fmt.Errorf("路由标识无效或重复")
		}
		if err := validateRoute(r); err != nil {
			return err
		}
		key := r.Host + "\x00" + r.Path
		if routes[key] {
			return fmt.Errorf("同一域名和路径不能重复配置路由")
		}
		routes[key] = true
	}
	for _, v := range s.Volumes {
		if !checkName(v.Name) || !checkPath(v.MountPath) {
			return fmt.Errorf("存储名称或挂载路径冲突")
		}
		if v.ExistingClaim != "" {
			if !validName(v.ExistingClaim) || v.Capacity != "" || v.StorageClass != "" {
				return fmt.Errorf("已有PVC只能引用，不能同时修改容量和StorageClass")
			}
		} else {
			q, err := resource.ParseQuantity(v.Capacity)
			if err == nil && q.Cmp(resource.MustParse("1Mi")) < 0 {
				return fmt.Errorf("PVC 容量至少为 1Mi，例如 1Gi；2 表示 2 字节，不是 2Gi")
			}
			if err != nil || q.Sign() <= 0 || (v.StorageClass != "" && !validName(v.StorageClass)) || (v.AccessMode != "ReadWriteOnce" && v.AccessMode != "ReadOnlyMany" && v.AccessMode != "ReadWriteMany" && v.AccessMode != "ReadWriteOncePod") {
				return fmt.Errorf("新建PVC需填写有效容量和访问模式；StorageClass留空使用集群默认类")
			}
		}
	}
	return nil
}
