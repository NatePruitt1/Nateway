package domain

import (
	"crypto/rsa"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type Authenticator struct {
	Keys map[string]*rsa.PrivateKey
}

func extractToken(header string) string {
	if header == "" {
		return header
	} else {
		splauth := strings.Split(header, " ")
		if len(splauth) == 2 && splauth[0] == "Bearer" {
			return splauth[0]
		}
	}

	return ""
}

func (auth *Authenticator) verifyToken(token, keyname string) bool {
	var claims jwt.RegisteredClaims

	// Checks the standard fields for you.
	parsedToken, err := jwt.ParseWithClaims(token, &claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", t.Header["alg"])
		}
		return auth.Keys[keyname], nil
	})

	if err != nil {
		return false
	}

	if !parsedToken.Valid {
		return false
	}

	return true
}

func AuthenticatorMiddleware(auth Authenticator) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			endpoint := r.Context().Value("endpoint").(*Endpoint)
			if endpoint == nil {
				http.Error(w, "Error retrieving endpoint routing information.", http.StatusInternalServerError)
				return
			} else if endpoint.AuthRequired {
				token := extractToken(r.Header.Get("Authorization"))
				if token != "" {
					valid := auth.verifyToken(token, endpoint.AuthKey)
					if !valid {
						http.Error(w, "Invalid token", http.StatusUnauthorized)
						return
					} else {
						next.ServeHTTP(w, r)
					}
				} else {
					http.Error(w, "Token not provided.", http.StatusUnauthorized)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
