package service

import (
	"log"

	"gorm.io/gorm"
)

// AlerterService manages alert routing and history
type AlerterService struct {
	db *gorm.DB
}

var DefaultAlerter *AlerterService

func InitAlerter(db *gorm.DB) {
	DefaultAlerter = &AlerterService{db: db}
}

// GetDB returns the database instance
func (a *AlerterService) GetDB() *gorm.DB {
	return a.db
}

// LogAlert logs an alert (used by other services)
func (a *AlerterService) LogAlert(severity, project, service, callerFile, job string, errorCount int, sample, comparison string) {
	log.Printf("[ALERT] %s %s/%s %s count=%d", severity, project, service, callerFile, errorCount)
}
