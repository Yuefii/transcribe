package config

import (
	"log"
	"os"
	"time"
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
		gormLogger = logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             200 * time.Millisecond,
				LogLevel:                  logger.Info,
				IgnoreRecordNotFoundError: true,
				Colorful:                  true,
			},
		)
	} else {
		gormLogger = logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             1000 * time.Millisecond,
				LogLevel:                  logger.Warn,
				IgnoreRecordNotFoundError: true,
				Colorful:                  false,
			},
		)
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
