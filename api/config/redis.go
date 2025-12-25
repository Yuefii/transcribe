package config

import (
	"context"
	"transcribe/pkg/logger"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client
var ctx = context.Background()

func InitRedis() {
	redisURL := AppConfig.RedisURL

	opt, err := redis.ParseURL(redisURL)

	if err != nil {
		logger.Log.Fatal("failed to parse REDIS_URL: ", err)
	}

	RedisClient = redis.NewClient(opt)

	if _, err := RedisClient.Ping(ctx).Result(); err != nil {
		logger.Log.Fatal("failed to connect Redis: ", err)
	}

	logger.Log.Info("redis connected successfully")
}
