package config

import (
	"context"
	"log"
	"os"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client
var ctx = context.Background()

func InitRedis() {
	redisURL := os.Getenv("REDIS_URL")

	if redisURL == "" {
		log.Fatal("environment variable REDIS_URL is not set")
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatal("failed to parse REDIS_URL: ", err)
	}

	RedisClient = redis.NewClient(opt)

	if _, err := RedisClient.Ping(ctx).Result(); err != nil {
		log.Fatal("failed to connect Redis: ", err)
	}

	log.Println("redis connected successfully")
}
