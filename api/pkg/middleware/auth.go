package middleware

import (
	"strings"
	"transcribe/pkg/helpers"

	"github.com/gofiber/fiber/v2"
)

func AuthMiddleware(c *fiber.Ctx) error {
	authHandler := c.Get("Authorization")

	if authHandler == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "missing authorization header",
		})
	}

	if !strings.HasPrefix(authHandler, "Bearer ") {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "invalid authorization format",
		})
	}

	tokenString := strings.TrimPrefix(authHandler, "Bearer ")

	claims, err := helpers.ValidateToken(tokenString)

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "invalid or expired token",
		})
	}

	c.Locals("user_id", claims.UserID)
	c.Locals("email", claims.Email)

	return c.Next()
}
