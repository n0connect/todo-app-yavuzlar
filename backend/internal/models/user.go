package models

import "time"

type User struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	UUID          string    `gorm:"uniqueIndex;size:24;not null" json:"uuid"` // Internal user ID (kept for backward compatibility)
	AccountLookup string    `gorm:"uniqueIndex;not null" json:"-"`            // HMAC(pepper, accountNumber) for fast lookup
	AccountHash   string    `gorm:"not null" json:"-"`                        // Argon2id hash of accountNumber
	EncryptedKey  string    `gorm:"type:text;not null" json:"-"`
	MasterKeyID   string    `gorm:"size:32;not null;default:''" json:"-"`
	CreatedAt     time.Time `gorm:"autoCreateTime;not null" json:"created_at"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName returns the table name for GORM
func (User) TableName() string {
	return "users"
}
