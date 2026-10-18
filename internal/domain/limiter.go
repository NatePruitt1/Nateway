package domain

import (
	"log"
	"net/http"
)

func RateLimiterMiddleware(gb SingleTokenBucket, ipb AddressableTokenBucket) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check the general token bucket.
			ok, _, err := gb.Allow(r.Context())
			if !ok || err != nil {
				log.Printf("General rate limit exceeded for %s", r.URL.Path)
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}

			ok, _, err = ipb.Allow(r.Context(), r.RemoteAddr)
			if !ok || err != nil {
				log.Printf("IP specific rate limit exceeded for ip %s on path %s", r.RemoteAddr, r.URL.Path)
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
