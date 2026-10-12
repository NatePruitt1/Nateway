package main

import (
	"ApiGateway/internal/domain"

	"github.com/redis/go-redis/v9"
)

func main() {
	client := redis.NewClient(
		&redis.Options{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
		},
	)

	redisRateLimitLoader, err := domain.NewRedisRateLimiterLoader("./redis_limiter_cfg.json", client)
	if err != nil {
		panic(err)
	}

	redisRateLimitLoader.Watch()

	for {
	}
}
