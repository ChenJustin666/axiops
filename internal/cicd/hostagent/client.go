package hostagent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const agentVersion = "0.1.0"

type ClientConfig struct {
	ServerURL         string
	RegistrationToken string
	AgentTokenFile    string
	AssetRef          string
	Name              string
	WorkDir           string
	CACertFile        string
	AllowInsecureHTTP bool
	PollInterval      time.Duration
	MaxArtifactBytes  int64
}

type Client struct {
	config ClientConfig
	http   *http.Client
	token  string
}

type apiResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func NewClient(config ClientConfig) (*Client, error) {
	config.ServerURL = strings.TrimRight(strings.TrimSpace(config.ServerURL), "/")
	parsed, err := url.Parse(config.ServerURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && !(config.AllowInsecureHTTP && parsed.Scheme == "http")) {
		return nil, fmt.Errorf("Agent server URL must use HTTPS (or explicitly allow HTTP for development)")
	}
	if config.AssetRef == "" || config.Name == "" || config.WorkDir == "" || config.AgentTokenFile == "" {
		return nil, fmt.Errorf("asset_ref, name, work_dir, and agent_token_file are required")
	}
	if config.PollInterval <= 0 {
		config.PollInterval = 5 * time.Second
	}
	if config.MaxArtifactBytes <= 0 {
		config.MaxArtifactBytes = 2 << 30
	}
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
	if config.CACertFile != "" {
		pem, err := os.ReadFile(config.CACertFile)
		if err != nil {
			return nil, fmt.Errorf("read Agent CA certificate: %w", err)
		}
		pool, err := x509.SystemCertPool()
		if err != nil || pool == nil {
			pool = x509.NewCertPool()
		}
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("Agent CA certificate is invalid")
		}
		tlsConfig.RootCAs = pool
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = tlsConfig
	return &Client{config: config, http: &http.Client{Transport: transport, Timeout: 30 * time.Second}}, nil
}

func (c *Client) Run(ctx context.Context) error {
	if err := os.MkdirAll(c.config.WorkDir, 0700); err != nil {
		return err
	}
	if err := c.loadOrRegister(ctx); err != nil {
		return err
	}
	for {
		// Heartbeat is explicit so an idle Agent remains observable even when
		// command polling is temporarily disabled by a network policy.
		if err := c.heartbeat(ctx); err != nil && ctx.Err() != nil {
			return ctx.Err()
		}
		command, err := c.poll(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
		} else if command != nil {
			c.executeAndReport(ctx, *command)
		}
		timer := time.NewTimer(c.config.PollInterval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (c *Client) heartbeat(ctx context.Context) error {
	return c.request(ctx, http.MethodPost, "/api/v1/agent/heartbeat", HeartbeatRequest{Version: agentVersion}, nil, true)
}

func (c *Client) loadOrRegister(ctx context.Context) error {
	data, err := os.ReadFile(c.config.AgentTokenFile)
	if err == nil && len(strings.TrimSpace(string(data))) == 64 {
		c.token = strings.TrimSpace(string(data))
		return nil
	}
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if len(strings.TrimSpace(c.config.RegistrationToken)) != 64 {
		return fmt.Errorf("registration token is required for first startup")
	}
	hostname, _ := os.Hostname()
	request := RegisterRequest{Token: c.config.RegistrationToken, AssetRef: c.config.AssetRef, Name: c.config.Name, Version: agentVersion, OS: runtime.GOOS, Arch: runtime.GOARCH, Metadata: map[string]interface{}{"hostname": hostname}}
	var result RegisterResponse
	if err := c.request(ctx, http.MethodPost, "/api/v1/agent/register", request, &result, false); err != nil {
		return err
	}
	if len(result.AgentToken) != 64 {
		return fmt.Errorf("server returned invalid Agent token")
	}
	if err := os.MkdirAll(filepath.Dir(c.config.AgentTokenFile), 0700); err != nil {
		return err
	}
	if err := os.WriteFile(c.config.AgentTokenFile, []byte(result.AgentToken), 0600); err != nil {
		return err
	}
	c.token = result.AgentToken
	return nil
}

func (c *Client) poll(ctx context.Context) (*Command, error) {
	var command *Command
	if err := c.request(ctx, http.MethodGet, "/api/v1/agent/commands/poll", nil, &command, true); err != nil {
		return nil, err
	}
	return command, nil
}

func (c *Client) report(ctx context.Context, commandID int64, request ReportRequest) error {
	return c.request(ctx, http.MethodPost, "/api/v1/agent/commands/"+strconv.FormatInt(commandID, 10)+"/report", request, nil, true)
}

func (c *Client) request(ctx context.Context, method, path string, body, target interface{}, authenticated bool) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.config.ServerURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if authenticated {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}
	var envelope apiResponse
	if json.Unmarshal(data, &envelope) != nil || resp.StatusCode < 200 || resp.StatusCode >= 300 || envelope.Code != 0 {
		return fmt.Errorf("Agent API %s failed: HTTP %d, code %d, message %s", path, resp.StatusCode, envelope.Code, envelope.Message)
	}
	if target != nil && len(envelope.Data) > 0 && string(envelope.Data) != "null" {
		return json.Unmarshal(envelope.Data, target)
	}
	return nil
}

func (c *Client) executeAndReport(parent context.Context, command Command) {
	logBuffer := &boundedBuffer{limit: 1 << 20}
	_ = c.report(parent, command.ID, ReportRequest{Status: StatusRunning})
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				_ = c.report(parent, command.ID, ReportRequest{Status: StatusRunning, LogText: logBuffer.String()})
			}
		}
	}()
	status, err := c.execute(parent, command, logBuffer)
	close(done)
	request := ReportRequest{Status: status, LogText: logBuffer.String()}
	if err != nil {
		request.ErrorMessage = err.Error()
	}
	_ = c.report(parent, command.ID, request)
}

