package main

import (
	"log"
	"os"
	"transcribe/config"
	"transcribe/internal/delivery/routes"

	"github.com/joho/godotenv"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func main() {
	config.InitDB()
	config.AutoMigrate()

	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found")
	}

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	app.Use(cors.New())

	routes.SetupRoutes(app)

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("environment variable PORT is not set")
	}

	log.Printf("server running on port %s", port)
	log.Fatal(app.Listen(":" + port))
}
