package main

import (
	apigateway "ApiGateway/gateway"
	"fmt"
	"net"
	"net/http"
	"time"

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

	bucketCfg := apigateway.TokenBucketCFG{
		Capacity:   25000,
		RefillRate: 10000,
		Client:     client,
	}

	bucket := apigateway.NewTokenBucket(bucketCfg)
	limiter := apigateway.RateLimiter{Bucket: bucket}

	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Printf("Error opening interface: %w", err)
		return
	}

	server := http.Server{
		Handler:           http.HandlerFunc(limiter.CheckRequestAllowed),
		ReadHeaderTimeout: 2 * time.Second,
	}

	server.Serve(listener)
}
