package http

import (
	"transcribe/internal/domain"
	"transcribe/internal/repository"
	"transcribe/pkg/helpers"

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

	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Name == "" || req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "all field are required",
		})
	}

	if len(req.Password) < 6 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "password must be at least 6 characters",
		})
	}

	user, err := h.userRepo.Create(req)

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	token, err := helpers.GenerateToken(user.ID, user.Email)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to generate token",
		})
	}

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

	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	user, err := h.userRepo.FindByEmail(req.Email)

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "invalid email or password",
		})
	}

	if !h.userRepo.CheckPassword(req.Password, user.Password) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "invalid email or password",
		})
	}

	token, err := helpers.GenerateToken(user.ID, user.Email)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to generate token",
		})
	}

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
