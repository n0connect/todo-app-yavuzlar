package models

import "time"

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UUID         string    `gorm:"uniqueIndex;size:24;not null" json:"uuid"`
	EncryptedKey string    `gorm:"type:text;not null" json:"-"`
	CreatedAt    time.Time `gorm:"autoCreateTime;not null" json:"created_at"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName returns the table name for GORM
func (User) TableName() string {
	return "users"
}
