package http

import (
	"context"
	"fmt"
	"transcribe/config"
	"transcribe/internal/repository"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

type RealtimeHandler struct {
	transcriptionRepo *repository.TranscriptionRepository
}

func NewRealtimeHandler() *RealtimeHandler {
	return &RealtimeHandler{
		transcriptionRepo: repository.NewTranscriptionRepository(),
	}
}

func (h *RealtimeHandler) WSUpgrade(c *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(c) {
		c.Locals("allowed", true)
		return c.Next()
	}
	return fiber.ErrUpgradeRequired
}

func (h *RealtimeHandler) ListenForProgress(c *websocket.Conn) {
	jobID := c.Params("job_id")
	ctx := context.Background()
	
	job, err := h.transcriptionRepo.FindByID(jobID)
	if err == nil {
		userID := c.Locals("user_id").(uint)
		if job.UserID != userID {
			c.WriteJSON(fiber.Map{
				"status": "error", 
				"error": "forbidden: you do not have access to this job",
			})
			c.Close()
			return
		}

		initialMsg := fiber.Map{
			"job_id":  job.ID,
			"status":  job.Status,
			"message": "connected. current status fetched.",
		}
		
		if job.Status == "done" || job.Status == "failed" {
			initialMsg["final"] = true
		}
		
		if err := c.WriteJSON(initialMsg); err != nil {
			return
		}
	}

	channelName := fmt.Sprintf("job_progress:%s", jobID)
	pubsub := config.RedisClient.Subscribe(ctx, channelName)
	defer pubsub.Close()
	
	closeCh := make(chan struct{})

	go func() {
		defer close(closeCh)
		
		ch := pubsub.Channel()
		for msg := range ch {
			if err := c.WriteMessage(websocket.TextMessage, []byte(msg.Payload)); err != nil {
				return
			}
		}
	}()

	for {
		_, _, err := c.ReadMessage()
		if err != nil {
			break
		}
	}
}
