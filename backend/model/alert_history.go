package model

import (
	"time"
)

type AlertHistory struct {
	ID            int64      `json:"id" gorm:"primaryKey;autoIncrement"`
	RuleID        *int64     `json:"rule_id" gorm:"default:null"`
	Severity      string     `json:"severity" gorm:"type:enum('critical','warning','noise');not null"`
	Project       string     `json:"project" gorm:"size:255;not null"`
	Service       string     `json:"service" gorm:"size:255;not null"`
	CallerFile    string     `json:"caller_file" gorm:"size:255;default:''"`
	Job           string     `json:"job" gorm:"size:255;default:''"`
	ErrorCount    int        `json:"error_count" gorm:"not null"`
	SampleContent string     `json:"sample_content" gorm:"type:text"`
	Comparison    string     `json:"comparison" gorm:"size:100;default:''"`
	Resolved      bool       `json:"resolved" gorm:"not null;default:false"`
	ResolvedAt    *time.Time `json:"resolved_at" gorm:"default:null"`
	Notified      bool       `json:"notified" gorm:"not null;default:false"`
	CreatedAt     time.Time  `json:"created_at"`
}

func (AlertHistory) TableName() string {
	return "alert_history"
}
