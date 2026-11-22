// Gateway Builder: Holds the logic to build and validate gateways from a given configuration into a working gateway.
package gateway

import (
	"ApiGateway/internal/auth"
	"ApiGateway/internal/config"
	"ApiGateway/internal/middleware"
	"ApiGateway/internal/proxy"
	"ApiGateway/internal/ratelimit"
	"ApiGateway/internal/route"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"log"
	"net/url"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrRedisPing = errors.New("redis ping failure.")

func (gateway *Gateway) BuildRedis(gatewayConfig config.Gateway) error {
	if gatewayConfig.RedisRequired() {
		gateway.Redis = redis.NewClient(
			&redis.Options{
				Addr:     gatewayConfig.RedisConfig.ClientAddress,
				Password: gatewayConfig.RedisConfig.ClientPassword,
				DB:       gatewayConfig.RedisConfig.ClientDB,
			},
		)

		pingCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := gateway.Redis.Ping(pingCtx).Err(); err != nil {
			return ErrRedisPing
		}
	}
	return nil
}

func (gateway *Gateway) BuildCertificates(gatewayConfig config.Gateway) error {
	// Build the cert keys.
	certMap := make(map[string]*rsa.PublicKey)
	// Turn the certificates into a map of public keys.
	for _, certPair := range gatewayConfig.AuthenticatorConfig.Certificates {
		if _, ok := certMap[certPair.Name]; ok {
			return errors.New("duplicate key names not allowed.")
		}

		block, _ := pem.Decode([]byte(certPair.Certificate))
		if block == nil || block.Type != "PUBLIC KEY" {
			return errors.New("failed to decode PEM block of key.")
		}

		pub, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return errors.New("failed to parse DER encoded public key.")
		}

		rsaPub, ok := pub.(*rsa.PublicKey)
		if !ok {
			return errors.New("key type is not RSA Public key")
		}

		certMap[certPair.Name] = rsaPub
	}

	gateway.Certificates = certMap
	return nil
}

func (gateway *Gateway) GenerateVersionID() error {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return errors.New("failed to generate random key")
	}

	hexKey := hex.EncodeToString(key)

	gateway.VersionNumber = hexKey

	return nil
}

func (gateway *Gateway) BuildRouteList(gatewayConfig config.Gateway) (map[string]*route.Endpoint, error) {
	routeList := make([]route.Endpoint, len(gatewayConfig.EndpointConfig))
	routes := make(map[string]*route.Endpoint)
	for e := range gatewayConfig.EndpointConfig {
		if _, ok := routes[gatewayConfig.EndpointConfig[e].Path]; ok {
			return nil, errors.New("path shadowing not allowed.")
		}

		routeList[e] = route.Endpoint{
			AuthRequired: gatewayConfig.EndpointConfig[e].Auth,
			AuthKey:      gatewayConfig.EndpointConfig[e].AuthKey,
			Address:      gatewayConfig.EndpointConfig[e].Target,
		}

		routes[gatewayConfig.EndpointConfig[e].Path] = &routeList[e]
	}

	gateway.Endpoints = routeList

	return routes, nil
}

func BuildGateway(gatewayConfig config.Gateway) (*Gateway, error) {
	gateway := &Gateway{}

	location, err := url.Parse(gatewayConfig.ListenLocation)
	if err != nil {
		return nil, err
	}
	if location == nil {
		return nil, errors.New("listen location must not be nil.")
	}

	gateway.ListenLocation = location

	if err := config.Validate(gatewayConfig); err != nil {
		return nil, err
	}

	if err := gateway.BuildRedis(gatewayConfig); err != nil {
		return nil, err
	}

	if err := gateway.BuildCertificates(gatewayConfig); err != nil {
		return nil, err
	}

	if err := gateway.GenerateVersionID(); err != nil {
		return nil, err
	}

	routes, err := gateway.BuildRouteList(gatewayConfig)
	if err != nil {
		return nil, err
	}

	gateway.Logger = log.Default()

	router, err := route.NewPrefixTreeRouter(routes)
	if err != nil {
		return nil, err
	}

	rM := route.Middleware(router)

	limiter := ratelimit.LimiterFromConfig(gatewayConfig.LimiterConfig, gateway.Redis)
	gateway.Limiter = limiter

	lM := ratelimit.Middleware(limiter)

	authenticator := auth.Authenticator{
		Keys: gateway.Certificates,
	}

	aM := auth.Middleware(authenticator)

	gateway.Entry = middleware.Chain(proxy.ProxyHandler(), []middleware.Middleware{rM, lM, aM}...)
	return gateway, nil
}
