package apigateway

import (
	"fmt"
	"net/http"
)

type RateLimiter struct {
	Bucket *TokenBucket
}

func (rl *RateLimiter) CheckRequestAllowed(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	allowed, _, err := rl.Bucket.Allow(ctx, "global:api")
	if err != nil {
		w.WriteHeader(500)
	} else if !allowed {
		w.WriteHeader(503)
	} else {
		w.WriteHeader(200)
		fmt.Fprint(w, "Hello!")
	}
}
