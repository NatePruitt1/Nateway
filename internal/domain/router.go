package domain

import (
	"context"
	"log"
	"net/http"
)

type Router interface {
	GetEndpoint(r *http.Request) (*Endpoint, error)
}

type Routes struct {
	Endpoints map[string]Endpoint
}

func NewRoutes(cfg *GatewayConfig) *Routes {
	m := make(map[string]Endpoint, len(cfg.Server.Locations))
	for loc := range cfg.Server.Locations {
		epcfg := cfg.Server.Locations[loc]
		m[epcfg.Path] = Endpoint{
			Address:      epcfg.Target,
			AuthKey:      epcfg.AuthKey,
			AuthRequired: epcfg.Auth,
		}
	}

	return &Routes{
		Endpoints: m,
	}
}

func (routes *Routes) GetEndpoint(r *http.Request) (*Endpoint, error) {
	// Search for the longest key match
	matchLen := -1
	var longestMatch *Endpoint = nil
	for key, endpoint := range routes.Endpoints {
		currMatch := getMatchLength(key, r.URL.Path)
		if currMatch > matchLen {
			matchLen = currMatch
			longestMatch = &endpoint
		}
	}

	if longestMatch == nil {
		return nil, NoRouteFoundErr
	} else {
		return longestMatch, nil
	}
}

func getMatchLength(s, t string) int {
	for i := 0; i < min(len(s), len(t)); i++ {
		if s[i] != t[i] {
			return i
		}
	}
	return min(len(s), len(t))
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
