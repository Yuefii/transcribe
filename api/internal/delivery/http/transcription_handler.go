package http

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"transcribe/config"
	"transcribe/internal/domain"
	"transcribe/internal/repository"
	"transcribe/pkg/logger"

	"github.com/sirupsen/logrus"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type TranscriptionHandler struct {
	transcriptionRepo *repository.TranscriptionRepository
}

func NewTranscriptionHandler() *TranscriptionHandler {
	return &TranscriptionHandler{
		transcriptionRepo: repository.NewTranscriptionRepository(),
	}
}

func (h *TranscriptionHandler) CreateJob(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	log := logger.Log.WithField("user_id", userID)

	file, err := c.FormFile("audio")

	if err != nil {
		log.Warn("audio file is required")

		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "audio file is required",
		})
	}

	allowedTypes := []string{".mp3", ".wav", ".m4a", ".ogg", ".flac", ".mp4", ".avi", ".mov"}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	isValid := false

	for _, t := range allowedTypes {
		if ext == t {
			isValid = true
			break
		}
	}

	if !isValid {
		log.Warnf("invalid file type: %s", ext)

		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid file type. Allowed: mp3, wav, m4a, ogg, flac, mp4, avi, mov",
		})
	}

	maxSize := int64(100 * 1024 * 1024)

	if file.Size > maxSize {
		log.Warnf("file size exceeds 100mb limit: %d", file.Size)

		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "file size exceeds 100mb limit",
		})
	}

	jobID := uuid.New().String()
	log = log.WithField("job_id", jobID)

	uploadDir := config.AppConfig.UploadDir
	userDir := filepath.Join(uploadDir, "user_"+strconv.Itoa(int(userID)))

	if err := os.MkdirAll(userDir, 0755); err != nil {
		log.Errorf("failed to create upload directory: %v", err)

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create upload directory",
		})
	}

	fileName := jobID + ext
	filePath := filepath.Join(userDir, fileName)

	if err := c.SaveFile(file, filePath); err != nil {
		log.Errorf("failed to save file: %v", err)

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to save file",
		})
	}

	job := &domain.TranscriptionJob{
		ID:       jobID,
		UserID:   userID,
		FileName: file.Filename,
		FilePath: filePath,
		FileSize: file.Size,
		Status:   "queued",
	}

	if err := h.transcriptionRepo.Create(job); err != nil {
		log.Errorf("failed to create transcription job in db: %v", err)

		os.Remove(filePath)

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create transcription job",
		})
	}

	jobData := map[string]interface{}{
		"job_id":    jobID,
		"file_path": filePath,
		"user_id":   userID,
	}

	jobJSON, _ := json.Marshal(jobData)

	ctx := context.Background()

	if err := config.RedisClient.RPush(ctx, "transcription_queue", jobJSON).Err(); err != nil {
		log.Errorf("failed to queue job in redis: %v", err)

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to queue job",
		})
	}

	log.Info("transcription job created and queued successfully")

	return c.Status(fiber.StatusCreated).JSON(domain.TranscriptionResponse{
		JobID:   jobID,
		Status:  "queued",
		Message: "job created and queued for transcription",
	})
}

func (h *TranscriptionHandler) GetJobStatus(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	jobID := c.Params("job_id")

	log := logger.Log.WithFields(logrus.Fields{
		"user_id": userID,
		"job_id":  jobID,
	})

	job, err := h.transcriptionRepo.FindByID(jobID)

	if err != nil {
		log.Warnf("job not found: %v", err)

		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "job not found",
		})
	}

	if job.UserID != userID {
		log.Warn("access denied to job")

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	response := domain.TranscriptionResponse{
		JobID:       job.ID,
		Status:      job.Status,
		FileName:    job.FileName,
		FileSize:    job.FileSize,
		Duration:    job.Duration,
		CreatedAt:   job.CreatedAt,
		CompletedAt: job.CompletedAt,
	}

	if job.Status == "done" {
		response.Text = job.Text

		if job.Segments != "" {
			var segments []domain.Segment
			if err := json.Unmarshal([]byte(job.Segments), &segments); err == nil {
				response.Segments = segments
			}
		}
	}

	if job.Status == "failed" {
		response.ErrorMsg = job.ErrorMsg
	}

	log.Info("job status retrieved successfully")

	return c.JSON(response)
}

