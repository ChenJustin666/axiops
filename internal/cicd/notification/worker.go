package notification

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/ChenJustin666/axiops/internal/shared/secretbox"
)

type Worker struct {
	repo        *Repository
	box         *secretbox.Box
	client      *http.Client
	maxAttempts int
	lease       time.Duration
}

func NewWorker(repo *Repository, box *secretbox.Box, maxAttempts int, lease time.Duration) *Worker {
	if maxAttempts <= 0 {
		maxAttempts = 5
	}
	if lease <= 0 {
		lease = time.Minute
	}
	return &Worker{repo: repo, box: box, client: &http.Client{Timeout: 15 * time.Second}, maxAttempts: maxAttempts, lease: lease}
}
func (w *Worker) ProcessOne(ctx context.Context, workerID string) (bool, error) {
	item, err := w.repo.Claim(ctx, workerID, w.lease)
	if err != nil || item == nil {
		return false, err
	}
	subscription, credential, err := w.repo.DeliveryTarget(ctx, item.SubscriptionID)
	if err == nil && (credential == nil || (credential.ProjectID != 0 && credential.ProjectID != item.ProjectID)) {
		err = fmt.Errorf("notification credential is unavailable")
	}
	var message Message
	if err == nil {
		err = json.Unmarshal([]byte(item.Payload), &message)
	}
	var plaintext string
	if err == nil {
		plaintext, err = w.box.Decrypt(credential.Data)
	}
	if err == nil {
		err = w.deliver(ctx, subscription, plaintext, message)
	}
	if err != nil {
		_ = w.repo.Fail(ctx, item.ID, item.Attempts, w.maxAttempts, truncate(err.Error(), 1024))
		return true, err
	}
	return true, w.repo.Complete(ctx, item.ID)
}
func (w *Worker) deliver(ctx context.Context, s Subscription, secret string, message Message) error {
	if s.Channel == ChannelEmail {
		return sendEmail(secret, s.Recipient, message)
	}
	var config struct {
		URL string `json:"url"`
	}
	if json.Unmarshal([]byte(secret), &config) != nil || strings.TrimSpace(config.URL) == "" || !strings.HasPrefix(config.URL, "https://") {
		return fmt.Errorf("invalid notification webhook credential")
	}
	payload := webhookPayload(s.Channel, message)
	data, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, config.URL, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := w.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("notification endpoint returned %d", resp.StatusCode)
	}
	return nil
}
func webhookPayload(channel string, m Message) interface{} {
	text := m.Title + "\n" + m.Content
	if m.URL != "" {
		text += "\n" + m.URL
	}
	switch channel {
	case ChannelFeishu:
		return map[string]interface{}{"msg_type": "text", "content": map[string]string{"text": text}}
	case ChannelDingTalk:
		return map[string]interface{}{"msgtype": "text", "text": map[string]string{"content": text}}
	case ChannelWeCom:
		return map[string]interface{}{"msgtype": "text", "text": map[string]interface{}{"content": text}}
	default:
		return m
	}
}
func sendEmail(secret, recipient string, m Message) error {
	var c struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		Username string `json:"username"`
		Password string `json:"password"`
		From     string `json:"from"`
	}
	if json.Unmarshal([]byte(secret), &c) != nil || c.Host == "" || c.Port <= 0 || c.Username == "" || c.From == "" || recipient == "" {
		return fmt.Errorf("invalid smtp credential")
	}
	addr := c.Host + ":" + strconv.Itoa(c.Port)
	auth := smtp.PlainAuth("", c.Username, c.Password, c.Host)
	body := "To: " + recipient + "\r\nFrom: " + c.From + "\r\nSubject: " + m.Title + "\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" + m.Content + "\r\n" + m.URL
	_ = tls.VersionTLS12
	return smtp.SendMail(addr, auth, c.From, []string{recipient}, []byte(body))
}
func truncate(v string, n int) string {
	if len(v) > n {
		return v[:n]
	}
	return v
}
