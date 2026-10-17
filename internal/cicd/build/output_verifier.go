package build

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	VerificationUnverified    = "unverified"
	VerificationVerified      = "verified"
	VerificationFailed        = "failed"
	VerificationSkipped       = "skipped"
	VerificationNotApplicable = "not_applicable"
)

type OutputVerification struct {
	ImageStatus    string
	ArtifactStatus string
	Error          string
	VerifiedAt     time.Time
}

func (v OutputVerification) Failed() bool {
	return v.ImageStatus == VerificationFailed || v.ArtifactStatus == VerificationFailed
}

type OutputVerifier interface {
	Verify(context.Context, BuildRun, ProviderState) (OutputVerification, error)
}

type CredentialDecryptor interface {
	Decrypt(string) (string, error)
}

type RegistryAuth struct {
	BaseURL    string
	Namespaces []string
	Username   string
	Password   string
	Token      string
}

type RegistryAuthStore interface {
	Load(context.Context, int64, int64) (*RegistryAuth, error)
}

type SQLRegistryAuthStore struct {
	db        *sql.DB
	decryptor CredentialDecryptor
}

func NewSQLRegistryAuthStore(db *sql.DB, decryptor CredentialDecryptor) *SQLRegistryAuthStore {
	return &SQLRegistryAuthStore{db: db, decryptor: decryptor}
}