func (h *TranscriptionHandler) GetUserJobs(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	log := logger.Log.WithField("user_id", userID)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "10"))

	if page < 1 {
		page = 1
	}

	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	jobs, total, err := h.transcriptionRepo.FindByUserID(userID, page, pageSize)

	if err != nil {
		log.Errorf("failed to fetch jobs for user: %v", err)

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to fetch jobs",
		})
	}

	var responses []domain.TranscriptionResponse

	for _, job := range jobs {
		resp := domain.TranscriptionResponse{
			JobID:       job.ID,
			Status:      job.Status,
			FileName:    job.FileName,
			FileSize:    job.FileSize,
			Duration:    job.Duration,
			CreatedAt:   job.CreatedAt,
			CompletedAt: job.CompletedAt,
		}
		if job.Status == "done" {
			resp.Text = job.Text

			if job.Segments != "" {
				var segments []domain.Segment
				if err := json.Unmarshal([]byte(job.Segments), &segments); err == nil {
					resp.Segments = segments
				}
			}
		}

		if job.Status == "failed" {
			resp.ErrorMsg = job.ErrorMsg
		}

		responses = append(responses, resp)
	}

	log.Infof("retrieved %d jobs for user", len(jobs))

	return c.JSON(fiber.Map{
		"jobs": responses,
		"pagination": fiber.Map{
			"page":       page,
			"page_size":  pageSize,
			"total":      total,
			"total_page": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

func (h *TranscriptionHandler) DeleteJob(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	jobID := c.Params("job_id")

	log := logger.Log.WithFields(logrus.Fields{
		"user_id": userID,
		"job_id":  jobID,
	})

	job, err := h.transcriptionRepo.FindByID(jobID)
	if err != nil {
		log.Warnf("job not found for deletion: %v", err)

		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "job not found",
		})
	}

	if job.UserID != userID {
		log.Warn("access denied for job deletion")

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "access denied",
		})
	}

	if err := os.Remove(job.FilePath); err != nil {
		log.Warnf("failed to delete file from filesystem: %v", err)
	}

	if err := h.transcriptionRepo.Delete(jobID); err != nil {
		log.Errorf("failed to delete job from db: %v", err)

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to delete job",
		})
	}

	log.Info("job deleted successfully")

	return c.JSON(fiber.Map{
		"message": "job deleted successfully",
	})
}

func (h *TranscriptionHandler) CancelJob(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	jobID := c.Params("job_id")
	ctx := context.Background()

	log := logger.Log.WithFields(logrus.Fields{
		"user_id": userID,
		"job_id":  jobID,
	})

	job, err := h.transcriptionRepo.FindByID(jobID)
	
	if err != nil {
		log.Warnf("job not found for cancellation: %v", err)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "job not found",
		})
	}

	if job.UserID != userID {
		log.Warn("access denied for job cancellation")
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "access denied",
		})
	}

	if job.Status == "done" || job.Status == "failed" || job.Status == "cancelled" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "job is already finished or cancelled",
		})
	}

	cancelKey := "job_cancellation:" + jobID

	if err := config.RedisClient.Set(ctx, cancelKey, "1", 24*time.Hour).Err(); err != nil {
		log.Errorf("failed to set cancellation signal in redis: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to process cancellation signal",
		})
	}

	job.Status = "cancelled"

	if err := h.transcriptionRepo.UpdateStatus(job.ID, "cancelled", nil, nil, nil, nil); err != nil {
		 log.Errorf("failed to update job status to cancelled: %v", err)
	}

	log.Info("job cancellation signal sent")

	return c.JSON(fiber.Map{
		"message": "job cancellation request sent",
		"status":  "cancelled",
	})
}
