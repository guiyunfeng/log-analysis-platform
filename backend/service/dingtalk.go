package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DingTalkConfig holds DingTalk channel configuration
type DingTalkConfig struct {
	Webhook      string   `json:"webhook"`
	Keyword      string   `json:"keyword"`       // legacy single keyword
	Keywords     []string `json:"keywords"`      // multiple keywords
	SecurityType string   `json:"security_type"` // "keyword", "sign", "ip_whitelist"
	SignSecret   string   `json:"sign_secret"`   // HMAC-SHA256 secret for "sign" mode
	AtMobiles    []string `json:"at_mobiles"`    // phone numbers to @ in messages
	AtAll        bool     `json:"at_all"`
}

// DingTalkMessage represents a DingTalk webhook message
type DingTalkMessage struct {
	MsgType  string           `json:"msgtype"`
	Markdown DingTalkMarkdown `json:"markdown"`
	At       *DingTalkAt      `json:"at,omitempty"`
}

type DingTalkMarkdown struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

type DingTalkAt struct {
	AtMobiles []string `json:"atMobiles,omitempty"`
	AtAll     bool     `json:"isAtAll"`
}

// DingTalkNotifier implements Notifier for DingTalk
type DingTalkNotifier struct {
	cfg    DingTalkConfig
	client *http.Client
}

