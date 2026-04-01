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

// DingTalkConfig holds DingTalk channel configuration
type DingTalkConfig struct {
	Webhook string `json:"webhook"`
	Keyword string `json:"keyword"`
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
	AtAll bool `json:"isAtAll"`
}

// DingTalkNotifier implements Notifier for DingTalk
type DingTalkNotifier struct {
	webhookURL string
	client     *http.Client
}

func NewDingTalkNotifier(webhook string) *DingTalkNotifier {
	return &DingTalkNotifier{
		webhookURL: webhook,
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

// SendAlert sends an alert notification to DingTalk
func (n *DingTalkNotifier) SendAlert(alert AlertMessage) error {
	if n.webhookURL == "" {
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

	msg := DingTalkMessage{
		MsgType: "markdown",
		Markdown: DingTalkMarkdown{
			Title: title,
			Text:  text,
		},
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	resp, err := n.client.Post(n.webhookURL, "application/json", bytes.NewReader(body))
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

// SendBatchWarnings sends a batch of warnings to DingTalk with full details
func (n *DingTalkNotifier) SendBatchWarnings(items []BatchWarningItem, alertTime time.Time) error {
	if n.webhookURL == "" || len(items) == 0 {
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

	msg := DingTalkMessage{
		MsgType: "markdown",
		Markdown: DingTalkMarkdown{
			Title: title,
			Text:  sb.String(),
		},
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	resp, err := n.client.Post(n.webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("DingTalk batch response: %s", string(respBody))
	return nil
}
