package route

import (
	"errors"
)

var ErrNoRoute error = errors.New("no route found")

type Endpoint struct {
	AuthRequired bool   `json:"auth_required"`
	AuthKey      string `json:"auth_key"`
	Address      string `json:"address"`
}
