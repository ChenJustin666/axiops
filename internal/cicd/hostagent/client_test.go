package hostagent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestClientDownloadVerifiesSHA256AndLimit(t *testing.T) {
	payload := []byte("immutable host artifact")
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(payload) }))
	defer server.Close()
	directory := t.TempDir()
	client := &Client{config: ClientConfig{MaxArtifactBytes: 1024}, http: server.Client()}
	sum := sha256.Sum256(payload)
	target := filepath.Join(directory, "artifact")
	if err := client.download(context.Background(), server.URL, hex.EncodeToString(sum[:]), target); err != nil {
		t.Fatalf("download: %v", err)
	}
	if data, err := os.ReadFile(target); err != nil || string(data) != string(payload) {
		t.Fatalf("unexpected artifact data: %q err=%v", data, err)
	}
	if err := client.download(context.Background(), server.URL, strings.Repeat("0", 64), target+"-bad"); err == nil || !strings.Contains(err.Error(), "mismatch") {
		t.Fatalf("expected checksum mismatch, got %v", err)
	}
	client.config.MaxArtifactBytes = 3
	if err := client.download(context.Background(), server.URL, hex.EncodeToString(sum[:]), target+"-large"); err == nil || !strings.Contains(err.Error(), "size limit") {
		t.Fatalf("expected size limit error, got %v", err)
	}
}

func TestClientExecutesHealthAndRollback(t *testing.T) {
	payload := []byte("artifact")
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(payload) }))
	defer server.Close()
	sum := sha256.Sum256(payload)
	directory := t.TempDir()
	client := &Client{config: ClientConfig{WorkDir: directory, MaxArtifactBytes: 1024}, http: server.Client()}
	deploy, health, rollback := platformScripts()
	command := Command{ID: 7, ArtifactURL: server.URL, ArtifactSHA256: hex.EncodeToString(sum[:]), DeployScript: deploy, HealthScript: health, RollbackScript: rollback, TimeoutSeconds: 20}
	buffer := &boundedBuffer{limit: 1 << 20}
	status, err := client.execute(context.Background(), command, buffer)
	if status != StatusRolledBack || err == nil {
		t.Fatalf("expected rollback after health failure, status=%s err=%v output=%s", status, err, buffer.String())
	}
	if _, err := os.Stat(filepath.Join(directory, "releases", "7", "rolled-back")); err != nil {
		t.Fatalf("rollback marker missing: %v", err)
	}
}

func platformScripts() (deploy, health, rollback string) {
	if runtime.GOOS == "windows" {
		return `Set-Content -Path (Join-Path $env:AXIOPS_RELEASE_DIR 'deployed') -Value 'ok'`, `exit 1`, `Set-Content -Path (Join-Path $env:AXIOPS_RELEASE_DIR 'rolled-back') -Value 'ok'`
	}
	return `touch "$AXIOPS_RELEASE_DIR/deployed"`, `exit 1`, `touch "$AXIOPS_RELEASE_DIR/rolled-back"`
}

func TestBoundedBufferKeepsTail(t *testing.T) {
	buffer := &boundedBuffer{limit: 5}
	_, _ = fmt.Fprint(buffer, "1234567")
	if buffer.String() != "34567" {
		t.Fatalf("unexpected bounded log %q", buffer.String())
	}
}
