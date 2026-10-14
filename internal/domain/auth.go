package domain

import (
	"crypto/rsa"
	"net/http"
	"strings"
)

type Authenticator struct {
	keys map[string]*rsa.PrivateKey
}

func AuthenticatorMiddleware(auth Authenticator) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authstr := r.Header.Get("Authorization")
			if authstr != "" {
				splauth := strings.Split(authstr, " ")
				if len(splauth) >= 2 && splauth[0] == "Bearer" {
					// MOVE TO EXTRACT TOKEN FUNCTION.
				}
			}
		})
	}
}
