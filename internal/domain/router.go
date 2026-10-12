package domain

import (
	"errors"
	"net/http"
)

var NoRouteFoundErr error = errors.New("no route found")

type Endpoint struct {
	AuthRequired bool   `json:"auth_required"`
	AuthKey      string `json:"auth_key"`
	Address      string `json:"address"`
}

type Router interface {
	GetEndpoint(r *http.Request) (*Endpoint, error)
}
