package ratelimit

//Credit: https://github.com/redis/docs/blob/main/content/develop/use-cases/rate-limiter/go/token_bucket.go

import (
	"ApiGateway/internal/config"
	"context"
	_ "embed"

	"github.com/redis/go-redis/v9"
)

//go:embed token_bucket.lua
var tokenScriptContents string

type TokenBucket interface {
	Allow(ctx context.Context, address string) (bool, float64, error)
	AllowAll(ctx context.Context) (bool, float64, error)
}

func BucketFromConfig(bucketConfig *config.Bucket, client *redis.Client) TokenBucket {
	if bucketConfig == nil {
		return nil
	}

	if bucketConfig.Type == config.RedisBucket {
		return NewRedisTokenBucket(float64(bucketConfig.Capacity), bucketConfig.Refill, client)
	} else {
		panic("Bucket type not implemented!")
	}
}
