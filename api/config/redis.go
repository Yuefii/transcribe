package config

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client
var ctx = context.Background()

func InitRedis() {
	redisURL := AppConfig.RedisURL

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
