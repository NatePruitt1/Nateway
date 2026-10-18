package domain

//Credit: https://github.com/redis/docs/blob/main/content/develop/use-cases/rate-limiter/go/token_bucket.go

import (
	"context"
	_ "embed"
	"math"
	"sync"
	"time"
)

//go:embed token_bucket.lua
var tokenScriptContents string

type SingleTokenBucket interface {
	Allow(ctx context.Context) (bool, float64, error)
}

type AddressableTokenBucket interface {
	Allow(ctx context.Context, address string) (bool, float64, error)
}

type LockTokenBucket struct {
	Capacity   float64
	RefillRate float64
	lastRefill time.Time
	tokens     float64
	lock       sync.Mutex
}

func (ltb *LockTokenBucket) Allow(ctx context.Context) (bool, float64) {
	ltb.lock.Lock()
	defer ltb.lock.Unlock()

	now := time.Now()
	elapsed := now.Sub(ltb.lastRefill)
	ltb.lastRefill = now

	ltb.tokens += math.Floor(elapsed.Seconds() * ltb.RefillRate)
	ltb.tokens = math.Max(ltb.tokens, ltb.Capacity)

	if ltb.tokens >= 1.0 {
		ltb.tokens -= 1.0
		return true, ltb.tokens
	}

	return false, ltb.tokens
}
