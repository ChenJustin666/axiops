package webhook

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/ChenJustin666/axiops/internal/cicd/build"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"github.com/ChenJustin666/axiops/internal/shared/response"
	"github.com/ChenJustin666/axiops/internal/shared/secretbox"
	"github.com/ChenJustin666/axiops/pkg/gitprovider"
)

// BuildTrigger is deliberately limited to the build queue adapter. Webhooks
// never call the HTTP build endpoint and therefore cannot bypass its durable queue.
type BuildTrigger interface {
	TriggerFromWebhook(context.Context, *build.WebhookBuildRequest) (int64, error)
}

type Handler struct {
	db           *sql.DB
	buildTrigger BuildTrigger
	box          *secretbox.Box
}

func NewHandler(db *sql.DB, buildTrigger BuildTrigger, encryptionKey string) *Handler {
	box, _ := secretbox.New(encryptionKey)
	return &Handler{db: db, buildTrigger: buildTrigger, box: box}
}

// HandleGitWebhook authenticates, deduplicates and queues Git push events.
func (h *Handler) HandleGitWebhook(w http.ResponseWriter, r *http.Request) {
	sourceID := parseID(r.URL.Query().Get("source_id"))
	if sourceID <= 0 {
		response.ErrorWithMsg(w, errcode.CodeWebhookInvalid, "source_id is required")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 4<<20))
	if err != nil {
		response.ErrorWithMsg(w, errcode.CodeWebhookInvalid, "invalid webhook body")
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(body))

	var sourceType, sourceURL string
	var webhookCredentialID sql.NullInt64
	err = h.db.QueryRowContext(r.Context(), `SELECT type, url, webhook_credential_id FROM code_source WHERE id=? AND status='active'`, sourceID).
		Scan(&sourceType, &sourceURL, &webhookCredentialID)
	if err != nil {
		response.ErrorWithMsg(w, errcode.CodeCodeSourceNotFound, "code source not found")
		return
	}
	secret, err := h.webhookSecret(r.Context(), webhookCredentialID)
	if err != nil {
		response.ErrorWithMsg(w, errcode.CodeWebhookInvalid, "webhook credential is invalid")
		return
	}
	provider, err := gitprovider.New(&gitprovider.Config{Type: sourceType, BaseURL: sourceURL, WebhookSecret: secret})
	if err != nil {
		response.ErrorWithMsg(w, errcode.CodeInternalError, "unsupported git provider")
		return
	}
	event, err := provider.ParseWebhookEvent(r)
	if err != nil {
		response.ErrorWithMsg(w, errcode.CodeWebhookInvalid, "webhook signature verification failed")
		return
	}
	if event.EventType != "push" && event.EventType != "tag_push" {
		response.Success(w, map[string]string{"status": "ignored", "reason": "event type not handled"})
		return
	}

	deliveryID := strings.TrimSpace(r.Header.Get("X-Gitlab-Event-UUID"))
	if deliveryID == "" {
		deliveryID = strings.TrimSpace(r.Header.Get("X-GitHub-Delivery"))
	}
	if deliveryID == "" {
		sum := sha256.Sum256(body)
		deliveryID = hex.EncodeToString(sum[:])
	}
	claimed, existing, err := h.claimDelivery(r.Context(), sourceID, deliveryID, event)
	if err != nil {
		response.ErrorWithMsg(w, errcode.CodeInternalError, "webhook delivery claim failed")
		return
	}
	if !claimed && existing > 0 {
		response.Success(w, map[string]interface{}{"status": "duplicate", "build_id": existing})
		return
	}

	repoPath := strings.Trim(event.Namespace+"/"+event.RepoName, "/")
	var serviceID int64
	var rules sql.NullString
	err = h.db.QueryRowContext(r.Context(), `SELECT s.id, bc.branch_rules FROM app_service s LEFT JOIN build_config bc ON bc.service_id=s.id WHERE s.code_source_id=? AND (s.repo_url LIKE CONCAT('%', ?, '%') OR s.repo_url LIKE CONCAT('%', ?)) AND s.status='active' LIMIT 1`, sourceID, repoPath, event.RepoName).Scan(&serviceID, &rules)
	if errors.Is(err, sql.ErrNoRows) {
		h.recordDelivery(r.Context(), sourceID, deliveryID, event, "ignored", 0, "no matching service")
		response.Success(w, map[string]string{"status": "ignored", "reason": "no matching service"})
		return
	}
	if err != nil {
		response.ErrorWithMsg(w, errcode.CodeInternalError, "service lookup failed")
		return
	}
	if !matchBranchRule(rules.String, event.Branch) {
		h.recordDelivery(r.Context(), sourceID, deliveryID, event, "ignored", 0, "branch not matched")
		response.Success(w, map[string]string{"status": "ignored", "reason": "branch not matched"})
		return
	}
	commit, author, email, message := event.After, event.UserName, event.UserEmail, ""
	if len(event.Commits) > 0 {
		c := event.Commits[len(event.Commits)-1]
		commit, author, email, message = c.ID, c.Author, c.Email, c.Message
	}
	buildID, err := h.buildTrigger.TriggerFromWebhook(r.Context(), &build.WebhookBuildRequest{ServiceID: serviceID, SourceID: sourceID, Branch: event.Branch, CommitID: commit, Author: author, AuthorEmail: email, Message: message, DeliveryID: deliveryID})
	if err != nil {
		h.recordDelivery(r.Context(), sourceID, deliveryID, event, "failed", 0, err.Error())
		response.ErrorWithMsg(w, errcode.CodeInternalError, "build trigger failed")
		return
	}
	if err := h.recordDelivery(r.Context(), sourceID, deliveryID, event, "triggered", buildID, ""); err != nil {
		log.Printf("record webhook delivery: %v", err)
	}
	response.Success(w, map[string]interface{}{"status": "triggered", "build_id": buildID})
}

