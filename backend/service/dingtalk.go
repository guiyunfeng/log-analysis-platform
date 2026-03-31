package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"log-analysis-platform/config"
)

// DingTalkMessage represents a DingTalk webhook message
type DingTalkMessage struct {
	MsgType  string              `json:"msgtype"`
	Markdown DingTalkMarkdown    `json:"markdown"`
	At       *DingTalkAt         `json:"at,omitempty"`
}

type DingTalkMarkdown struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

type DingTalkAt struct {
	AtAll bool `json:"isAtAll"`
}

// DingTalkService handles DingTalk notifications
type DingTalkService struct {
	webhookURL string
	client     *http.Client
}

var DefaultDingTalkService *DingTalkService

func InitDingTalk() {
	DefaultDingTalkService = &DingTalkService{
		webhookURL: config.GlobalConfig.DingTalkWebhook,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SendAlert sends an alert notification to DingTalk
func (s *DingTalkService) SendAlert(severity, project, service, callerFile, job string, errorCount int, comparison, sampleContent string, alertTime time.Time) error {
	if s.webhookURL == "" {
		log.Println("DingTalk webhook not configured, skipping notification")
		return nil
	}

	icon := "⚠️"
	levelStr := "WARNING"
	switch severity {
	case "critical":
		icon = "🔴"
		levelStr = "CRITICAL"
	case "warning":
		icon = "⚠️"
		levelStr = "WARNING"
	case "noise":
		return nil // Don't send noise alerts
	}

	title := fmt.Sprintf("%s [%s] 服务异常告警", icon, levelStr)

	comparisonLine := ""
	if comparison != "" {
		comparisonLine = fmt.Sprintf("环比上一小时: %s  \n", comparison)
	}

	sampleLine := ""
	if sampleContent != "" {
		truncated := sampleContent
		if len(truncated) > 200 {
			truncated = truncated[:200] + "..."
		}
		sampleLine = fmt.Sprintf("━━━━━━━━━━━━━━━━━━━━━  \n**示例报错:**  \n%s  \n", truncated)
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
		project, service, callerFile, job,
		errorCount,
		comparisonLine,
		sampleLine,
		alertTime.Format("2006-01-02 15:04:05"),
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

	resp, err := s.client.Post(s.webhookURL, "application/json", bytes.NewReader(body))
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

// SendBatchWarnings sends a batch of warnings to DingTalk
func (s *DingTalkService) SendBatchWarnings(items []BatchWarningItem, alertTime time.Time) error {
	if s.webhookURL == "" || len(items) == 0 {
		return nil
	}

	title := "⚠️ [WARNING] 批量服务告警"
	lines := []string{title, "━━━━━━━━━━━━━━━━━━━━━"}
	for i, item := range items {
		if i >= 10 {
			lines = append(lines, fmt.Sprintf("... 共 %d 条告警", len(items)))
			break
		}
		lines = append(lines, fmt.Sprintf("**%s/%s** - %d次错误 (%s)", item.Service, item.CallerFile, item.ErrorCount, item.Comparison))
	}
	lines = append(lines, "━━━━━━━━━━━━━━━━━━━━━")
	lines = append(lines, fmt.Sprintf("⏰ %s", alertTime.Format("2006-01-02 15:04:05")))

	text := ""
	for _, l := range lines {
		text += l + "  \n"
	}

	msg := DingTalkMessage{
		MsgType: "markdown",
		Markdown: DingTalkMarkdown{
			Title: title,
			Text:  text,
		},
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	resp, err := s.client.Post(s.webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

type BatchWarningItem struct {
	Service    string
	CallerFile string
	ErrorCount int
	Comparison string
}
