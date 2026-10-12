package domain

import "net/http"

type RateLimiter interface {
	CheckGeneralRequestAllowed(r *http.Request) (bool, error)
	CheckIpRequestAllowed(r *http.Request) (bool, error)
}
