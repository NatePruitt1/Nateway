package domain

//Credit: https://github.com/redis/docs/blob/main/content/develop/use-cases/rate-limiter/go/token_bucket.go

import (
	"context"
	"crypto/sha1"
	_ "embed"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

//go:embed token_bucket.lua
var tokenScriptContents string

type TokenBucketCFG struct {
	Capacity   int
	RefillRate float64
	Client     *redis.Client
}

type TokenBucket struct {
	client       *redis.Client
	capacity     int
	refillRate   float64 // Tokens per second
	scriptSHA    string
	scriptLoaded bool
}

func NewTokenBucket(cfg TokenBucketCFG) *TokenBucket {
	if cfg.Capacity == 0 {
		cfg.Capacity = 10
	}
	if cfg.RefillRate == 0 {
		cfg.RefillRate = 1.0
	}

	h := sha1.New()
	h.Write([]byte(tokenScriptContents))
	sha := fmt.Sprintf("%x", h.Sum(nil))

	return &TokenBucket{
		client:     cfg.Client,
		capacity:   cfg.Capacity,
		refillRate: cfg.RefillRate,
		scriptSHA:  sha,
	}
}

func (tb *TokenBucket) ensureScriptLoaded(ctx context.Context) {
	if !tb.scriptLoaded {
		sha, err := tb.client.ScriptLoad(ctx, tokenScriptContents).Result()
		if err == nil {
			tb.scriptSHA = sha
			tb.scriptLoaded = true
		}
	}
}

func (tb *TokenBucket) Allow(ctx context.Context, key string) (bool, float64, error) {
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
