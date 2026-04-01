package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"log-analysis-platform/config"
	"log-analysis-platform/model"

	"gorm.io/gorm"
)

// Notifier is the unified notification interface
type Notifier interface {
	SendAlert(alert AlertMessage) error
	SendBatchWarnings(items []BatchWarningItem, alertTime time.Time) error
}

// AlertMessage is the unified alert message structure
type AlertMessage struct {
	Severity      string
	Project       string
	Service       string
	CallerFile    string
	Job           string
	ErrorCount    int
	Comparison    string
	SampleContent string
	AlertTime     time.Time
}

// BatchWarningItem represents a single item in a batch warning
type BatchWarningItem struct {
	Project       string
	Service       string
	CallerFile    string
	Job           string
	ErrorCount    int
	Comparison    string
	SampleContent string
}

// maxSampleLen is the maximum length of a sample content shown in a single-alert message.
const maxSampleLen = 200

// maxBatchSampleLen is the maximum length of a sample content shown per item in a batch message.
const maxBatchSampleLen = 100

// NotifyService manages multiple notification channels
type NotifyService struct {
	db *gorm.DB
}

var DefaultNotifyService *NotifyService

// InitNotifyService initialises the notification service and auto-creates the
// DingTalk channel if DINGTALK_WEBHOOK env variable is set.
func InitNotifyService(db *gorm.DB) {
	DefaultNotifyService = &NotifyService{db: db}
	autoCreateDingTalkChannel(db)
}

func autoCreateDingTalkChannel(db *gorm.DB) {
	webhook := config.GlobalConfig.DingTalkWebhook
	if webhook == "" {
		return
	}
	var count int64
	db.Model(&model.NotifyChannel{}).Where("type = ?", "dingtalk").Count(&count)
	if count > 0 {
		return
	}
	cfg, err := json.Marshal(map[string]string{"webhook": webhook, "keyword": "告警"})
	if err != nil {
		log.Printf("Failed to marshal DingTalk config: %v", err)
		return
	}
	channel := model.NotifyChannel{
		Name:    "钉钉告警群",
		Type:    "dingtalk",
		Config:  string(cfg),
		Enabled: true,
	}
	db.Create(&channel)
	log.Println("Auto-created DingTalk notify channel from env DINGTALK_WEBHOOK")
}

func (s *NotifyService) loadChannels() []model.NotifyChannel {
	var channels []model.NotifyChannel
	s.db.Where("enabled = ?", true).Find(&channels)
	return channels
}

func (s *NotifyService) buildNotifier(ch model.NotifyChannel) Notifier {
	switch ch.Type {
	case "dingtalk":
		var cfg DingTalkConfig
		if err := json.Unmarshal([]byte(ch.Config), &cfg); err != nil {
			log.Printf("Failed to parse DingTalk config for channel %d: %v", ch.ID, err)
			return nil
		}
		return NewDingTalkNotifier(cfg.Webhook)
	case "wecom":
		var cfg WeComConfig
		if err := json.Unmarshal([]byte(ch.Config), &cfg); err != nil {
			log.Printf("Failed to parse WeCom config for channel %d: %v", ch.ID, err)
			return nil
		}
		return NewWeComNotifier(cfg.Webhook)
	case "email":
		var cfg EmailConfig
		if err := json.Unmarshal([]byte(ch.Config), &cfg); err != nil {
			log.Printf("Failed to parse Email config for channel %d: %v", ch.ID, err)
			return nil
		}
		return NewEmailNotifier(cfg)
	case "telegram":
		var cfg TelegramConfig
		if err := json.Unmarshal([]byte(ch.Config), &cfg); err != nil {
			log.Printf("Failed to parse Telegram config for channel %d: %v", ch.ID, err)
			return nil
		}
		return NewTelegramNotifier(cfg.BotToken, cfg.ChatID)
	case "feishu":
		var cfg FeishuConfig
		if err := json.Unmarshal([]byte(ch.Config), &cfg); err != nil {
			log.Printf("Failed to parse Feishu config for channel %d: %v", ch.ID, err)
			return nil
		}
		return NewFeishuNotifier(cfg.Webhook)
	}
	return nil
}

