package domain

import (
	"context"
	"crypto/sha1"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisTokenBucketInternal struct {
	client       *redis.Client
	capacity     float64
	refillRate   float64 // Tokens per second
	scriptSHA    string
	scriptLoaded bool
}

type RedisTokenBucketGeneral struct {
	RedisTokenBucketInternal
}

type RedisTokenBucketAddressable struct {
	RedisTokenBucketInternal
}

func NewRedisTokenBucket(capacity, refillRate float64, client *redis.Client) *RedisTokenBucketInternal {
	h := sha1.New()
	h.Write([]byte(tokenScriptContents))
	sha := fmt.Sprintf("%x", h.Sum(nil))

	return &RedisTokenBucketInternal{
		client:     client,
		capacity:   capacity,
		refillRate: refillRate,
		scriptSHA:  sha,
	}
}

func (tb *RedisTokenBucketInternal) ensureScriptLoaded(ctx context.Context) {
	if !tb.scriptLoaded {
		sha, err := tb.client.ScriptLoad(ctx, tokenScriptContents).Result()
		if err == nil {
			tb.scriptSHA = sha
			tb.scriptLoaded = true
		}
	}
}

func (tb *RedisTokenBucketInternal) Allow(ctx context.Context, key string) (bool, float64, error) {
	tb.ensureScriptLoaded(ctx)

	now := float64(time.Now().UnixMicro()) / 1e6

	args := []interface{}{
		tb.capacity,
		tb.refillRate,
		now,
	}

	result, err := tb.client.EvalSha(ctx, tb.scriptSHA, []string{key}, args...).Int64Slice()
	if err != nil {
		result, err = tb.client.Eval(ctx, tokenScriptContents, []string{key}, args...).Int64Slice()
		if err != nil {
			return false, 0, fmt.Errorf("token bucket eval failed: %w", err)
		}

		tb.scriptLoaded = false
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
