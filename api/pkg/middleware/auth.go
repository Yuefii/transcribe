package middleware

import (
	"strings"
	"transcribe/pkg/helpers"
	"transcribe/pkg/logger"

	"github.com/gofiber/fiber/v2"
)

func AuthMiddleware(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")

	log := logger.Log.WithField("request_ip", c.IP())

	if authHeader == "" {
		log.Warn("missing authorization header")

		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "missing authorization header",
		})
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		log.Warn("invalid authorization format")

		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "invalid authorization format",
		})
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	claims, err := helpers.ValidateToken(tokenString)

	if err != nil {
		log.Warnf("invalid or expired token: %v", err)

		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "invalid or expired token",
		})
	}

	c.Locals("user_id", claims.UserID)
	c.Locals("email", claims.Email)

	return c.Next()
}
