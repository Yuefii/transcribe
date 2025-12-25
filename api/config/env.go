package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv      string
	DatabaseURL string
	RedisURL    string
	Port        string
	JWTSecret   string
	UploadDir   string
}

var AppConfig *Config

func LoadConfig() {
	err := godotenv.Load()

	if err != nil {
		log.Println("no .env file found, reading from environment variables")
	}

	AppConfig = &Config{
		AppEnv:      getEnv("APP_ENV", "development"),
		DatabaseURL: getEnv("DATABASE_URL", ""),
		RedisURL:    getEnv("REDIS_URL", ""),
		Port:        getEnv("PORT", ""),
		JWTSecret:   getEnv("JWT_SECRET", ""),
		UploadDir:   getEnv("UPLOAD_DIR", ""),
	}

	validateRequired(AppConfig.DatabaseURL, "DATABASE_URL")
	validateRequired(AppConfig.RedisURL, "REDIS_URL")
	validateRequired(AppConfig.Port, "PORT")
	validateRequired(AppConfig.JWTSecret, "JWT_SECRET")
	validateRequired(AppConfig.UploadDir, "UPLOAD_DIR")
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return fallback
}

func validateRequired(value, key string) {
	if value == "" {
		log.Fatalf("FATAL: environment variable %s is not set", key)
	}
}
