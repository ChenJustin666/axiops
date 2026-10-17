package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCredentialEncryptionKeyUsesConfiguredValueWhenEnvironmentIsUnset(t *testing.T) {
	t.Setenv("AXIOPS_CREDENTIAL_ENCRYPTION_KEY", "")
	t.Setenv("AXIOPS_CREDENTIAL_ENCRYPTION_KEY_FILE", filepath.Join(t.TempDir(), "missing"))
	if got := CredentialEncryptionKey(" configured-key "); got != "configured-key" {
		t.Fatalf("unexpected configured key %q", got)
	}
}

func TestCredentialEncryptionKeyPrefersEnvironment(t *testing.T) {
	t.Setenv("AXIOPS_CREDENTIAL_ENCRYPTION_KEY", " runtime-key ")
	if got := CredentialEncryptionKey("configured-key"); got != "runtime-key" {
		t.Fatalf("unexpected runtime key %q", got)
	}
}

func TestCredentialEncryptionKeyReadsStableLocalFile(t *testing.T) {
	t.Setenv("AXIOPS_CREDENTIAL_ENCRYPTION_KEY", "")
	path := filepath.Join(t.TempDir(), "credential.key")
	if err := os.WriteFile(path, []byte(" local-file-key \n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AXIOPS_CREDENTIAL_ENCRYPTION_KEY_FILE", path)
	if got := CredentialEncryptionKey(""); got != "local-file-key" {
		t.Fatalf("unexpected file key %q", got)
	}
}

func TestCredentialEncryptionKeyConfiguredValuePrecedesFile(t *testing.T) {
	t.Setenv("AXIOPS_CREDENTIAL_ENCRYPTION_KEY", "")
	path := filepath.Join(t.TempDir(), "credential.key")
	if err := os.WriteFile(path, []byte("file-key"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AXIOPS_CREDENTIAL_ENCRYPTION_KEY_FILE", path)
	if got := CredentialEncryptionKey("configured-key"); got != "configured-key" {
		t.Fatalf("unexpected configured key %q", got)
	}
}
