package gitprovider

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGitLabWebhookRejectsInvalidSecret(t *testing.T) {
	provider, err := newGitLabProvider(&Config{WebhookSecret: "expected"})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{}`))
	req.Header.Set("X-Gitlab-Token", "wrong")
	req.Header.Set("X-Gitlab-Event", "Push Hook")
	if _, err := provider.ParseWebhookEvent(req); err != ErrWebhookSignature {
		t.Fatalf("got %v want ErrWebhookSignature", err)
	}
}

func TestGitLabWebhookParsesPush(t *testing.T) {
	provider, _ := newGitLabProvider(&Config{WebhookSecret: "expected"})
	payload := `{"ref":"refs/heads/main","after":"abcdef","project":{"name":"api","namespace":"team","http_url":"https://git.example/team/api.git"}}`
	req := httptest.NewRequest("POST", "/", strings.NewReader(payload))
	req.Header.Set("X-Gitlab-Token", "expected")
	req.Header.Set("X-Gitlab-Event", "Push Hook")
	event, err := provider.ParseWebhookEvent(req)
	if err != nil || event.EventType != "push" || event.Branch != "main" || event.After != "abcdef" {
		t.Fatalf("unexpected event=%+v err=%v", event, err)
	}
}