func (h *Handler) webhookSecret(ctx context.Context, ref sql.NullInt64) (string, error) {
	if !ref.Valid || h.box == nil {
		return "", errors.New("missing webhook credential")
	}
	var typ, data string
	if err := h.db.QueryRowContext(ctx, "SELECT type,data FROM credential WHERE id=?", ref.Int64).Scan(&typ, &data); err != nil || typ != "git_webhook_secret" {
		return "", errors.New("invalid webhook credential")
	}
	plain, err := h.box.Decrypt(data)
	if err != nil {
		return "", err
	}
	var v struct {
		Secret string `json:"secret"`
	}
	if json.Unmarshal([]byte(plain), &v) != nil || strings.TrimSpace(v.Secret) == "" {
		return "", errors.New("invalid webhook secret")
	}
	return v.Secret, nil
}

func (h *Handler) recordDelivery(ctx context.Context, sourceID int64, deliveryID string, event *gitprovider.WebhookEvent, status string, buildID int64, reason string) error {
	_, err := h.db.ExecContext(ctx, `INSERT INTO webhook_delivery(source_id,delivery_id,provider,event_type,branch,commit_id,status,build_id,error_message,payload_hash) VALUES(?,?,?,?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE status=VALUES(status),build_id=VALUES(build_id),error_message=VALUES(error_message)`, sourceID, deliveryID, event.Source, event.EventType, event.Branch, event.After, status, buildID, reason, hashPayload(event.RawPayload))
	return err
}
func (h *Handler) claimDelivery(ctx context.Context, sourceID int64, deliveryID string, event *gitprovider.WebhookEvent) (bool, int64, error) {
	result, err := h.db.ExecContext(ctx, `INSERT IGNORE INTO webhook_delivery(source_id,delivery_id,provider,event_type,branch,commit_id,status,payload_hash) VALUES(?,?,?,?,?,?,'processing',?)`, sourceID, deliveryID, event.Source, event.EventType, event.Branch, event.After, hashPayload(event.RawPayload))
	if err != nil {
		return false, 0, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, 0, err
	}
	if affected == 1 {
		return true, 0, nil
	}
	var buildID int64
	err = h.db.QueryRowContext(ctx, `SELECT build_id FROM webhook_delivery WHERE source_id=? AND delivery_id=?`, sourceID, deliveryID).Scan(&buildID)
	return false, buildID, err
}
func hashPayload(v []byte) string { s := sha256.Sum256(v); return hex.EncodeToString(s[:]) }
func parseID(v string) int64 {
	id, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	if err != nil || id <= 0 {
		return 0
	}
	return id
}
func matchBranchRule(raw, branch string) bool {
	if strings.TrimSpace(raw) == "" || raw == "null" || raw == "[]" {
		return true
	}
	var rules build.BranchRules
	if json.Unmarshal([]byte(raw), &rules) != nil {
		return false
	}
	for _, p := range rules.Excludes {
		if matchPattern(p, branch) {
			return false
		}
	}
	if len(rules.Includes) == 0 {
		return true
	}
	for _, p := range rules.Includes {
		if matchPattern(p, branch) {
			return true
		}
	}
	return false
}
func matchPattern(pattern, value string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return false
	}
	matched, err := path.Match(pattern, value)
	return err == nil && matched
}
