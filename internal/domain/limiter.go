package domain

import (
	"log"
	"net"
	"net/http"
)

func RateLimiterMiddleware(gb SingleTokenBucket, ipb AddressableTokenBucket) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check the general token bucket.
			ok, _, err := gb.Allow(r.Context())

			if err != nil {
				log.Printf("Error encountered allowing request in general token bucket: %v", err)
				http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
				return
			} else if !ok {
				log.Printf("General rate limit exceeded for %s", r.URL.Path)
				http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
				return
			}

			host, _, err := net.SplitHostPort(r.RemoteAddr)

			if err != nil {
				log.Printf("Error encountered parsing request host %s: %v", r.RemoteAddr, err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			ok, _, err = ipb.Allow(r.Context(), host)
			if err != nil {
				log.Printf("Error encountered allowing request in ip token bucket: %v", err)
				http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
				return
			} else if !ok {
				log.Printf("IP specific rate limit exceeded for ip %s on path %s", r.RemoteAddr, r.URL.Path)
				http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
