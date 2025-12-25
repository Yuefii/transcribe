package main

import (
	"log"
	"transcribe/config"
	"transcribe/internal/delivery/routes"
	"transcribe/pkg/helpers"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func main() {
	config.LoadConfig()
	helpers.InitJWT()

	config.InitDB()
	config.AutoMigrate()
	config.InitRedis()

	app := fiber.New(fiber.Config{
		BodyLimit: 100 * 1024 * 1024,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			if config.AppConfig.AppEnv == "development" {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": err.Error(),
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal Server Error",
			})
		},
	})

	app.Use(cors.New())

	routes.SetupRoutes(app)

	port := config.AppConfig.Port

	log.Printf("server running on port %s in %s mode", port, config.AppConfig.AppEnv)
	log.Fatal(app.Listen(":" + port))
}
