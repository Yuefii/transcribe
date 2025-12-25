package config

import (
	"time"
	"transcribe/internal/domain"
	"transcribe/pkg/logger"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDB() {
	var err error

	dsn := AppConfig.DatabaseURL

	var gormLogger gormlogger.Interface

	if AppConfig.AppEnv == "development" {
		gormLogger = gormlogger.New(
			logger.Log,
			gormlogger.Config{
				SlowThreshold:             200 * time.Millisecond,
				LogLevel:                  gormlogger.Info,
				IgnoreRecordNotFoundError: true,
				Colorful:                  true,
			},
		)
	} else {
		gormLogger = gormlogger.New(
			logger.Log,
			gormlogger.Config{
				SlowThreshold:             1000 * time.Millisecond, // 1 second
				LogLevel:                  gormlogger.Warn,
				IgnoreRecordNotFoundError: true,
				Colorful:                  false,
			},
		)
	}

	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})

	if err != nil {
		logger.Log.Fatal("failed to connect database:", err)
	}

	logger.Log.Info("database connected successfully")
}

func AutoMigrate() {
	err := DB.AutoMigrate(&domain.User{}, &domain.TranscriptionJob{})

	if err != nil {
		logger.Log.Fatal("failed to migrate database:", err)
	}

	logger.Log.Info("database migrated successfully")
}