// SendAlert dispatches an alert to all enabled channels
func (s *NotifyService) SendAlert(alert AlertMessage) {
	channels := s.loadChannels()
	for _, ch := range channels {
		notifier := s.buildNotifier(ch)
		if notifier == nil {
			continue
		}
		if err := notifier.SendAlert(alert); err != nil {
			log.Printf("Send alert via %s channel %d error: %v", ch.Type, ch.ID, err)
		}
	}
}

// SendBatchWarnings dispatches batch warnings to all enabled channels
func (s *NotifyService) SendBatchWarnings(items []BatchWarningItem, alertTime time.Time) {
	if len(items) == 0 {
		return
	}
	channels := s.loadChannels()
	for _, ch := range channels {
		notifier := s.buildNotifier(ch)
		if notifier == nil {
			continue
		}
		if err := notifier.SendBatchWarnings(items, alertTime); err != nil {
			log.Printf("Send batch warnings via %s channel %d error: %v", ch.Type, ch.ID, err)
		}
	}
}

// TestChannel sends a test message to the given channel
func (s *NotifyService) TestChannel(ch model.NotifyChannel) error {
	notifier := s.buildNotifier(ch)
	if notifier == nil {
		return fmt.Errorf("unsupported channel type: %s", ch.Type)
	}
	alert := AlertMessage{
		Severity:      "warning",
		Project:       "test_project",
		Service:       "test_service",
		CallerFile:    "test/main.go",
		Job:           "test-machine",
		ErrorCount:    1,
		Comparison:    "↑ 100%",
		SampleContent: "这是一条测试告警消息",
		AlertTime:     time.Now(),
	}
	return notifier.SendAlert(alert)
}

// ─────────────────────────────────────────────────────────────────────────────
// WeCom (企业微信) Notifier
// ─────────────────────────────────────────────────────────────────────────────

type WeComConfig struct {
	Webhook string `json:"webhook"`
}

type WeComNotifier struct {
	webhookURL string
	client     *http.Client
}

