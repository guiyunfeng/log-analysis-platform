package model

import "time"

type NotifyChannel struct {
	ID        uint64    `json:"id" gorm:"primaryKey;autoIncrement"`
	Name      string    `json:"name" gorm:"size:100;not null"`
	Type      string    `json:"type" gorm:"size:50;not null"`
	Config    string    `json:"config" gorm:"type:text;not null"`
	Enabled   bool      `json:"enabled" gorm:"default:true"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (NotifyChannel) TableName() string {
	return "notify_channels"
}
