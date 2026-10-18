package domain

import (
	"context"
	"log"
	"net/http"
)

type Router interface {
	GetEndpoint(r *http.Request) (*Endpoint, error)
}

func RouterMiddleware(routes Router) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			endpoint, err := routes.GetEndpoint(r)
			if err != nil || endpoint == nil {
				log.Printf("No route found for %s", r.URL.Path)
				http.Error(w, "Endpoint not found.", http.StatusNotFound)
				return
			}

			ctx := context.WithValue(r.Context(), "endpoint", endpoint)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
