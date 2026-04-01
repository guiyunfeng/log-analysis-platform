package service

import (
	"fmt"
	"log"
	"strconv"
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
		start := now.Add(-time.Duration(rule.TimeWindow) * time.Second)
		count, sample, err := a.loki.CountErrors(rule.Project, rule.Service, "", rule.CallerFile, rule.ContentPattern, start, now)
		if err != nil {
			log.Printf("CountErrors for rule %d error: %v", rule.ID, err)
			continue
		}

		if count <= int64(rule.Threshold) {
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
		a.notify.SendAlert(AlertMessage{
			Severity:      alert.Severity,
			Project:       alert.Project,
			Service:       alert.Service,
			CallerFile:    alert.CallerFile,
			Job:           alert.Job,
			ErrorCount:    alert.ErrorCount,
			Comparison:    alert.Comparison,
			SampleContent: alert.SampleContent,
			AlertTime:     now,
		})
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
