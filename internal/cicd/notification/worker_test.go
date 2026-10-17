package notification

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ChenJustin666/axiops/internal/shared/secretbox"
)

func TestWebhookPayloadShapes(t *testing.T) {
	message := Message{EventType: "workflow", Title: "Build failed", Content: "compile error", URL: "https://axiops.example/runs/1"}
	for _, test := range []struct {
		channel string
		key     string
		value   string
	}{
		{ChannelFeishu, "msg_type", "text"},
		{ChannelDingTalk, "msgtype", "text"},
		{ChannelWeCom, "msgtype", "text"},
	} {
		data, err := json.Marshal(webhookPayload(test.channel, message))
		if err != nil || !strings.Contains(string(data), `"`+test.key+`":"`+test.value+`"`) || !strings.Contains(string(data), "Build failed") {
			t.Fatalf("unexpected %s payload: %s err=%v", test.channel, data, err)
		}
	}
}

func TestWorkerDeliverUsesHTTPSAndDoesNotExposeCredential(t *testing.T) {
	var received []byte
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	box, err := secretbox.New("test-key")
	if err != nil {
		t.Fatal(err)
	}
	secret, err := box.Encrypt(`{"url":"` + server.URL + `","sign":"private-value"}`)
	if err != nil {
		t.Fatal(err)
	}
	worker := NewWorker(nil, box, 3, 0)
	worker.client = server.Client()
	worker.client.Transport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // httptest certificate only.
	if err := worker.deliver(context.Background(), Subscription{Channel: ChannelFeishu}, mustDecrypt(t, box, secret), Message{Title: "ok", Content: "done"}); err != nil {
		t.Fatalf("deliver: %v", err)
	}
	if strings.Contains(string(received), "private-value") {
		t.Fatal("notification credential leaked into payload")
	}
	if err := worker.deliver(context.Background(), Subscription{Channel: ChannelFeishu}, `{"url":"http://insecure.example"}`, Message{}); err == nil {
		t.Fatal("expected insecure notification URL to be rejected")
	}
}

func mustDecrypt(t *testing.T, box *secretbox.Box, value string) string {
	t.Helper()
	plaintext, err := box.Decrypt(value)
	if err != nil {
		t.Fatal(err)
	}
	return plaintext
}

func TestValidCredentialType(t *testing.T) {
	if !validCredentialType(ChannelEmail, "smtp") || validCredentialType(ChannelEmail, "notification_webhook") {
		t.Fatal("smtp credential validation is incorrect")
	}
	if !validCredentialType(ChannelFeishu, "notification_webhook") || validCredentialType(ChannelFeishu, "smtp") {
		t.Fatal("webhook credential validation is incorrect")
	}
}
