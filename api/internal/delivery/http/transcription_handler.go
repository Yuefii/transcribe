package http

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"transcribe/config"
	"transcribe/internal/domain"
	"transcribe/internal/repository"

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

	file, err := c.FormFile("audio")

	if err != nil {
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
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid file type. Allowed: mp3, wav, m4a, ogg, flac, mp4, avi, mov",
		})
	}

	maxSize := int64(100 * 1024 * 1024)

	if file.Size > maxSize {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "file size exceeds 100mb limit",
		})
	}

	jobID := uuid.New().String()

	uploadDir := os.Getenv("UPLOAD_DIR")

	if uploadDir == "" {
		log.Fatal("environment variable UPLOAD_DIR is not set")
	}

	userDir := filepath.Join(uploadDir, fmt.Sprintf("user_%d", userID))

	if err := os.Mkdir(userDir, 0755); err != nil && !os.IsExist(err) {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create upload directory",
		})
	}

	fileName := fmt.Sprintf("%s%s", jobID, ext)
	filePath := filepath.Join(userDir, fileName)

	if err := c.SaveFile(file, filePath); err != nil {
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
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to queue job",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(domain.TranscriptionResponse{
		JobID:   jobID,
		Status:  "queued",
		Message: "job created and queued for transcription",
	})
}

func (h *TranscriptionHandler) GetJobStatus(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	JobID := c.Params("job_id")

	job, err := h.transcriptionRepo.FindByID(JobID)

	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "job not found",
		})
	}

	if job.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	response := domain.TranscriptionResponse{
		JobID:       job.ID,
		Status:      job.Status,
		FileName:    job.FileName,
		FileSize:    job.FileSize,
		CreatedAt:   job.CreatedAt,
		CompletedAt: job.CompletedAt,
	}

	if job.Status == "done" {
		response.Text = job.Text
	}

	if job.Status == "failed" {
		response.ErrorMsg = job.ErrorMsg
	}

	return c.JSON(response)
}

func (h *TranscriptionHandler) GetUserJobs(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

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
			CreatedAt:   job.CreatedAt,
			CompletedAt: job.CompletedAt,
		}
		if job.Status == "done" {
			resp.Text = job.Text
		}

		if job.Status == "failed" {
			resp.ErrorMsg = job.ErrorMsg
		}

		responses = append(responses, resp)
	}

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

	job, err := h.transcriptionRepo.FindByID(jobID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "job not found",
		})
	}

	if job.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "access denied",
		})
	}

	if err := os.Remove(job.FilePath); err != nil {
		fmt.Printf("failed to delete file: %v\n", err)
	}

	if err := h.transcriptionRepo.Delete(jobID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to delete job",
		})
	}

	return c.JSON(fiber.Map{
		"message": "job deleted successfully",
	})
}
