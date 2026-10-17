package gitprovider

// REST-backed providers share the same repository and commit API adapter.
// GitLab remains separate because its webhook and diff payloads differ.

func newGitHubProvider(cfg *Config) (Provider, error) {
	return newRESTProvider(cfg, "github"), nil
}

func newGiteeProvider(cfg *Config) (Provider, error) {
	return newRESTProvider(cfg, "gitee"), nil
}

func newGiteaProvider(cfg *Config) (Provider, error) {
	return newRESTProvider(cfg, "gitea"), nil
}
