package ratelimit

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type RedisTokenBucket struct {
	capacity   float64
	refillRate float64 // Tokens per second
	script     *redis.Script
	client     *redis.Client
}

func NewRedisTokenBucket(capacity, refillRate float64, client *redis.Client) *RedisTokenBucket {
	rb := &RedisTokenBucket{
		capacity:   capacity,
		refillRate: refillRate,
		script:     redis.NewScript(tokenScriptContents),
		client:     client,
	}

	return rb
}

func (tb *RedisTokenBucket) Allow(ctx context.Context, key string) (bool, float64, error) {
	args := []interface{}{
		tb.capacity,
		tb.refillRate,
	}

	result, err := tb.script.Run(ctx, tb.client, []string{key}, args...).Int64Slice()
	if err != nil {
		return false, 0, fmt.Errorf("Error attempting to run token bucket script: %w", err)
	}

	allowed := result[0] == 1
	remaining := float64(result[1])

	return allowed, remaining, nil
}

func (tb *RedisTokenBucket) AllowAll(ctx context.Context) (bool, float64, error) {
	return tb.Allow(ctx, "global:api")
}
