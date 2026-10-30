package domain

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type RedisTokenBucketInternal struct {
	client     *redis.Client
	capacity   float64
	refillRate float64 // Tokens per second
	script     *redis.Script
}

type RedisTokenBucketGeneral struct {
	RedisTokenBucketInternal
}

type RedisTokenBucketAddressable struct {
	RedisTokenBucketInternal
}

func NewRedisTokenBucket(capacity, refillRate float64, client *redis.Client) *RedisTokenBucketInternal {
	rb := &RedisTokenBucketInternal{
		client:     client,
		capacity:   capacity,
		refillRate: refillRate,
		script:     redis.NewScript(tokenScriptContents),
	}

	return rb
}

func (tb *RedisTokenBucketInternal) Allow(ctx context.Context, key string) (bool, float64, error) {
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

func (tb *RedisTokenBucketGeneral) Allow(ctx context.Context) (bool, float64, error) {
	return tb.RedisTokenBucketInternal.Allow(ctx, "global:api")
}

func (tb *RedisTokenBucketAddressable) Allow(ctx context.Context, address string) (bool, float64, error) {
	return tb.RedisTokenBucketInternal.Allow(ctx, "ip:"+address)
}
