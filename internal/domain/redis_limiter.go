package domain

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/fsnotify/fsnotify"
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

// Manages the hot loading for the redis rate limiter.
// Is Pinnable, and covers the RateLimiter interface.
type RedisRateLimiterLoader struct {
	redisLimiter atomic.Pointer[RedisRateLimiter]
	redisClient  *redis.Client
	fileLocation string
}

func NewRedisRateLimiterLoader(fileLocation string, client *redis.Client) (*RedisRateLimiterLoader, error) {
	limitLoader := &RedisRateLimiterLoader{
		redisClient:  client,
		fileLocation: fileLocation,
	}

	limitLoader.Load()

	return limitLoader, nil
}

func (rll *RedisRateLimiterLoader) Pin() *RedisRateLimiter {
	return rll.redisLimiter.Load()
}

// Loads the configuration stored in the file location in the redis rate limiter loader,
// and then creates a new redis rate limiter with the configuration and performs
// an atomic update to the redis rate limiter loader's pointer.
func (rll *RedisRateLimiterLoader) Load() error {
	content, err := os.ReadFile(rll.fileLocation)
	if err != nil {
		return err
	}

	var newCFG RedisRateLimiterCFG

	err = json.Unmarshal(content, &newCFG)

	newRl := NewRedisRateLimiter(newCFG, rll.redisClient)
	rll.redisLimiter.Store(newRl)

	log.Printf("Redis rate limiter loaded, new cfg\n%+v", newRl)

	return nil
}

// Watch for configuration changes, and when necessary, reload
// the redis rate limiter.
func (rll *RedisRateLimiterLoader) Watch() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	go func() {
		defer watcher.Close()
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				log.Println("Event:", event)
				if event.Has(fsnotify.Write) {
					log.Println("File changed, reloading...")
					rll.Load()
				}
			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
			}
		}
	}()

	err = watcher.Add(rll.fileLocation)
	if err != nil {
		log.Fatal(err)
	}
	return err
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
