package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// PhoneAlertConfig holds phone-call alert channel configuration.
type PhoneAlertConfig struct {
	Provider         string   `json:"provider"`           // "aliyun", "tencent", "webhook"
	AccessKeyID      string   `json:"access_key_id"`      // reserved for future direct SDK integration
	SecretKey        string   `json:"secret_key"`         // reserved for future direct SDK integration
	TemplateID       string   `json:"template_id"`
	CalledShowNumber string   `json:"called_show_number"` // caller-id shown on phone
	PhoneNumbers     []string `json:"phone_numbers"`      // numbers to call
	WebhookURL       string   `json:"webhook_url"`        // for "webhook" provider or proxy URL
}

// PhoneAlertNotifier implements the Notifier interface via phone/voice alerts.
// For aliyun/tencent providers it uses a webhook-proxy mode (POST to WebhookURL)
// so that a real SDK integration can be added later without changing the interface.
type PhoneAlertNotifier struct {
	cfg    PhoneAlertConfig
	client *http.Client
}

// NewPhoneAlertNotifier creates a PhoneAlertNotifier from the given config.
func NewPhoneAlertNotifier(cfg PhoneAlertConfig) *PhoneAlertNotifier {
	return &PhoneAlertNotifier{
		cfg:    cfg,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// SendAlert sends a voice alert for a single alert message.
// Only Critical severity triggers a phone call to avoid alert fatigue.
func (n *PhoneAlertNotifier) SendAlert(alert AlertMessage) error {
	if alert.Severity != "critical" {
		// Phone calls are reserved for critical alerts only.
		return nil
	}
	if len(n.cfg.PhoneNumbers) == 0 {
		log.Println("PhoneAlert: no phone numbers configured, skipping")
		return nil
	}

	levelStr := "CRITICAL"
	payload := map[string]interface{}{
		"severity":     levelStr,
		"project":      alert.Project,
		"service":      alert.Service,
		"error_count":  alert.ErrorCount,
		"phone_numbers": n.cfg.PhoneNumbers,
		"template_id":  n.cfg.TemplateID,
		"show_number":  n.cfg.CalledShowNumber,
	}

	return n.dispatch(payload)
}

// SendBatchWarnings sends a batch warning via phone (no-op for non-critical).
func (n *PhoneAlertNotifier) SendBatchWarnings(items []BatchWarningItem, alertTime time.Time) error {
	// Phone alerts are only for critical; batch warnings are warning-level.
	return nil
}

// dispatch routes the call request to the configured provider.
func (n *PhoneAlertNotifier) dispatch(payload map[string]interface{}) error {
	switch strings.ToLower(n.cfg.Provider) {
	case "aliyun":
		return n.sendViaProxy("aliyun", payload)
	case "tencent":
		return n.sendViaProxy("tencent", payload)
	case "webhook":
		return n.sendWebhook(payload)
	default:
		log.Printf("PhoneAlert: unknown provider %q, falling back to webhook", n.cfg.Provider)
		return n.sendWebhook(payload)
	}
}

// sendViaProxy posts the payload to WebhookURL with a provider hint so an
// external gateway or proxy can handle the actual SDK call.
//
// TODO: Replace this with direct Alibaba Cloud / Tencent Cloud SDK calls once
// the corresponding API credentials are available in the deployment environment.
func (n *PhoneAlertNotifier) sendViaProxy(provider string, payload map[string]interface{}) error {
	if n.cfg.WebhookURL == "" {
		log.Printf("PhoneAlert (%s): WebhookURL not configured, skipping", provider)
		return nil
	}
	payload["provider"] = provider
	return n.sendWebhook(payload)
}

// sendWebhook POSTs the payload as JSON to the configured WebhookURL.
func (n *PhoneAlertNotifier) sendWebhook(payload map[string]interface{}) error {
	if n.cfg.WebhookURL == "" {
		log.Println("PhoneAlert: WebhookURL not configured, skipping")
		return nil
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("PhoneAlert marshal: %w", err)
	}

	resp, err := n.client.Post(n.cfg.WebhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("PhoneAlert request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("PhoneAlert webhook response: %s", string(respBody))

	if resp.StatusCode >= 300 {
		return fmt.Errorf("PhoneAlert webhook error: status=%d body=%s", resp.StatusCode, string(respBody))
	}
	return nil
}
