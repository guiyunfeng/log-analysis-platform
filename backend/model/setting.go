package model

import (
	"time"
)

type Setting struct {
	ID          int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	Key         string    `json:"key" gorm:"size:255;not null;uniqueIndex"`
	Value       string    `json:"value" gorm:"type:text;not null"`
	Description string    `json:"description" gorm:"size:500;default:''"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Setting) TableName() string {
	return "settings"
}
