package routes

import (
	"transcribe/internal/delivery/http"
	"transcribe/pkg/middleware"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(app *fiber.App) {
	authHandler := http.NewAuthHandler()
	userHandler := http.NewUserHandler()

	transcriptionHandler := http.NewTranscriptionHandler()
	realtimeHandler := http.NewRealtimeHandler()

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

	transcribe := api.Group("/transcribe", middleware.AuthMiddleware)
	transcribe.Post("/", transcriptionHandler.CreateJob)
	transcribe.Get("/:job_id", transcriptionHandler.GetJobStatus)
	transcribe.Get("/", transcriptionHandler.GetUserJobs)

	transcribe.Delete("/:job_id", transcriptionHandler.DeleteJob)

	ws := api.Group("/ws", middleware.AuthMiddleware, realtimeHandler.WSUpgrade)
	ws.Get("/job/:job_id", websocket.New(realtimeHandler.ListenForProgress))
}
