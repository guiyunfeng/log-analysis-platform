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
	// Enhanced fields (Nightingale-style)
	Labels         string `json:"labels" gorm:"size:500;default:''"`           // 标签，逗号分隔，如 "env:prod,team:backend"
	EffectiveStart string `json:"effective_start" gorm:"size:10;default:''"`   // 生效开始时间 HH:MM，如 "09:00"
	EffectiveEnd   string `json:"effective_end" gorm:"size:10;default:''"`     // 生效结束时间 HH:MM，如 "18:00"
	EffectiveDays  string `json:"effective_days" gorm:"size:20;default:'1,2,3,4,5'"` // 生效星期，逗号分隔 1-7
	NotifyChannels string `json:"notify_channels" gorm:"size:255;default:''"` // 通知渠道ID列表，逗号分隔
	NotifyRecovery bool   `json:"notify_recovery" gorm:"not null;default:false"` // 是否通知恢复
	RecoveryWindow int    `json:"recovery_window" gorm:"not null;default:600"` // 恢复判定窗口（秒）
	MaxAlertCount  int    `json:"max_alert_count" gorm:"not null;default:0"`   // 最大告警次数，0=无限
	Description    string `json:"description" gorm:"type:text"`                // 规则描述/备注
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (AlertRule) TableName() string {
	return "alert_rules"
}
