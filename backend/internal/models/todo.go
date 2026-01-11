package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Priority levels for todos
type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

type Todo struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserUUID  string         `gorm:"size:24;index;not null" json:"user_uuid"`
	Title     string         `gorm:"type:text;not null" json:"title"`
	Completed bool           `gorm:"default:false;not null" json:"completed"`
	Tags      pq.StringArray `gorm:"type:text[]" json:"tags"`
	DueDate   *time.Time     `gorm:"type:timestamp" json:"due_date,omitempty"`
	Priority  Priority       `gorm:"size:10;default:'medium'" json:"priority"`
	CreatedAt time.Time      `gorm:"autoCreateTime;not null" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName returns the table name for GORM
func (Todo) TableName() string {
	return "todos"
}
