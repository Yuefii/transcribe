package config

import (
	"log"
	"os"
	"transcribe/internal/domain"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDB() {
	var err error

	dsn := os.Getenv("DATABASE_URL")

	if dsn == "" {
		log.Fatal("environment variable DATABASE_URL is not set")
	}

	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		log.Fatal("failed to connect database:", err)
	}

	log.Println("database connected successfully")
}

func AutoMigrate() {
	err := DB.AutoMigrate(&domain.User{})

	if err != nil {
		log.Fatal("failed to migrate database:", err)
	}

	log.Println("database migrated successfully")
}
