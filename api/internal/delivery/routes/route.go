package routes

import (
	"transcribe/internal/delivery/http"
	"transcribe/pkg/middleware"

	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(app *fiber.App) {
	authHandler := http.NewAuthHandler()
	userHandler := http.NewUserHandler()

	api := app.Group("/api")

	api.Get("health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "server is running",
		})
	})

	auth := api.Group("/auth")
	auth.Post("/sign-up", authHandler.SignUp)
	auth.Post("/sign-in", authHandler.SignIn)

	proctected := api.Group("/user", middleware.AuthMiddleware)
	proctected.Get("/profile", userHandler.GetProfile)
	proctected.Put("/profile", userHandler.UpdateProfile)
}
