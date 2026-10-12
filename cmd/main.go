package main

import (
	"ApiGateway/internal/domain"
	"log"
	"net"

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

	longestMatchLimitLoader, err := domain.NewLongestMatchRouterLoader("./longest_match_router_cfg.json")
	if err != nil {
		panic(err)
	}

	longestMatchLimitLoader.Watch()

	gateway := &domain.Gateway{
		Limiter: redisRateLimitLoader,
		Router:  longestMatchLimitLoader,
	}

	listener, err := net.Listen("tcp", "localhost:8000")
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	log.Println("Server started on port 8000")
	gateway.StartGateway(listener)
}
