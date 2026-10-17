package build

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"sort"
	"strings"
)

// BuildFingerprint is the stable identity of a build input. It is safe to
// persist in logs and database keys because it contains no credentials.
func BuildFingerprint(serviceID int64, repositoryURL, branch, commit string, config *BuildConfig, customCommand string, skipTests bool) string {
	// Global builder changes are not frozen yet; never reuse across an unknown
	// toolchain. A fully specified builder is required for automatic reuse.
	if config == nil || config.BuildImage == "" || config.BuildCommand == "" {
		return ""
	}
	// A moving branch or short SHA cannot safely identify a reusable output.
	if !regexp.MustCompile(`^[a-fA-F0-9]{40}([a-fA-F0-9]{24})?$`).MatchString(commit) {
		return ""
	}
	type fingerprintConfig struct {
		TemplateType      string   `json:"template_type"`
		BuildImage        string   `json:"build_image"`
		BuildCommand      string   `json:"build_command"`
		DockerfilePath    string   `json:"dockerfile_path"`
		DockerfileSource  string   `json:"dockerfile_source"`
		DockerfileContent string   `json:"dockerfile_content"`
		BuildContext      string   `json:"build_context"`
		ImageRepo         string   `json:"image_repo"`
		CacheEnabled      bool     `json:"cache_enabled"`
		CacheDir          string   `json:"cache_dir"`
		CacheStrategy     string   `json:"cache_strategy"`
		CachePVCName      string   `json:"cache_pvc_name,omitempty"`
		TimeoutSeconds    int      `json:"timeout_seconds"`
		EnvVars           []EnvVar `json:"env_vars"`
		ResourceCPU       string   `json:"resource_cpu"`
		ResourceMemory    string   `json:"resource_memory"`
		Provider          string   `json:"provider"`
		BuildMode         string   `json:"build_mode"`
	}
	input := struct {
		ServiceID     int64             `json:"service_id"`
		RepositoryURL string            `json:"repository_url"`
		Branch        string            `json:"branch"`
		Commit        string            `json:"commit"`
		Config        fingerprintConfig `json:"config"`
		CustomCommand string            `json:"custom_command,omitempty"`
		SkipTests     bool              `json:"skip_tests,omitempty"`
	}{ServiceID: serviceID, RepositoryURL: strings.TrimSpace(repositoryURL), Branch: strings.TrimSpace(branch), Commit: strings.TrimSpace(commit), CustomCommand: strings.TrimSpace(customCommand), SkipTests: skipTests}
	if config != nil {
		input.Config = fingerprintConfig{
			TemplateType: config.TemplateType, BuildImage: config.BuildImage, BuildCommand: config.BuildCommand,
			DockerfileSource: config.DockerfileSource, DockerfileContent: config.DockerfileContent,
			DockerfilePath: config.DockerfilePath, BuildContext: config.BuildContext, ImageRepo: config.ImageRepo,
			CacheEnabled: config.CacheEnabled, CacheDir: config.CacheDir, TimeoutSeconds: config.TimeoutSeconds,
			CacheStrategy: config.CacheStrategy, CachePVCName: config.CachePVCName,
			EnvVars: config.ParseEnvVars(), ResourceCPU: config.ResourceCPU, ResourceMemory: config.ResourceMemory,
			Provider: config.Provider, BuildMode: config.BuildMode,
		}
		sort.Slice(input.Config.EnvVars, func(i, j int) bool {
			return input.Config.EnvVars[i].Key < input.Config.EnvVars[j].Key
		})
	}
	data, _ := json.Marshal(input)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func (c *BuildConfig) ParseEnvVars() []EnvVar {
	if c == nil || c.EnvVars == nil || strings.TrimSpace(*c.EnvVars) == "" {
		return nil
	}
	var values []EnvVar
	if json.Unmarshal([]byte(*c.EnvVars), &values) != nil {
		return nil
	}
	return values
}
