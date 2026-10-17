package build

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// DockerConfigForImage validates registry/namespace ownership before returning
// runtime-only credential material. Never persist this in execution snapshots.
func (auth *RegistryAuth) DockerConfigForImage(imageURL string) (string, error) {
	return registryDockerConfig(imageURL, auth)
}

// Only the matching registry receives credentials. This runtime value must
// never be persisted in build snapshots or serialized into Job environment values.
func registryDockerConfig(imageURL string, auth *RegistryAuth) (string, error) {
	if auth == nil {
		return "", nil // Explicitly unbound registries may allow anonymous pushes.
	}
	if err := validateRegistryImage(imageURL, auth); err != nil {
		return "", err
	}
	host, _, _, _ := parseImageReference(imageURL)
	entry := map[string]string{}
	if auth.Username != "" && auth.Password != "" {
		entry["auth"] = base64.StdEncoding.EncodeToString([]byte(auth.Username + ":" + auth.Password))
	} else if auth.Token != "" {
		entry["identitytoken"] = auth.Token
	} else if auth.Username != "" || auth.Password != "" {
		return "", fmt.Errorf("image registry credential requires both username and password")
	} else {
		return "", nil
	}
	data, err := json.Marshal(map[string]any{"auths": map[string]any{host: entry}})
	return string(data), err
}

func validateRegistryImage(imageURL string, auth *RegistryAuth) error {
	host, repository, _, err := parseImageReference(imageURL)
	if err != nil {
		return err
	}
	base := strings.TrimSpace(auth.BaseURL)
	if !strings.Contains(base, "://") {
		base = "https://" + base
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return fmt.Errorf("image registry URL is invalid")
	}
	prefix := strings.Trim(parsed.Path, "/")
	if !strings.EqualFold(parsed.Host, host) {
		return fmt.Errorf("build image does not match the configured service registry")
	}
	if prefix == "" && len(auth.Namespaces) == 0 {
		return nil // Legacy host-wide registry.
	}
	for _, namespace := range append([]string{prefix}, auth.Namespaces...) {
		if namespace != "" && strings.HasPrefix(repository, namespace+"/") {
			return nil
		}
	}
	return fmt.Errorf("build image is outside the configured registry namespaces")
}
