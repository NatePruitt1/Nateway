package gateway

import (
	"ApiGateway/internal/config"
	"ApiGateway/internal/ratelimit"
	"ApiGateway/internal/route"
	"crypto/rsa"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/goccy/go-yaml"
	"github.com/redis/go-redis/v9"
)

var ErrNoListenLocation = errors.New("No listening target provided.")

// Gateway stores the information for the middlewares to complete the request path.
//
// The global information that we need from the gateway is:
// *Redis*: Stores a redis client for the middlewares to use.
// *Logger*: Stores a log client that acts like a file descriptor that we can write to.
// *Endpoints*: Stores a list of endpoints, with their constructed target list.
// *Version Number*: A hash generated at the beginning to tell which iteration we are running.
//
// The gateway also needs a WaitGroup semaphore. This is how we can tell if the gateway is safe to fully close.
// If we need to close this gateway iteration, we shut down the incoming interface, we wait until the WG is 0, then we close all
// the resources. This is to ensure the resources do not close while in use.
type Gateway struct {
	Redis          *redis.Client
	Logger         *log.Logger
	Endpoints      []route.Endpoint
	Certificates   map[string]*rsa.PublicKey
	VersionNumber  string
	Limiter        *ratelimit.Limiter
	ListenLocation *url.URL

	Entry http.Handler
}

func LoadGateway(filepath string) (*Gateway, error) {
	raw, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	expanded := os.ExpandEnv(string(raw))

	var cfg config.Gateway
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal yaml: %w", err)
	}

	return BuildGateway(cfg)
}
