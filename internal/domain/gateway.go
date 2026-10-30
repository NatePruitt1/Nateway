package domain

import (
	"crypto/rsa"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/fsnotify/fsnotify"
	"github.com/redis/go-redis/v9"
	"go.yaml.in/yaml/v4"
)

type Gateway struct {
	Config *GatewayConfig
	Router Router
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

type BucketAlgCFG struct {
	ClientAddress  string `yaml:"address"`
	ClientPassword string `yaml:"password"`
	ClientDB       int    `yaml:"db"`
}

type GatewayConfig struct {
	Limits struct {
		Algorithm   BucketAlgCFG `yaml:"algorithm"`
		General     BucketCFG    `yaml:"general"`
		IpBucketCFG BucketCFG    `yaml:"ip"`
	} `yaml:"limits"`
	Server struct {
		ListenLocation string        `yaml:"listen"`
		Locations      []EndpointCFG `yaml:"locations"`
	} `yaml:"gateway"`
}

func LoadGateway(filepath string) (*Gateway, error) {
	cfg := &GatewayConfig{}

	txt, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	txt = []byte(os.ExpandEnv(string(txt)))

	err = yaml.Unmarshal(txt, cfg)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(
		&redis.Options{
			Addr:     cfg.Limits.Algorithm.ClientAddress,
			Password: cfg.Limits.Algorithm.ClientPassword,
			DB:       cfg.Limits.Algorithm.ClientDB,
		},
	)

	genBucket := &RedisTokenBucketGeneral{
		*NewRedisTokenBucket(float64(cfg.Limits.General.Capacity), cfg.Limits.General.Refill, client),
	}

	ipBucket := &RedisTokenBucketAddressable{
		*NewRedisTokenBucket(float64(cfg.Limits.IpBucketCFG.Capacity), cfg.Limits.IpBucketCFG.Refill, client),
	}

	router := NewPrefixTreeRouter(cfg)

	key1, err := loadPrivateKey("cert.pem")
	if err != nil {
		return nil, err
	}

	keys := map[string]*rsa.PublicKey{"key1": key1}

	auth := Authenticator{
		Keys: keys,
	}

	middlewares := []Middleware{RateLimiterMiddleware(genBucket, ipBucket), RouterMiddleware(router), AuthenticatorMiddleware(auth)}

	entry := Chain(ProxyHandler(), middlewares...)

	return &Gateway{
		Config: cfg,
		Router: router,
		Entry:  entry,
	}, nil
}

func WatchGateway(filepath string, gateway *atomic.Pointer[Gateway]) {

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	// 2. Start a goroutine to listen for filesystem events
	go func() {
		defer watcher.Close()

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				// Log the raw event info
				log.Println("Event received:", event)

				// Check specific event types using .Has()
				if event.Has(fsnotify.Write) {
					log.Println("Modified file:", event.Name)
					newgtwy, err := LoadGateway(filepath)
					if err != nil {
						log.Printf("Failed gateway reload: %v\n", err)
					} else if newgtwy == nil {
						log.Printf("Faild to load gateway on reload.")
					} else {
						gateway.Store(newgtwy)
					}
				}
				if event.Has(fsnotify.Create) {
					log.Println("Created file:", event.Name)
				}
				if event.Has(fsnotify.Remove) {
					log.Println("Removed file:", event.Name)
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("Error:", err)
			}
		}
	}()

	// 3. Specify the directory or file path to watch
	err = watcher.Add(filepath)
	if err != nil {
		log.Fatal(err)
	}

}
