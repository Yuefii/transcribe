package http

import (
	"transcribe/internal/domain"
	"transcribe/internal/repository"
	"transcribe/pkg/helpers"
	"transcribe/pkg/logger"

	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	userRepo *repository.UserRepository
}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		userRepo: repository.NewUserRepository(),
	}
}

func (h *AuthHandler) SignUp(c *fiber.Ctx) error {
	req := new(domain.RegisterRequest)

	log := logger.Log.WithField("request_ip", c.IP())

	if err := c.BodyParser(req); err != nil {
		log.Warnf("invalid request body: %v", err)

		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	log = log.WithField("email", req.Email)

	if req.Name == "" || req.Email == "" || req.Password == "" {
		log.Warn("all fields are required validation failed")

		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "all field are required",
		})
	}

	if len(req.Password) < 6 {
		log.Warn("password must be at least 6 characters validation failed")

		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "password must be at least 6 characters",
		})
	}

	user, err := h.userRepo.Create(req)

	if err != nil {
		log.Errorf("failed to create user: %v", err)

		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	log = log.WithField("user_id", user.ID)

	token, err := helpers.GenerateToken(user.ID, user.Email)

	if err != nil {
		log.Errorf("failed to generate token: %v", err)

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to generate token",
		})
	}

	log.Info("user registered successfully")

	return c.Status(fiber.StatusCreated).JSON(domain.AuthResponse{
		Message: "user registered successfully",
		Token:   token,
		User: domain.UserProfile{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Email,
		},
	})
}

func (h *AuthHandler) SignIn(c *fiber.Ctx) error {
	req := new(domain.LoginRequest)

	log := logger.Log.WithField("request_ip", c.IP())

	if err := c.BodyParser(req); err != nil {
		log.Warnf("invalid request body: %v", err)

		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	log = log.WithField("email", req.Email)

	user, err := h.userRepo.FindByEmail(req.Email)

	if err != nil {
		log.Warnf("failed to find user by email: %v", err)

		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "invalid email or password",
		})
	}

	log = log.WithField("user_id", user.ID)

	if !h.userRepo.CheckPassword(req.Password, user.Password) {
		log.Warn("invalid password")

		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "invalid email or password",
		})
	}

	token, err := helpers.GenerateToken(user.ID, user.Email)

	if err != nil {
		log.Errorf("failed to generate token: %v", err)

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to generate token",
		})
	}

	log.Info("user signed in successfully")

	return c.JSON(domain.AuthResponse{
		Message: "signin successfully",
		Token:   token,
		User: domain.UserProfile{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Email,
		},
	})
}
