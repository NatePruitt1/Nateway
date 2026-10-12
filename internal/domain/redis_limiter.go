package domain

import (
	"encoding/json"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/redis/go-redis/v9"
)

// The config for the redis rate limiter.
// Includes provisions to allow the redis rate limiter to say how long it should wait
// before expiring individual addressed buckets.
type RedisRateLimiterCFG struct {
	GenCapacity   int     `json:"gen_capacity"`
	GenRefillRate float64 `json:"gen_refill_rate"`

	AddrCapacity      int     `json:"addr_capacity"`
	AddrRefillRate    float64 `json:"addr_refill_rate"`
	AddrExpireSeconds int     `json:"addr_expire_time"`
}

type RedisRateLimiterLoader struct {
	redisLimiter atomic.Pointer[RedisRateLimiter]
	redisClient  *redis.Client
	fileLocation string
}

func (rll *RedisRateLimiterLoader) Pin() *RedisRateLimiter {
	return rll.redisLimiter.Load()
}

func (rll *RedisRateLimiterLoader) Load() error {
	content, err := os.ReadFile(rll.fileLocation)
	if err != nil {
		return err
	}

	var newCFG RedisRateLimiterCFG

	err = json.Unmarshal(content, &newCFG)

	newRl := NewRedisRateLimiter(newCFG, rll.redisClient)
	rll.redisLimiter.Store(newRl)
}

type RedisRateLimiter struct {
	generalBucket *TokenBucket
	ipBucket      *TokenBucket
}

func NewRedisRateLimiter(cfg RedisRateLimiterCFG, client *redis.Client) *RedisRateLimiter {
	genBucketCfg := TokenBucketCFG{
		Capacity:   cfg.GenCapacity,
		RefillRate: cfg.GenRefillRate,
		Client:     client,
	}

	addrBucketCfg := TokenBucketCFG{
		Capacity:   cfg.AddrCapacity,
		RefillRate: cfg.GenRefillRate,
		Client:     client,
	}

	genBucket := NewTokenBucket(genBucketCfg)
	addrBucket := NewTokenBucket(addrBucketCfg)

	newRateLim := RedisRateLimiter{
		generalBucket: genBucket,
		ipBucket:      addrBucket,
	}

	return &newRateLim
}

func (rl *RedisRateLimiter) CheckGeneralRequestAllowed(r *http.Request) (bool, error) {
	ctx := r.Context()
	allowed, _, err := rl.generalBucket.Allow(ctx, "global:api")
	return allowed, err
}

func (rl *RedisRateLimiter) CheckIpRequestAllowed(r *http.Request) (bool, error) {
	ctx := r.Context()
	allowed, _, err := rl.ipBucket.Allow(ctx, "ip:"+r.RemoteAddr)
	return allowed, err
}
