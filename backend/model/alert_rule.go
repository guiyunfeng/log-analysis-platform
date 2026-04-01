package model

import (
	"time"
)

type AlertRule struct {
	ID             int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	Name           string    `json:"name" gorm:"size:255;not null"`
	Severity       string    `json:"severity" gorm:"type:enum('critical','warning','noise');not null;default:'warning'"`
	Project        string    `json:"project" gorm:"size:255;default:''"`
	Service        string    `json:"service" gorm:"size:255;default:''"`
	CallerFile     string    `json:"caller_file" gorm:"size:255;default:''"`
	ContentPattern string    `json:"content_pattern" gorm:"size:500;default:''"`
	TimeWindow     int       `json:"time_window" gorm:"not null;default:300"`
	Threshold      int       `json:"threshold" gorm:"not null;default:1"`
	SilenceMinutes int       `json:"silence_minutes" gorm:"not null;default:30"`
	Enabled        bool      `json:"enabled" gorm:"not null;default:true"`
	// Extended fields
	Labels         string    `json:"labels" gorm:"size:500;default:''"`
	Description    string    `json:"description" gorm:"type:text"`
	EffectiveStart string    `json:"effective_start" gorm:"size:10;default:''"`  // e.g. "08:00"
	EffectiveEnd   string    `json:"effective_end" gorm:"size:10;default:''"`    // e.g. "22:00"
	EffectiveDays  string    `json:"effective_days" gorm:"size:20;default:''"`   // e.g. "1,2,3,4,5"
	NotifyChannels string    `json:"notify_channels" gorm:"size:500;default:''"` // comma-separated channel IDs
	NotifyRecovery bool      `json:"notify_recovery" gorm:"not null;default:false"`
	RecoveryWindow int       `json:"recovery_window" gorm:"not null;default:600"` // seconds
	MaxAlertCount  int       `json:"max_alert_count" gorm:"not null;default:0"`   // 0 = unlimited
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (AlertRule) TableName() string {
	return "alert_rules"
}
