package domain

import (
	"log"
	"net/http"

	"github.com/redis/go-redis/v9"
)

func RedisRateLimiterMiddleware(gb *TokenBucket, ipb *TokenBucket, client *redis.Client) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check the general token bucket.
			ok, _, err := gb.Allow(r.Context(), "global:api")
			if !ok || err != nil {
				log.Printf("General rate limit exceeded for %s", r.URL.Path)
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}

			ok, _, err = ipb.Allow(r.Context(), "ip:"+r.RemoteAddr)
			if !ok || err != nil {
				log.Printf("IP specific rate limit exceeded for ip %s on path %s", r.RemoteAddr, r.URL.Path)
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func LockerLimiterMiddleware(gb *LockTokenBucket) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check the general token bucket.
			ok, _ := gb.Allow(r.Context())
			if !ok {
				log.Printf("General rate limit exceeded for %s", r.URL.Path)
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
