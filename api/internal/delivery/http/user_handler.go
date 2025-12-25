package http

import (
	"transcribe/internal/domain"
	"transcribe/internal/repository"
	"transcribe/pkg/logger"

	"github.com/gofiber/fiber/v2"
)

type UserHandler struct {
	userRepo *repository.UserRepository
}

func NewUserHandler() *UserHandler {
	return &UserHandler{
		userRepo: repository.NewUserRepository(),
	}
}

func (h *UserHandler) GetProfile(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	log := logger.Log.WithField("user_id", userID)

	user, err := h.userRepo.FindByID(userID)

	if err != nil {
		log.Warnf("user not found: %v", err)

		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "user not found",
		})
	}

	log.Info("user profile retrieved successfully")

	return c.JSON(fiber.Map{
		"user": domain.UserProfile{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Email,
		},
	})
}

func (h *UserHandler) UpdateProfile(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	log := logger.Log.WithField("user_id", userID)

	var req struct {
		Name string `json:"name"`
	}

	if err := c.BodyParser(&req); err != nil {
		log.Warnf("invalid request body for profile update: %v", err)

		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	updates := map[string]interface{}{
		"name": req.Name,
	}

	err := h.userRepo.Update(userID, updates)

	if err != nil {
		log.Errorf("failed to update profile: %v", err)

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update profile",
		})
	}

	log.Info("profile updated successfully")

	return c.JSON(fiber.Map{
		"message": "profile update succesfully",
	})
}
