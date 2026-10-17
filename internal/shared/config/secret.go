package config

import (
	"os"
	"path/filepath"
	"strings"
)

const defaultCredentialEncryptionKeyFile = "local/credential-encryption.key"

// CredentialEncryptionKey returns the deployment secret when supplied. This
// keeps shared YAML configuration free of credential-encryption material while
// preserving an explicit YAML value for controlled local development.
func CredentialEncryptionKey(configured string) string {
	if value := strings.TrimSpace(os.Getenv("AXIOPS_CREDENTIAL_ENCRYPTION_KEY")); value != "" {
		return value
	}
	if value := strings.TrimSpace(configured); value != "" {
		return value
	}
	keyFile := strings.TrimSpace(os.Getenv("AXIOPS_CREDENTIAL_ENCRYPTION_KEY_FILE"))
	if keyFile == "" {
		keyFile = filepath.FromSlash(defaultCredentialEncryptionKeyFile)
	}
	data, err := os.ReadFile(keyFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