func (s *SQLRegistryAuthStore) Load(ctx context.Context, projectID, serviceID int64) (*RegistryAuth, error) {
	var baseURL, credentialType string
	var namespaces string
	var credentialID int64
	var encrypted sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT registry.url,COALESCE(credential.type,''),credential.data,registry.credential_id,COALESCE(registry.namespaces,JSON_ARRAY())
		FROM app_service service
		JOIN image_registry registry ON registry.id=service.registry_id AND registry.status='active'
		LEFT JOIN credential ON credential.id=registry.credential_id AND credential.project_id IN (0,service.project_id)
		WHERE service.id=? AND service.project_id=? AND registry.project_id IN (0,?)`, serviceID, projectID, projectID).
		Scan(&baseURL, &credentialType, &encrypted, &credentialID, &namespaces)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	auth := &RegistryAuth{BaseURL: strings.TrimSpace(baseURL)}
	if err := json.Unmarshal([]byte(namespaces), &auth.Namespaces); err != nil {
		return nil, fmt.Errorf("image registry namespaces are invalid")
	}
	if !encrypted.Valid || encrypted.String == "" {
		if credentialID > 0 {
			return nil, fmt.Errorf("image registry credential is missing or outside the project scope")
		}
		return auth, nil
	}
	if credentialType != "harbor_auth" && credentialType != "registry_auth" {
		return nil, fmt.Errorf("unsupported image registry credential type %q", credentialType)
	}
	if s.decryptor == nil {
		return nil, fmt.Errorf("image registry credential decryptor is not configured")
	}
	plaintext, err := s.decryptor.Decrypt(encrypted.String)
	if err != nil {
		return nil, fmt.Errorf("decrypt image registry credential: authentication failed")
	}
	var credential struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Token    string `json:"token"`
	}
	if err := json.Unmarshal([]byte(plaintext), &credential); err != nil {
		return nil, fmt.Errorf("image registry credential is invalid")
	}
	auth.Username = strings.TrimSpace(credential.Username)
	auth.Password = credential.Password
	auth.Token = strings.TrimSpace(credential.Token)
	if auth.Username == "" && auth.Token == "" {
		return nil, fmt.Errorf("image registry credential requires username or token")
	}
	return auth, nil
}

type DefaultOutputVerifier struct {
	httpClient       *http.Client
	registryStore    RegistryAuthStore
	verifyImage      bool
	maxArtifactBytes int64
}

func NewOutputVerifierFromEnvironment(db *sql.DB, decryptor CredentialDecryptor) (*DefaultOutputVerifier, error) {
	verifyImage := false
	if raw := strings.TrimSpace(os.Getenv("AXIOPS_BUILD_VERIFY_IMAGE")); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, fmt.Errorf("AXIOPS_BUILD_VERIFY_IMAGE must be true or false")
		}
		verifyImage = value
	}
	maxBytes := int64(2 << 30)
	if raw := strings.TrimSpace(os.Getenv("AXIOPS_BUILD_VERIFY_MAX_ARTIFACT_BYTES")); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value <= 0 {
			return nil, fmt.Errorf("AXIOPS_BUILD_VERIFY_MAX_ARTIFACT_BYTES must be a positive integer")
		}
		maxBytes = value
	}
	client := &http.Client{
		Timeout: 30 * time.Minute,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 || req.URL.Scheme != "https" {
				return fmt.Errorf("artifact redirect is not allowed")
			}
			return nil
		},
	}
	return &DefaultOutputVerifier{httpClient: client, registryStore: NewSQLRegistryAuthStore(db, decryptor), verifyImage: verifyImage, maxArtifactBytes: maxBytes}, nil
}

func (v *DefaultOutputVerifier) Verify(ctx context.Context, run BuildRun, state ProviderState) (OutputVerification, error) {
	result := OutputVerification{ImageStatus: VerificationNotApplicable, ArtifactStatus: VerificationNotApplicable, VerifiedAt: time.Now().UTC()}
	if state.ArtifactURL != "" {
		status, reason, err := v.verifyArtifact(ctx, state)
		if err != nil {
			return result, err
		}
		result.ArtifactStatus = status
		if status == VerificationFailed {
			result.Error = reason
		}
	}
	if state.ImageDigest != "" || run.ImageURL != "" {
		if !v.verifyImage {
			result.ImageStatus = VerificationSkipped
		} else {
			status, reason, err := v.verifyImageManifest(ctx, run, state)
			if err != nil {
				return result, err
			}
			result.ImageStatus = status
			if status == VerificationFailed {
				if result.Error == "" {
					result.Error = reason
				} else {
					result.Error += "; " + reason
				}
			}
		}
	}
	return result, nil
}

func (v *DefaultOutputVerifier) verifyArtifact(ctx context.Context, state ProviderState) (string, string, error) {
	parsed, err := url.Parse(state.ArtifactURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return VerificationFailed, "artifact verification requires a credential-free HTTPS URL", nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return VerificationFailed, "artifact URL is invalid", nil
	}
	response, err := v.httpClient.Do(req)
	if err != nil {
		return VerificationUnverified, "", fmt.Errorf("download artifact for verification: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return VerificationUnverified, "", fmt.Errorf("download artifact for verification: HTTP %d", response.StatusCode)
	}
	if response.ContentLength > v.maxArtifactBytes {
		return VerificationFailed, "artifact exceeds the configured verification size limit", nil
	}
	hash := sha256.New()
	written, err := io.Copy(hash, io.LimitReader(response.Body, v.maxArtifactBytes+1))
	if err != nil {
		return VerificationUnverified, "", fmt.Errorf("stream artifact for verification: %w", err)
	}
	if written > v.maxArtifactBytes {
		return VerificationFailed, "artifact exceeds the configured verification size limit", nil
	}
	actualSHA := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(actualSHA, state.ArtifactSHA256) {
		return VerificationFailed, "artifact SHA-256 does not match the build result", nil
	}
	if state.ArtifactSize > 0 && written != state.ArtifactSize {
		return VerificationFailed, "artifact size does not match the build result", nil
	}
	return VerificationVerified, "", nil
}

func (v *DefaultOutputVerifier) verifyImageManifest(ctx context.Context, run BuildRun, state ProviderState) (string, string, error) {
	if state.ImageDigest == "" {
		return VerificationFailed, "image digest is missing", nil
	}
	registry, repository, reference, err := parseImageReference(run.ImageURL)
	if err != nil {
		return VerificationFailed, err.Error(), nil
	}
	auth, err := v.registryStore.Load(ctx, run.ProjectID, run.ServiceID)
	if err != nil {
		return VerificationUnverified, "", err
	}
	baseURL := "https://" + registry
	if auth != nil && strings.TrimSpace(auth.BaseURL) != "" {
		baseURL = strings.TrimRight(strings.TrimSpace(auth.BaseURL), "/")
		if !strings.Contains(baseURL, "://") {
			baseURL = "https://" + baseURL
		}
	}
	parsedBase, err := url.Parse(baseURL)
	if err != nil || parsedBase.Host == "" || (parsedBase.Scheme != "https" && parsedBase.Scheme != "http") {
		return VerificationFailed, "image registry URL is invalid", nil
	}
	if !strings.EqualFold(parsedBase.Host, registry) {
		return VerificationFailed, "image registry does not match the configured service registry", nil
	}
	if auth != nil {
		if err := validateRegistryImage(run.ImageURL, auth); err != nil {
			return VerificationFailed, err.Error(), nil
		}
	}
	manifestURL := parsedBase.Scheme + "://" + parsedBase.Host + "/v2/" + repository + "/manifests/" + url.PathEscape(reference)
	response, err := v.registryRequest(ctx, manifestURL, repository, auth)
	if err != nil {
		return VerificationUnverified, "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return VerificationFailed, fmt.Sprintf("image registry returned HTTP %d", response.StatusCode), nil
	}
	actual := strings.TrimSpace(response.Header.Get("Docker-Content-Digest"))
	if actual == "" || !strings.EqualFold(actual, state.ImageDigest) {
		return VerificationFailed, "registry manifest digest does not match the build result", nil
	}
	return VerificationVerified, "", nil
}

func (v *DefaultOutputVerifier) registryRequest(ctx context.Context, manifestURL, repository string, auth *RegistryAuth) (*http.Response, error) {
	request := func(token string) (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodHead, manifestURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.v2+json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		} else if auth != nil && auth.Username != "" {
			req.SetBasicAuth(auth.Username, auth.Password)
		}
		return v.httpClient.Do(req)
	}
	token := ""
	if auth != nil {
		token = auth.Token
	}
	response, err := request(token)
	if err != nil || response.StatusCode != http.StatusUnauthorized || token != "" {
		return response, err
	}
	challenge := response.Header.Get("WWW-Authenticate")
	response.Body.Close()
	realm, service := bearerChallenge(challenge)
	if realm == "" {
		return response, nil
	}
	tokenURL, err := url.Parse(realm)
	if err != nil || tokenURL.Scheme != "https" || tokenURL.Host == "" {
		return nil, fmt.Errorf("registry bearer authentication realm is invalid")
	}
	query := tokenURL.Query()
	query.Set("service", service)
	query.Set("scope", "repository:"+repository+":pull")
	tokenURL.RawQuery = query.Encode()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL.String(), nil)
	if auth != nil && auth.Username != "" {
		req.SetBasicAuth(auth.Username, auth.Password)
	}
	tokenResponse, err := v.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request registry access token: %w", err)
	}
	defer tokenResponse.Body.Close()
	if tokenResponse.StatusCode < 200 || tokenResponse.StatusCode >= 300 {
		return nil, fmt.Errorf("request registry access token: HTTP %d", tokenResponse.StatusCode)
	}
	var payload struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}
	if json.NewDecoder(io.LimitReader(tokenResponse.Body, 1<<20)).Decode(&payload) != nil {
		return nil, fmt.Errorf("registry access token response is invalid")
	}
	if payload.Token == "" {
		payload.Token = payload.AccessToken
	}
	if payload.Token == "" {
		return nil, fmt.Errorf("registry access token is empty")
	}
	return request(payload.Token)
}

func parseImageReference(value string) (string, string, string, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.Contains(value, "://") {
		return "", "", "", fmt.Errorf("image reference is invalid")
	}
	parts := strings.SplitN(value, "/", 2)
	if len(parts) != 2 || (!strings.Contains(parts[0], ".") && !strings.Contains(parts[0], ":") && parts[0] != "localhost") {
		return "", "", "", fmt.Errorf("image reference must include an explicit registry")
	}
	registry, remainder := parts[0], parts[1]
	reference := "latest"
	if at := strings.LastIndex(remainder, "@"); at >= 0 {
		reference, remainder = remainder[at+1:], remainder[:at]
	} else if colon := strings.LastIndex(remainder, ":"); colon > strings.LastIndex(remainder, "/") {
		reference, remainder = remainder[colon+1:], remainder[:colon]
	}
	if registry == "" || remainder == "" || reference == "" || strings.Contains(remainder, "..") {
		return "", "", "", fmt.Errorf("image reference is invalid")
	}
	return registry, remainder, reference, nil
}

func bearerChallenge(value string) (string, string) {
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(value)), "bearer ") {
		return "", ""
	}
	params := strings.TrimSpace(value[strings.Index(value, " ")+1:])
	values := map[string]string{}
	for _, item := range strings.Split(params, ",") {
		pair := strings.SplitN(strings.TrimSpace(item), "=", 2)
		if len(pair) == 2 {
			values[strings.ToLower(pair[0])] = strings.Trim(pair[1], `"`)
		}
	}
	return values["realm"], values["service"]
}
