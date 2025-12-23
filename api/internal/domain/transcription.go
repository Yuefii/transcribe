package domain

import (
	"time"

	"gorm.io/gorm"
)

type TranscriptionJob struct {
	ID          string         `gorm:"type:varchar(36);primaryKey" json:"job_id"`
	UserID      uint           `gorm:"not null;index" json:"user_id"`
	User        User           `gorm:"foreignKey:UserID" json:"-"`
	FileName    string         `gorm:"type:varchar(255);not null" json:"file_name"`
	FilePath    string         `gorm:"type:varchar(500);not null" json:"file_path"`
	FileSize    int64          `gorm:"not null" json:"file_size"`
	Status      string         `gorm:"type:varchar(20);not null;default:'queued';index" json:"status"`
	Text        string         `gorm:"type:text" json:"text,omitempty"`
	ErrorMsg    string         `gorm:"type:text" json:"error_message,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

type TranscriptionResponse struct {
	JobID       string     `json:"job_id"`
	Status      string     `json:"status"`
	Message     string     `json:"message,omitempty"`
	Text        string     `json:"text,omitempty"`
	FileName    string     `json:"file_name,omitempty"`
	FileSize    int64      `json:"file_size,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	ErrorMsg    string     `json:"error_message,omitempty"`
}
