package domain

import (
	"crypto/rsa"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type Authenticator struct {
	Keys map[string]*rsa.PublicKey
}

func extractToken(header string) string {
	if header == "" {
		return header
	} else {
		splauth := strings.Split(header, " ")
		if len(splauth) == 2 && splauth[0] == "Bearer" {
			return splauth[1]
		}
	}

	return ""
}

func (auth *Authenticator) verifyToken(token, keyname string) (bool, error) {
	var claims jwt.RegisteredClaims

	// Checks the standard fields for you.
	parsedToken, err := jwt.ParseWithClaims(token, &claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", t.Header["alg"])
		}
		return auth.Keys[keyname], nil
	})

	if err != nil {
		return false, err
	}

	if !parsedToken.Valid {
		return false, nil
	}

	return true, nil
}

func AuthenticatorMiddleware(auth Authenticator) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			endpoint := r.Context().Value("endpoint").(*Endpoint)
			if endpoint == nil {
				http.Error(w, "Error retrieving endpoint routing information.", http.StatusInternalServerError)
				log.Print("Failed to retrieve endpoint.\n")
				return
			} else if endpoint.AuthRequired {
				token := extractToken(r.Header.Get("Authorization"))
				if token != "" {
					valid, err := auth.verifyToken(token, endpoint.AuthKey)
					if err != nil {
						log.Printf("Invalid token error: %v\n", err)
					}
					if !valid {
						http.Error(w, "Invalid token", http.StatusUnauthorized)
						log.Print("Invalid Token\n")
						return
					} else {
						next.ServeHTTP(w, r)
					}
				} else {
					http.Error(w, "Token not provided.", http.StatusUnauthorized)
					log.Print("Token not provided.\n")
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func loadPrivateKey(path string) (*rsa.PublicKey, error) {
	pemBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	rsaKey, err := jwt.ParseRSAPublicKeyFromPEM(pemBytes)

	return rsaKey, nil
}
