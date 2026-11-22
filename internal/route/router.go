package route

import (
	"ApiGateway/internal/middleware"
	"context"
	"log"
	"net/http"
	"regexp"
)

type Router interface {
	GetEndpoint(r *http.Request) (*Endpoint, error)
}

const pathValidationPattern = `^(/|(/(\.?[\p{L}\p{N}\p{M}])+)+)$`

func getPathValidationPattern() *regexp.Regexp {
	return regexp.MustCompile(pathValidationPattern)
}

func Middleware(routes Router) middleware.Middleware {
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