func NewWeComNotifier(webhook string) *WeComNotifier {
	return &WeComNotifier{
		webhookURL: webhook,
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

func (n *WeComNotifier) sendMarkdown(title, content string) error {
	if n.webhookURL == "" {
		return nil
	}
	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"content": content,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := n.client.Post(n.webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("wecom request: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("WeCom response: %s", string(respBody))
	return nil
}

func (n *WeComNotifier) SendAlert(alert AlertMessage) error {
	icon := "⚠️"
	levelStr := "WARNING"
	if alert.Severity == "critical" {
		icon = "🔴"
		levelStr = "CRITICAL"
	}

	content := fmt.Sprintf(`%s **[%s] 服务异常告警**
> **项目:** %s
> **服务:** %s
> **调用点:** %s
> **机器:** %s
> **过去5分钟错误:** %d 次`,
		icon, levelStr,
		alert.Project, alert.Service, alert.CallerFile, alert.Job,
		alert.ErrorCount,
	)
	if alert.Comparison != "" {
		content += fmt.Sprintf("\n> **环比:** %s", alert.Comparison)
	}
	if alert.SampleContent != "" {
		sample := alert.SampleContent
		if len(sample) > maxSampleLen {
			sample = sample[:maxSampleLen] + "..."
		}
		content += fmt.Sprintf("\n> **示例:** %s", sample)
	}
	content += fmt.Sprintf("\n> ⏰ %s", alert.AlertTime.Format("2006-01-02 15:04:05"))

	return n.sendMarkdown(levelStr+" 服务异常告警", content)
}

func (n *WeComNotifier) SendBatchWarnings(items []BatchWarningItem, alertTime time.Time) error {
	if len(items) == 0 {
		return nil
	}
	var sb strings.Builder
	sb.WriteString("⚠️ **[WARNING] 批量服务告警**\n\n")
	for i, item := range items {
		if i >= 10 {
			sb.WriteString(fmt.Sprintf("... 共 %d 条告警\n", len(items)))
			break
		}
		sb.WriteString(fmt.Sprintf("> 🔸 **项目:** %s\n", item.Project))
		sb.WriteString(fmt.Sprintf("> **服务:** %s\n", item.Service))
		if item.CallerFile != "" {
			sb.WriteString(fmt.Sprintf("> **调用点:** %s\n", item.CallerFile))
		}
		if item.Job != "" {
			sb.WriteString(fmt.Sprintf("> **机器:** %s\n", item.Job))
		}
		sb.WriteString(fmt.Sprintf("> **过去5分钟错误:** %d 次 (%s)\n", item.ErrorCount, item.Comparison))
		if item.SampleContent != "" {
			sample := item.SampleContent
			if len(sample) > maxBatchSampleLen {
				sample = sample[:maxBatchSampleLen] + "..."
			}
			sb.WriteString(fmt.Sprintf("> **示例:** %s\n", sample))
		}
		sb.WriteString("\n")
	}
	sb.WriteString(fmt.Sprintf("> ⏰ %s", alertTime.Format("2006-01-02 15:04:05")))
	return n.sendMarkdown("批量服务告警", sb.String())
}

// ─────────────────────────────────────────────────────────────────────────────
// Email Notifier
// ─────────────────────────────────────────────────────────────────────────────

type EmailConfig struct {
	SMTPHost string   `json:"smtp_host"`
	SMTPPort int      `json:"smtp_port"`
	Username string   `json:"username"`
	Password string   `json:"password"`
	From     string   `json:"from"`
	To       []string `json:"to"`
	UseTLS   bool     `json:"use_tls"`
}

type EmailNotifier struct {
	cfg EmailConfig
}

func NewEmailNotifier(cfg EmailConfig) *EmailNotifier {
	return &EmailNotifier{cfg: cfg}
}

func (n *EmailNotifier) sendEmail(subject, body string) error {
	if n.cfg.SMTPHost == "" || len(n.cfg.To) == 0 {
		return nil
	}
	addr := fmt.Sprintf("%s:%d", n.cfg.SMTPHost, n.cfg.SMTPPort)
	auth := smtp.PlainAuth("", n.cfg.Username, n.cfg.Password, n.cfg.SMTPHost)

	headers := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n",
		n.cfg.From,
		strings.Join(n.cfg.To, ", "),
		subject,
	)
	msg := []byte(headers + body)

	return smtp.SendMail(addr, auth, n.cfg.From, n.cfg.To, msg)
}

func (n *EmailNotifier) SendAlert(alert AlertMessage) error {
	levelStr := "WARNING"
	if alert.Severity == "critical" {
		levelStr = "CRITICAL"
	}
	subject := fmt.Sprintf("[%s] 服务异常告警 - %s/%s", levelStr, alert.Project, alert.Service)
	body := fmt.Sprintf(`告警级别: %s
项目: %s
服务: %s
调用点: %s
机器: %s
过去5分钟错误: %d 次`,
		levelStr, alert.Project, alert.Service, alert.CallerFile, alert.Job, alert.ErrorCount,
	)
	if alert.Comparison != "" {
		body += fmt.Sprintf("\n环比: %s", alert.Comparison)
	}
	if alert.SampleContent != "" {
		sample := alert.SampleContent
		if len(sample) > maxSampleLen {
			sample = sample[:maxSampleLen] + "..."
		}
		body += fmt.Sprintf("\n示例报错: %s", sample)
	}
	body += fmt.Sprintf("\n\n告警时间: %s", alert.AlertTime.Format("2006-01-02 15:04:05"))
	return n.sendEmail(subject, body)
}

func (n *EmailNotifier) SendBatchWarnings(items []BatchWarningItem, alertTime time.Time) error {
	if len(items) == 0 {
		return nil
	}
	subject := fmt.Sprintf("[WARNING] 批量服务告警 (%d 条)", len(items))
	var sb strings.Builder
	sb.WriteString("批量服务告警\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━\n")
	for i, item := range items {
		if i >= 10 {
			sb.WriteString(fmt.Sprintf("... 共 %d 条告警\n", len(items)))
			break
		}
		sb.WriteString(fmt.Sprintf("项目: %s\n", item.Project))
		sb.WriteString(fmt.Sprintf("服务: %s\n", item.Service))
		if item.CallerFile != "" {
			sb.WriteString(fmt.Sprintf("调用点: %s\n", item.CallerFile))
		}
		if item.Job != "" {
			sb.WriteString(fmt.Sprintf("机器: %s\n", item.Job))
		}
		sb.WriteString(fmt.Sprintf("过去5分钟错误: %d 次 (%s)\n", item.ErrorCount, item.Comparison))
		if item.SampleContent != "" {
			sample := item.SampleContent
			if len(sample) > maxBatchSampleLen {
				sample = sample[:maxBatchSampleLen] + "..."
			}
			sb.WriteString(fmt.Sprintf("示例: %s\n", sample))
		}
		sb.WriteString("━━━━━━━━━━━━━━━━━━━━━\n")
	}
	sb.WriteString(fmt.Sprintf("⏰ %s", alertTime.Format("2006-01-02 15:04:05")))
	return n.sendEmail(subject, sb.String())
}

// ─────────────────────────────────────────────────────────────────────────────
// Telegram Notifier
// ─────────────────────────────────────────────────────────────────────────────

type TelegramConfig struct {
	BotToken string `json:"bot_token"`
	ChatID   string `json:"chat_id"`
}

type TelegramNotifier struct {
	botToken string
	chatID   string
	client   *http.Client
}

func NewTelegramNotifier(botToken, chatID string) *TelegramNotifier {
	return &TelegramNotifier{
		botToken: botToken,
		chatID:   chatID,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (n *TelegramNotifier) sendMessage(text string) error {
	if n.botToken == "" || n.chatID == "" {
		return nil
	}
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", n.botToken)
	payload := map[string]interface{}{
		"chat_id":    n.chatID,
		"text":       text,
		"parse_mode": "HTML",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := n.client.Post(apiURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("telegram request: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("Telegram response: %s", string(respBody))
	return nil
}

func (n *TelegramNotifier) SendAlert(alert AlertMessage) error {
	levelStr := "WARNING ⚠️"
	if alert.Severity == "critical" {
		levelStr = "CRITICAL 🔴"
	}
	text := fmt.Sprintf(`<b>[%s] 服务异常告警</b>
━━━━━━━━━━━━━━━━━━━━━
<b>项目:</b> %s
<b>服务:</b> %s
<b>调用点:</b> %s
<b>机器:</b> %s
<b>过去5分钟错误:</b> %d 次`,
		levelStr, alert.Project, alert.Service, alert.CallerFile, alert.Job, alert.ErrorCount,
	)
	if alert.Comparison != "" {
		text += fmt.Sprintf("\n<b>环比:</b> %s", alert.Comparison)
	}
	if alert.SampleContent != "" {
		sample := alert.SampleContent
		if len(sample) > maxSampleLen {
			sample = sample[:maxSampleLen] + "..."
		}
		text += fmt.Sprintf("\n<b>示例:</b> <code>%s</code>", sample)
	}
	text += fmt.Sprintf("\n━━━━━━━━━━━━━━━━━━━━━\n⏰ %s", alert.AlertTime.Format("2006-01-02 15:04:05"))
	return n.sendMessage(text)
}

func (n *TelegramNotifier) SendBatchWarnings(items []BatchWarningItem, alertTime time.Time) error {
	if len(items) == 0 {
		return nil
	}
	var sb strings.Builder
	sb.WriteString("<b>⚠️ [WARNING] 批量服务告警</b>\n━━━━━━━━━━━━━━━━━━━━━\n")
	for i, item := range items {
		if i >= 10 {
			sb.WriteString(fmt.Sprintf("... 共 %d 条告警\n", len(items)))
			break
		}
		sb.WriteString(fmt.Sprintf("🔸 <b>项目:</b> %s\n", item.Project))
		sb.WriteString(fmt.Sprintf("   <b>服务:</b> %s\n", item.Service))
		if item.CallerFile != "" {
			sb.WriteString(fmt.Sprintf("   <b>调用点:</b> %s\n", item.CallerFile))
		}
		if item.Job != "" {
			sb.WriteString(fmt.Sprintf("   <b>机器:</b> %s\n", item.Job))
		}
		sb.WriteString(fmt.Sprintf("   <b>过去5分钟错误:</b> %d 次 (%s)\n", item.ErrorCount, item.Comparison))
		if item.SampleContent != "" {
			sample := item.SampleContent
			if len(sample) > maxBatchSampleLen {
				sample = sample[:maxBatchSampleLen] + "..."
			}
			sb.WriteString(fmt.Sprintf("   <b>示例:</b> <code>%s</code>\n", sample))
		}
		sb.WriteString("━━━━━━━━━━━━━━━━━━━━━\n")
	}
	sb.WriteString(fmt.Sprintf("⏰ %s", alertTime.Format("2006-01-02 15:04:05")))
	return n.sendMessage(sb.String())
}

// ─────────────────────────────────────────────────────────────────────────────
// Feishu (飞书) Notifier
// ─────────────────────────────────────────────────────────────────────────────

type FeishuConfig struct {
	Webhook string `json:"webhook"`
}

type FeishuNotifier struct {
	webhookURL string
	client     *http.Client
}

func NewFeishuNotifier(webhook string) *FeishuNotifier {
	return &FeishuNotifier{
		webhookURL: webhook,
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

func (n *FeishuNotifier) sendText(content string) error {
	if n.webhookURL == "" {
		return nil
	}
	payload := map[string]interface{}{
		"msg_type": "text",
		"content": map[string]string{
			"text": content,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := n.client.Post(n.webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("feishu request: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("Feishu response: %s", string(respBody))
	return nil
}

func (n *FeishuNotifier) SendAlert(alert AlertMessage) error {
	levelStr := "WARNING ⚠️"
	if alert.Severity == "critical" {
		levelStr = "CRITICAL 🔴"
	}
	text := fmt.Sprintf("[%s] 服务异常告警\n━━━━━━━━━━━━━━━━━━━━━\n项目: %s\n服务: %s\n调用点: %s\n机器: %s\n过去5分钟错误: %d 次",
		levelStr, alert.Project, alert.Service, alert.CallerFile, alert.Job, alert.ErrorCount,
	)
	if alert.Comparison != "" {
		text += fmt.Sprintf("\n环比: %s", alert.Comparison)
	}
	if alert.SampleContent != "" {
		sample := alert.SampleContent
		if len(sample) > maxSampleLen {
			sample = sample[:maxSampleLen] + "..."
		}
		text += fmt.Sprintf("\n示例: %s", sample)
	}
	text += fmt.Sprintf("\n━━━━━━━━━━━━━━━━━━━━━\n⏰ %s", alert.AlertTime.Format("2006-01-02 15:04:05"))
	return n.sendText(text)
}

func (n *FeishuNotifier) SendBatchWarnings(items []BatchWarningItem, alertTime time.Time) error {
	if len(items) == 0 {
		return nil
	}
	var sb strings.Builder
	sb.WriteString("⚠️ [WARNING] 批量服务告警\n━━━━━━━━━━━━━━━━━━━━━\n")
	for i, item := range items {
		if i >= 10 {
			sb.WriteString(fmt.Sprintf("... 共 %d 条告警\n", len(items)))
			break
		}
		sb.WriteString(fmt.Sprintf("🔸 项目: %s\n", item.Project))
		sb.WriteString(fmt.Sprintf("   服务: %s\n", item.Service))
		if item.CallerFile != "" {
			sb.WriteString(fmt.Sprintf("   调用点: %s\n", item.CallerFile))
		}
		if item.Job != "" {
			sb.WriteString(fmt.Sprintf("   机器: %s\n", item.Job))
		}
		sb.WriteString(fmt.Sprintf("   过去5分钟错误: %d 次 (%s)\n", item.ErrorCount, item.Comparison))
		if item.SampleContent != "" {
			sample := item.SampleContent
			if len(sample) > maxBatchSampleLen {
				sample = sample[:maxBatchSampleLen] + "..."
			}
			sb.WriteString(fmt.Sprintf("   示例: %s\n", sample))
		}
		sb.WriteString("━━━━━━━━━━━━━━━━━━━━━\n")
	}
	sb.WriteString(fmt.Sprintf("⏰ %s", alertTime.Format("2006-01-02 15:04:05")))
	return n.sendText(sb.String())
}
