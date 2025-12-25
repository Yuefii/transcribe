package main

import (
	"fmt"
	"transcribe/config"
	"transcribe/internal/delivery/routes"
	"transcribe/pkg/helpers"
	"transcribe/pkg/logger"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func main() {
	config.LoadConfig()
	logger.Init(config.AppConfig.AppEnv)
	helpers.InitJWT()

	config.InitDB()
	config.AutoMigrate()
	config.InitRedis()

	app := fiber.New(fiber.Config{
		BodyLimit: 100 * 1024 * 1024,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			logger.Log.Error(err)

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

	logger.Log.Infof("server running on port %s in %s mode", port, config.AppConfig.AppEnv)
	logger.Log.Fatal(app.Listen(fmt.Sprintf(":%s", port)))
}
