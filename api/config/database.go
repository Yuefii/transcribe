package config

import (
	"log"
	"transcribe/internal/domain"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDB() {
	var err error

	dsn := AppConfig.DatabaseURL

	var gormLogger logger.Interface

	if AppConfig.AppEnv == "development" {
		gormLogger = logger.Default.LogMode(logger.Info)
	} else {
		gormLogger = logger.Default.LogMode(logger.Warn)
	}

	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})

	if err != nil {
		log.Fatal("failed to connect database:", err)
	}

	log.Println("database connected successfully")
}

func AutoMigrate() {
	err := DB.AutoMigrate(&domain.User{}, &domain.TranscriptionJob{})

	if err != nil {
		log.Fatal("failed to migrate database:", err)
	}

	log.Println("database migrated successfully")
}