func NewDingTalkNotifier(webhook string) *DingTalkNotifier {
	return &DingTalkNotifier{
		cfg:    DingTalkConfig{Webhook: webhook},
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// NewDingTalkNotifierWithConfig creates a notifier from a full config
func NewDingTalkNotifierWithConfig(cfg DingTalkConfig) *DingTalkNotifier {
	return &DingTalkNotifier{
		cfg:    cfg,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// sign generates the timestamp and signature for DingTalk "sign" security mode.
func (n *DingTalkNotifier) sign() (timestamp string, sig string) {
	ts := time.Now().UnixMilli()
	timestamp = fmt.Sprintf("%d", ts)
	stringToSign := fmt.Sprintf("%s\n%s", timestamp, n.cfg.SignSecret)
	h := hmac.New(sha256.New, []byte(n.cfg.SignSecret))
	h.Write([]byte(stringToSign))
	sig = url.QueryEscape(base64.StdEncoding.EncodeToString(h.Sum(nil)))
	return
}

// buildURL returns the webhook URL, appending sign parameters when needed.
func (n *DingTalkNotifier) buildURL() string {
	if n.cfg.SecurityType != "sign" || n.cfg.SignSecret == "" {
		return n.cfg.Webhook
	}
	timestamp, sig := n.sign()
	return fmt.Sprintf("%s&timestamp=%s&sign=%s", n.cfg.Webhook, timestamp, sig)
}

// ensureKeyword makes sure the message text contains at least one required keyword.
func (n *DingTalkNotifier) ensureKeyword(text string) string {
	if n.cfg.SecurityType != "keyword" {
		return text
	}
	// Collect all configured keywords
	keywords := n.cfg.Keywords
	if len(keywords) == 0 && n.cfg.Keyword != "" {
		keywords = []string{n.cfg.Keyword}
	}
	if len(keywords) == 0 {
		return text
	}
	// If text already contains a keyword, return as-is
	for _, kw := range keywords {
		if kw != "" && strings.Contains(text, kw) {
			return text
		}
	}
	// Prepend the first keyword
	return keywords[0] + " " + text
}

// buildAt constructs an @mention object from config.
func (n *DingTalkNotifier) buildAt() *DingTalkAt {
	if len(n.cfg.AtMobiles) == 0 && !n.cfg.AtAll {
		return nil
	}
	return &DingTalkAt{
		AtMobiles: n.cfg.AtMobiles,
		AtAll:     n.cfg.AtAll,
	}
}

// post sends a DingTalkMessage to the webhook.
func (n *DingTalkNotifier) post(msg DingTalkMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}
	resp, err := n.client.Post(n.buildURL(), "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("DingTalk response: %s", string(respBody))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("dingtalk API error: status=%d body=%s", resp.StatusCode, string(respBody))
	}
	return nil
}

// SendAlert sends an alert notification to DingTalk
func (n *DingTalkNotifier) SendAlert(alert AlertMessage) error {
	if n.cfg.Webhook == "" {
		log.Println("DingTalk webhook not configured, skipping notification")
		return nil
	}
	if alert.Severity == "noise" {
		return nil
	}

	icon := "⚠️"
	levelStr := "WARNING"
	if alert.Severity == "critical" {
		icon = "🔴"
		levelStr = "CRITICAL"
	}

	title := fmt.Sprintf("%s [%s] 服务异常告警", icon, levelStr)

	comparisonLine := ""
	if alert.Comparison != "" {
		comparisonLine = fmt.Sprintf("**环比:** %s  \n", alert.Comparison)
	}

	sampleLine := ""
	if alert.SampleContent != "" {
		truncated := alert.SampleContent
		if len(truncated) > 200 {
			truncated = truncated[:200] + "..."
		}
		sampleLine = fmt.Sprintf("**示例报错:** %s  \n", truncated)
	}

	text := fmt.Sprintf(`%s [%s] 服务异常告警  
━━━━━━━━━━━━━━━━━━━━━  
**项目:** %s  
**服务:** %s  
**调用点:** %s  
**机器:** %s  
━━━━━━━━━━━━━━━━━━━━━  
**过去5分钟错误:** %d 次  
%s%s━━━━━━━━━━━━━━━━━━━━━  
⏰ %s`,
		icon, levelStr,
		alert.Project, alert.Service, alert.CallerFile, alert.Job,
		alert.ErrorCount,
		comparisonLine,
		sampleLine,
		alert.AlertTime.Format("2006-01-02 15:04:05"),
	)

	text = n.ensureKeyword(text)

	msg := DingTalkMessage{
		MsgType: "markdown",
		Markdown: DingTalkMarkdown{
			Title: title,
			Text:  text,
		},
		At: n.buildAt(),
	}

	return n.post(msg)
}

// SendBatchWarnings sends a batch of warnings to DingTalk with full details
func (n *DingTalkNotifier) SendBatchWarnings(items []BatchWarningItem, alertTime time.Time) error {
	if n.cfg.Webhook == "" || len(items) == 0 {
		return nil
	}

	title := "⚠️ [WARNING] 批量服务告警"
	var lines []string
	lines = append(lines, title, "━━━━━━━━━━━━━━━━━━━━━")
	for i, item := range items {
		if i >= 10 {
			lines = append(lines, fmt.Sprintf("... 共 %d 条告警", len(items)))
			break
		}
		lines = append(lines, fmt.Sprintf("🔸 **项目:** %s", item.Project))
		lines = append(lines, fmt.Sprintf("   **服务:** %s", item.Service))
		if item.CallerFile != "" {
			lines = append(lines, fmt.Sprintf("   **调用点:** %s", item.CallerFile))
		}
		if item.Job != "" {
			lines = append(lines, fmt.Sprintf("   **机器:** %s", item.Job))
		}
		lines = append(lines, fmt.Sprintf("   **过去5分钟错误:** %d 次 (%s)", item.ErrorCount, item.Comparison))
		if item.SampleContent != "" {
			sample := item.SampleContent
			if len(sample) > 100 {
				sample = sample[:100] + "..."
			}
			lines = append(lines, fmt.Sprintf("   **示例:** %s", sample))
		}
		lines = append(lines, "━━━━━━━━━━━━━━━━━━━━━")
	}
	lines = append(lines, fmt.Sprintf("⏰ %s", alertTime.Format("2006-01-02 15:04:05")))

	var sb strings.Builder
	for _, l := range lines {
		sb.WriteString(l)
		sb.WriteString("  \n")
	}

	text := n.ensureKeyword(sb.String())

	msg := DingTalkMessage{
		MsgType: "markdown",
		Markdown: DingTalkMarkdown{
			Title: title,
			Text:  text,
		},
		At: n.buildAt(),
	}

	return n.post(msg)
}