func (c *Client) execute(parent context.Context, command Command, output io.Writer) (string, error) {
	if command.ID <= 0 || command.TimeoutSeconds < 1 || command.TimeoutSeconds > 86400 || !validSHA256(command.ArtifactSHA256) {
		return StatusFailed, fmt.Errorf("invalid host deployment command")
	}
	releaseDir := filepath.Join(c.config.WorkDir, "releases", strconv.FormatInt(command.ID, 10))
	if err := os.MkdirAll(releaseDir, 0700); err != nil {
		return StatusFailed, err
	}
	artifactPath := filepath.Join(releaseDir, "artifact")
	if err := c.download(parent, command.ArtifactURL, command.ArtifactSHA256, artifactPath); err != nil {
		return StatusFailed, err
	}
	ctx, cancel := context.WithTimeout(parent, time.Duration(command.TimeoutSeconds)*time.Second)
	defer cancel()
	env := commandEnvironment(releaseDir, artifactPath, command.ID)
	if err := runScript(ctx, releaseDir, command.DeployScript, env, output); err == nil {
		if strings.TrimSpace(command.HealthScript) != "" {
			err = runScript(ctx, releaseDir, command.HealthScript, env, output)
		}
		if err == nil {
			marker, _ := json.Marshal(map[string]interface{}{"command_id": command.ID, "release_dir": releaseDir, "activated_at": time.Now().UTC()})
			if writeErr := atomicWrite(filepath.Join(c.config.WorkDir, "current.json"), marker, 0600); writeErr != nil {
				return StatusFailed, writeErr
			}
			return StatusSuccess, nil
		}
	}
	deployErr := fmt.Errorf("deploy or health check failed")
	if ctx.Err() != nil {
		deployErr = ctx.Err()
	}
	if strings.TrimSpace(command.RollbackScript) != "" {
		rollbackCtx, rollbackCancel := context.WithTimeout(parent, minDuration(5*time.Minute, time.Duration(command.TimeoutSeconds)*time.Second))
		defer rollbackCancel()
		if err := runScript(rollbackCtx, releaseDir, command.RollbackScript, env, output); err == nil {
			return StatusRolledBack, deployErr
		}
	}
	return StatusFailed, deployErr
}

func (c *Client) download(ctx context.Context, rawURL, expectedSHA256, target string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return fmt.Errorf("artifact URL must use HTTPS")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("artifact endpoint returned %d", resp.StatusCode)
	}
	if resp.ContentLength > c.config.MaxArtifactBytes {
		return fmt.Errorf("artifact exceeds configured size limit")
	}
	temporary := target + ".part"
	file, err := os.OpenFile(temporary, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	hash := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(file, hash), io.LimitReader(resp.Body, c.config.MaxArtifactBytes+1))
	closeErr := file.Close()
	if copyErr != nil || closeErr != nil || written > c.config.MaxArtifactBytes {
		_ = os.Remove(temporary)
		return fmt.Errorf("download artifact failed or exceeded size limit")
	}
	if hex.EncodeToString(hash.Sum(nil)) != strings.ToLower(expectedSHA256) {
		_ = os.Remove(temporary)
		return fmt.Errorf("artifact SHA-256 mismatch")
	}
	if err := os.Rename(temporary, target); err != nil {
		_ = os.Remove(temporary)
		return err
	}
	return nil
}

func runScript(ctx context.Context, workDir, script string, env []string, output io.Writer) error {
	if strings.TrimSpace(script) == "" {
		return nil
	}
	var command *exec.Cmd
	if runtime.GOOS == "windows" {
		command = exec.CommandContext(ctx, "powershell.exe", "-NoLogo", "-NonInteractive", "-NoProfile", "-Command", script)
	} else {
		command = exec.CommandContext(ctx, "/bin/sh", "-eu", "-c", script)
	}
	command.Dir = workDir
	command.Env = env
	command.Stdout, command.Stderr = output, output
	return command.Run()
}

func commandEnvironment(releaseDir, artifactPath string, commandID int64) []string {
	allowed := []string{"PATH", "SystemRoot", "WINDIR", "COMSPEC", "TMP", "TEMP", "LANG", "LC_ALL"}
	result := make([]string, 0, len(allowed)+3)
	for _, key := range allowed {
		if value := os.Getenv(key); value != "" {
			result = append(result, key+"="+value)
		}
	}
	return append(result, "AXIOPS_RELEASE_DIR="+releaseDir, "AXIOPS_ARTIFACT_PATH="+artifactPath, "AXIOPS_COMMAND_ID="+strconv.FormatInt(commandID, 10))
}

func atomicWrite(path string, data []byte, mode os.FileMode) error {
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, data, mode); err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		_ = os.Remove(path)
	}
	return os.Rename(temporary, path)
}

func minDuration(left, right time.Duration) time.Duration {
	if left < right {
		return left
	}
	return right
}

type boundedBuffer struct {
	mu    sync.Mutex
	data  []byte
	limit int
}

func (b *boundedBuffer) Write(value []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.data = append(b.data, value...)
	if len(b.data) > b.limit {
		b.data = append([]byte(nil), b.data[len(b.data)-b.limit:]...)
	}
	return len(value), nil
}

func (b *boundedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(append([]byte(nil), b.data...))
}
