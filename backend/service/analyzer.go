package service

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"log-analysis-platform/model"

	"gorm.io/gorm"
)

// AnalyzerService handles anomaly detection and alert evaluation
type AnalyzerService struct {
	db     *gorm.DB
	loki   *LokiService
	notify *NotifyService
}

var DefaultAnalyzer *AnalyzerService

func InitAnalyzer(db *gorm.DB) {
	DefaultAnalyzer = &AnalyzerService{
		db:     db,
		loki:   DefaultLokiService,
		notify: DefaultNotifyService,
	}
}

// isRuleEffective checks whether a rule should be evaluated at the given time
// based on EffectiveDays and EffectiveStart/EffectiveEnd.
func (a *AnalyzerService) isRuleEffective(rule model.AlertRule, now time.Time) bool {
	// Check effective weekdays (1=Monday … 7=Sunday, matching ISO 8601)
	if rule.EffectiveDays != "" {
		weekday := int(now.Weekday()) // 0=Sunday
		// Convert Go weekday (0=Sunday) to ISO (1=Monday … 7=Sunday)
		isoDay := weekday
		if isoDay == 0 {
			isoDay = 7
		}
		found := false
		for _, part := range strings.Split(rule.EffectiveDays, ",") {
			part = strings.TrimSpace(part)
			if d, err := strconv.Atoi(part); err == nil && d == isoDay {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check effective time range (HH:mm format)
	if rule.EffectiveStart != "" && rule.EffectiveEnd != "" {
		currentTime := now.Format("15:04")
		if rule.EffectiveStart <= rule.EffectiveEnd {
			// Same-day range, e.g. "08:00" – "22:00"
			if currentTime < rule.EffectiveStart || currentTime > rule.EffectiveEnd {
				return false
			}
		} else {
			// Overnight range, e.g. "22:00" – "06:00"
			if currentTime < rule.EffectiveStart && currentTime > rule.EffectiveEnd {
				return false
			}
		}
	}

	return true
}

// RunAnalysis is called periodically to check all alert rules
func (a *AnalyzerService) RunAnalysis() {
	log.Println("Running alert analysis...")

	now := time.Now()

	// Load global settings
	settings := a.loadSettings()
	spikeMultiplier, _ := strconv.ParseFloat(settings["spike_multiplier"], 64)
	if spikeMultiplier <= 0 {
		spikeMultiplier = 10
	}
	globalThreshold, _ := strconv.Atoi(settings["global_threshold"])
	if globalThreshold <= 0 {
		globalThreshold = 100
	}
	globalWindowSec, _ := strconv.Atoi(settings["global_time_window"])
	if globalWindowSec <= 0 {
		globalWindowSec = 300
	}

	// Load enabled rules
	var rules []model.AlertRule
	if err := a.db.Where("enabled = ?", true).Find(&rules).Error; err != nil {
		log.Printf("Load rules error: %v", err)
		return
	}

	var criticalAlerts []model.AlertHistory
	var warningAlerts []model.AlertHistory

	for _, rule := range rules {
		// Check effective time window
		if !a.isRuleEffective(rule, now) {
			continue
		}

		// Check MaxAlertCount: if > 0, count today's alerts for this rule
		if rule.MaxAlertCount > 0 {
			todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			var todayCount int64
			a.db.Model(&model.AlertHistory{}).
				Where("rule_id = ? AND created_at >= ?", rule.ID, todayStart).
				Count(&todayCount)
			if int(todayCount) >= rule.MaxAlertCount {
				log.Printf("Rule %d reached max alert count (%d), skipping", rule.ID, rule.MaxAlertCount)
				continue
			}
		}

		start := now.Add(-time.Duration(rule.TimeWindow) * time.Second)
		count, sample, err := a.loki.CountErrors(rule.Project, rule.Service, "", rule.CallerFile, rule.ContentPattern, start, now)
		if err != nil {
			log.Printf("CountErrors for rule %d error: %v", rule.ID, err)
			continue
		}

		if count <= int64(rule.Threshold) {
			// Check recovery: if NotifyRecovery is enabled, see if we should send recovery notice
			if rule.NotifyRecovery {
				a.checkRecovery(rule, now)
			}
			continue
		}

		// Check silence
		if a.isSilenced(rule.Service, rule.CallerFile, rule.Severity, rule.SilenceMinutes) {
			continue
		}

		// Compute comparison
		comparison := a.computeComparison(rule.Project, rule.Service, rule.CallerFile, rule.ContentPattern, now, rule.TimeWindow)

		ruleID := rule.ID
		hist := model.AlertHistory{
			RuleID:        &ruleID,
			Severity:      rule.Severity,
			Project:       rule.Project,
			Service:       rule.Service,
			CallerFile:    rule.CallerFile,
			ErrorCount:    int(count),
			SampleContent: sample,
			Comparison:    comparison,
			Labels:        rule.Labels,
		}

		if err := a.db.Create(&hist).Error; err != nil {
			log.Printf("Save alert history error: %v", err)
			continue
		}

		if rule.Severity == "critical" {
			criticalAlerts = append(criticalAlerts, hist)
		} else if rule.Severity == "warning" {
			warningAlerts = append(warningAlerts, hist)
		}
	}

	// Spike detection
	a.runSpikeDetection(now, spikeMultiplier, &criticalAlerts, &warningAlerts)

	// Send critical alerts immediately
	for _, alert := range criticalAlerts {
		msg := AlertMessage{
			Severity:      alert.Severity,
			Project:       alert.Project,
			Service:       alert.Service,
			CallerFile:    alert.CallerFile,
			Job:           alert.Job,
			ErrorCount:    alert.ErrorCount,
			Comparison:    alert.Comparison,
			SampleContent: alert.SampleContent,
			AlertTime:     now,
		}
		// Use per-rule notify channels if configured
		a.sendAlertWithRuleRouting(alert.RuleID, msg)
		a.db.Model(&alert).Update("notified", true)
	}

	// Batch send warnings (aggregated)
	if len(warningAlerts) > 0 {
		var batchItems []BatchWarningItem
		for _, alert := range warningAlerts {
			batchItems = append(batchItems, BatchWarningItem{
				Project:       alert.Project,
				Service:       alert.Service,
				CallerFile:    alert.CallerFile,
				Job:           alert.Job,
				ErrorCount:    alert.ErrorCount,
				Comparison:    alert.Comparison,
				SampleContent: alert.SampleContent,
			})
		}
		a.notify.SendBatchWarnings(batchItems, now)
		for _, alert := range warningAlerts {
			a.db.Model(&alert).Update("notified", true)
		}
	}

	log.Printf("Analysis done: %d critical, %d warning alerts generated", len(criticalAlerts), len(warningAlerts))
}

// sendAlertWithRuleRouting sends an alert using rule-specific channels if configured,
// falling back to all enabled channels.
func (a *AnalyzerService) sendAlertWithRuleRouting(ruleID *int64, msg AlertMessage) {
	if ruleID != nil {
		var rule model.AlertRule
		if err := a.db.First(&rule, *ruleID).Error; err == nil && rule.NotifyChannels != "" {
			a.notify.SendAlertToChannels(rule.NotifyChannels, msg)
			return
		}
	}
	a.notify.SendAlert(msg)
}

// checkRecovery sends a recovery notification if the rule had previous unrecovered alerts
// and the error count is now below the threshold (within RecoveryWindow seconds).
func (a *AnalyzerService) checkRecovery(rule model.AlertRule, now time.Time) {
	window := rule.RecoveryWindow
	if window <= 0 {
		window = 600
	}
	since := now.Add(-time.Duration(window) * time.Second)

	// Find unrecovered, un-notified-for-recovery alerts within the recovery window
	var unrecovered []model.AlertHistory
	a.db.Where(
		"rule_id = ? AND resolved = ? AND recovery_notified = ? AND created_at > ?",
		rule.ID, false, false, since,
	).Find(&unrecovered)

	if len(unrecovered) == 0 {
		return
	}

	// Mark them as recovered and send a recovery message
	for i := range unrecovered {
		resolvedAt := now
		a.db.Model(&unrecovered[i]).Updates(map[string]interface{}{
			"resolved":          true,
			"resolved_at":       resolvedAt,
			"recovery_notified": true,
		})
	}

	recoveryMsg := AlertMessage{
		Severity:   "warning",
		Project:    rule.Project,
		Service:    rule.Service,
		CallerFile: rule.CallerFile,
		AlertTime:  now,
	}

	// Use rule-specific channels for recovery notification too
	if rule.NotifyChannels != "" {
		a.notify.SendAlertToChannels(rule.NotifyChannels, recoveryMsg)
	} else {
		a.notify.SendRecovery(recoveryMsg)
	}
}

func (a *AnalyzerService) runSpikeDetection(now time.Time, multiplier float64, criticals, warnings *[]model.AlertHistory) {
	// Get current 5min error counts by service+caller_file
	start5m := now.Add(-5 * time.Minute)
	start1h := now.Add(-1 * time.Hour)

	// Use Loki to get top services in last 5 min
	topServices, err := a.loki.QueryTopNServices("", "", "", start5m, now, 20)
	if err != nil {
		log.Printf("Spike detection query error: %v", err)
		return
	}

	for _, svc := range topServices {
		service := svc.Name
		project := svc.Extra["project"]

		// Count current 5 min
		current5m, _, _ := a.loki.CountErrors(project, service, "", "", "", start5m, now)
		if current5m == 0 {
			continue
		}

		// Count last 1 hour (excluding last 5 min) in 5-min windows -> avg
		// Approximate: total last hour / 12 = avg per 5 min
		total1h, _, _ := a.loki.CountErrors(project, service, "", "", "", start1h, start5m)
		avg5m := float64(total1h) / 11.0 // 11 5-min windows in 55 min
		if avg5m < 1 {
			avg5m = 1
		}

		ratio := float64(current5m) / avg5m
		if ratio <= multiplier {
			continue
		}

		// Check silence
		if a.isSilenced(service, "", "warning", 30) {
			continue
		}

		comparison := fmt.Sprintf("↑ %.0f%%", (ratio-1)*100)

		// Get a sample log entry for both content and job info
		sampleContent := ""
		sampleJob := ""
		entries, _, _ := a.loki.QueryLogs(project, service, "", "", "", start5m, now, 1)
		if len(entries) > 0 {
			sampleContent = entries[0].Content
			sampleJob = entries[0].Job
		}

		hist := model.AlertHistory{
			Severity:      "warning",
			Project:       project,
			Service:       service,
			Job:           sampleJob,
			ErrorCount:    int(current5m),
			SampleContent: sampleContent,
			Comparison:    comparison,
		}
		if err := a.db.Create(&hist).Error; err == nil {
			*warnings = append(*warnings, hist)
		}
	}
}

func (a *AnalyzerService) isSilenced(service, callerFile, severity string, silenceMinutes int) bool {
	if silenceMinutes <= 0 {
		return false
	}
	since := time.Now().Add(-time.Duration(silenceMinutes) * time.Minute)
	var count int64
	a.db.Model(&model.AlertHistory{}).
		Where("service = ? AND caller_file = ? AND severity = ? AND created_at > ?",
			service, callerFile, severity, since).
		Count(&count)
	return count > 0
}

func (a *AnalyzerService) computeComparison(project, service, callerFile, contentPattern string, now time.Time, windowSec int) string {
	current, _, _ := a.loki.CountErrors(project, service, "", callerFile, contentPattern,
		now.Add(-time.Duration(windowSec)*time.Second), now)
	prev, _, _ := a.loki.CountErrors(project, service, "", callerFile, contentPattern,
		now.Add(-time.Duration(windowSec*2)*time.Second), now.Add(-time.Duration(windowSec)*time.Second))

	if prev == 0 {
		if current > 0 {
			return "↑ 新增"
		}
		return ""
	}

	change := float64(current-prev) / float64(prev) * 100
	if change > 0 {
		return fmt.Sprintf("↑ %.0f%%", change)
	}
	return fmt.Sprintf("↓ %.0f%%", -change)
}

func (a *AnalyzerService) loadSettings() map[string]string {
	var settings []model.Setting
	a.db.Find(&settings)
	result := map[string]string{}
	for _, s := range settings {
		result[s.Key] = s.Value
	}
	return result
}

