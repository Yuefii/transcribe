package repository

import (
	"errors"
	"time"
	"transcribe/config"
	"transcribe/internal/domain"

	"gorm.io/gorm"
)

type TranscriptionRepository struct{}

func NewTranscriptionRepository() *TranscriptionRepository {
	return &TranscriptionRepository{}
}

func (r *TranscriptionRepository) Create(job *domain.TranscriptionJob) error {
	result := config.DB.Create(job)
	return result.Error
}

func (r *TranscriptionRepository) FindByID(jobID string) (*domain.TranscriptionJob, error) {
	var job domain.TranscriptionJob

	result := config.DB.Where("id = ?", jobID).First(&job)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("job not found")
		}
		return nil, result.Error
	}

	return &job, nil
}

func (r *TranscriptionRepository) FindByUserID(userID uint, page, pageSize int) ([]domain.TranscriptionJob, int64, error) {
	var jobs []domain.TranscriptionJob
	var total int64

	offset := (page - 1) * pageSize

	config.DB.Model(&domain.TranscriptionJob{}).Where("user_id = ?", userID).Count(&total)

	result := config.DB.Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&jobs)

	if result.Error != nil {
		return nil, 0, result.Error
	}

	return jobs, total, nil
}

func (r *TranscriptionRepository) UpdateStatus(jobID, status string, text, errorMsg *string, segments *string, duration *float64) error {
	updates := map[string]interface{}{
		"status": status,
	}

	if text != nil {
		updates["text"] = *text
	}

	if errorMsg != nil {
		updates["error_msg"] = *errorMsg
	}

	if segments != nil {
		updates["segments"] = *segments
	}

	if duration != nil {
		updates["duration"] = *duration
	}

	if status == "done" || status == "failed" {
		now := time.Now()
		updates["completed_at"] = now
	}

	result := config.DB.Model(&domain.TranscriptionJob{}).Where("id = ?", jobID).Updates(updates)
	return result.Error
}

func (r *TranscriptionRepository) Delete(JobID string) error {
	result := config.DB.Delete(&domain.TranscriptionJob{}, "id = ?", JobID)

	return result.Error
}
