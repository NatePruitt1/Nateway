package domain

import (
	"net/http"
	"os"
	"sync/atomic"

	"github.com/redis/go-redis/v9"
	"go.yaml.in/yaml/v4"
)

type Gateway struct {
	Config *GatewayConfig
	Router *Routes
	Entry  http.Handler
}

type BucketCFG struct {
	Capacity int     `yaml:"capacity"`
	Refill   float64 `yaml:"refill"`
}

type EndpointCFG struct {
	Path    string `yaml:"path"`
	Target  string `yaml:"target"`
	Auth    bool   `yaml:"auth_required"`
	AuthKey string `yaml:"auth_key"`
}

type GatewayConfig struct {
	Limits struct {
		General     BucketCFG `yaml:"general"`
		IpBucketCFG BucketCFG `yaml:"ip"`
	} `yaml:"limits"`
	Server struct {
		ListenPort int           `yaml:"listen"`
		Locations  []EndpointCFG `yaml:"locations"`
	} `yaml:"gateway"`
}

func LoadGateway(filepath string, client *redis.Client) (*Gateway, error) {
	cfg := &GatewayConfig{}

	txt, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(txt, cfg)
	if err != nil {
		return nil, err
	}

	// genCfg := TokenBucketCFG{
	// 	Capacity:   cfg.Limits.General.Capacity,
	// 	RefillRate: cfg.Limits.General.Refill,
	// 	Client:     client,
	// }
	// ipCfg := TokenBucketCFG{
	// 	Capacity:   cfg.Limits.IpBucketCFG.Capacity,
	// 	RefillRate: cfg.Limits.IpBucketCFG.Refill,
	// 	Client:     client,
	// }

	// genBucket := NewTokenBucket(genCfg)
	// ipBucket := NewTokenBucket(ipCfg)

	router := NewRoutes(cfg)

	entry := Chain(ProxyHandler(), LongestMatchRouterMiddleware(*router))

	return &Gateway{
		Config: cfg,
		Router: router,
		Entry:  entry,
	}, nil
}

func WatchGateway(filepath string, gateway *atomic.Pointer[Gateway]) {

}
