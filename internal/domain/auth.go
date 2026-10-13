package domain

import "net/http"

type Authenicator interface {
	Authenticate(r *http.Request, authKey string) (bool, error)
}
