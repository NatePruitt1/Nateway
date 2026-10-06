package apigateway

import (
	"net/http"
)

type RateLimiter struct {
	Bucket *TokenBucket
	Router *Routes
}

func (rl *RateLimiter) CheckRequestAllowed(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	allowed, _, err := rl.Bucket.Allow(ctx, "global:api")
	if err != nil {
		w.WriteHeader(500)
	} else if !allowed {
		w.WriteHeader(503)
	} else {
		// Pass ahead to the request to the routing, so that we can initiate and pass along the request.
		path := r.URL.Path
		route := rl.Router.GetLongestPrefixRoute(path)
		if route != nil {
			// Send to a route. Maybe plan classes better so that the limiter is not the exposed resource?
		}
	}
}
