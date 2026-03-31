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
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (AlertRule) TableName() string {
	return "alert_rules"
}
