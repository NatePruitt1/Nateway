package ratelimit

import (
	"ApiGateway/internal/config"
	"ApiGateway/internal/middleware"
	"log"
	"net"
	"net/http"

	"github.com/redis/go-redis/v9"
)

type Limiter struct {
	GlobalBucket TokenBucket
	PathBucket   TokenBucket
	HostBucket   TokenBucket
}

func LimiterFromConfig(cfg *config.Limiter, redisClient *redis.Client) *Limiter {
	return &Limiter{
		GlobalBucket: BucketFromConfig(cfg.GlobalBucket, redisClient),
		PathBucket:   BucketFromConfig(cfg.PathBucket, redisClient),
		HostBucket:   BucketFromConfig(cfg.HostBucket, redisClient),
	}
}

func Middleware(limiter *Limiter) middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check the general token bucket.
			if limiter.GlobalBucket != nil {
				ok, _, err := limiter.GlobalBucket.AllowAll(r.Context())

				if err != nil {
					log.Printf("Error encountered allowing request in general token bucket: %v", err)
					http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
					return
				} else if !ok {
					log.Printf("General rate limit exceeded.")
					http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
					return
				}
			}

			if limiter.PathBucket != nil {
				ok, _, err := limiter.PathBucket.Allow(r.Context(), r.URL.Path)
				if err != nil {
					log.Printf("Error encountered allowing request in path token bucket: %v", err)
					http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
					return
				} else if !ok {
					log.Printf("Path rate limit exceeded on %s.", r.URL.Path)
					http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
					return
				}
			}

			if limiter.HostBucket != nil {
				host, _, err := net.SplitHostPort(r.RemoteAddr)

				if err != nil {
					log.Printf("Error encountered parsing request host %s: %v", r.RemoteAddr, err)
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					return
				}

				ok, _, err := limiter.HostBucket.Allow(r.Context(), host)
				if err != nil {
					log.Printf("Error encountered allowing request in ip token bucket: %v", err)
					http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
					return
				} else if !ok {
					log.Printf("IP specific rate limit exceeded for ip %s on path %s", r.RemoteAddr, r.URL.Path)
					http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
