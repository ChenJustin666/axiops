package build

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type staticRegistryAuthStore struct{ auth *RegistryAuth }

func (s staticRegistryAuthStore) Load(context.Context, int64, int64) (*RegistryAuth, error) {
	return s.auth, nil
}

func TestOutputVerifierStreamsAndVerifiesHTTPSArtifact(t *testing.T) {
	content := []byte("verified artifact")
	sum := sha256.Sum256(content)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(content)
	}))
	defer server.Close()

	verifier := &DefaultOutputVerifier{httpClient: server.Client(), maxArtifactBytes: 1024}
	result, err := verifier.Verify(context.Background(), BuildRun{}, ProviderState{
		ArtifactURL: server.URL, ArtifactSHA256: hex.EncodeToString(sum[:]), ArtifactSize: int64(len(content)),
	})
	if err != nil || result.ArtifactStatus != VerificationVerified || result.Failed() {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestOutputVerifierRejectsArtifactChecksumMismatch(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("different"))
	}))
	defer server.Close()

	verifier := &DefaultOutputVerifier{httpClient: server.Client(), maxArtifactBytes: 1024}
	result, err := verifier.Verify(context.Background(), BuildRun{}, ProviderState{
		ArtifactURL: server.URL, ArtifactSHA256: strings.Repeat("a", 64), ArtifactSize: 9,
	})
	if err != nil || result.ArtifactStatus != VerificationFailed || !result.Failed() {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestOutputVerifierChecksRegistryManifestDigest(t *testing.T) {
	digest := "sha256:" + strings.Repeat("b", 64)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodHead || request.URL.Path != "/v2/team/api/manifests/v1" {
			t.Fatalf("unexpected registry request %s %s", request.Method, request.URL.Path)
		}
		w.Header().Set("Docker-Content-Digest", digest)
	}))
	defer server.Close()
	registry := strings.TrimPrefix(server.URL, "https://")
	verifier := &DefaultOutputVerifier{
		httpClient: server.Client(), registryStore: staticRegistryAuthStore{auth: &RegistryAuth{BaseURL: server.URL}},
		verifyImage: true, maxArtifactBytes: 1024,
	}
	for _, basePath := range []string{"", "/team", "/original-namespace"} {
		verifier.registryStore = staticRegistryAuthStore{auth: &RegistryAuth{BaseURL: server.URL + basePath, Namespaces: []string{"team"}}}
		result, err := verifier.Verify(context.Background(), BuildRun{ProjectID: 2, ServiceID: 3, ImageURL: registry + "/team/api:v1"}, ProviderState{ImageDigest: digest})
		if err != nil || result.ImageStatus != VerificationVerified || result.Failed() {
			t.Fatalf("base path=%s result=%+v err=%v", basePath, result, err)
		}
		result, err = verifier.Verify(context.Background(), BuildRun{ImageURL: registry + "/unconfigured/api:v1"}, ProviderState{ImageDigest: digest})
		if err != nil || result.ImageStatus != VerificationFailed {
			t.Fatalf("unconfigured namespace was not rejected before network access: %+v %v", result, err)
		}
	}
}

func TestParseImageReferenceRequiresExplicitRegistry(t *testing.T) {
	registry, repository, reference, err := parseImageReference("harbor.example.com/team/api:v3")
	if err != nil || registry != "harbor.example.com" || repository != "team/api" || reference != "v3" {
		t.Fatalf("registry=%q repository=%q reference=%q err=%v", registry, repository, reference, err)
	}
	if _, _, _, err := parseImageReference("team/api:v3"); err == nil {
		t.Fatal("implicit registry image was accepted")
	}
}
