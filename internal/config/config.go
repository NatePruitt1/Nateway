package config

import "errors"

type BucketType string

const (
	RedisBucket  BucketType = "redis"
	MemoryBucket BucketType = "mem"
)

type Bucket struct {
	Capacity int        `yaml:"capacity"`
	Refill   float64    `yaml:"refill"`
	Type     BucketType `yaml:"bucket_type"`
}

type Limiter struct {
	GlobalBucket *Bucket `yaml:"global"`
	PathBucket   *Bucket `yaml:"path"`
	HostBucket   *Bucket `yaml:"host"`
}

type Endpoint struct {
	Path    string `yaml:"path"`
	Target  string `yaml:"target"`
	Auth    bool   `yaml:"auth_required"`
	AuthKey string `yaml:"auth_key"`
}

type Authenticator struct {
	Certificates []struct {
		Name        string `yaml:"name"`
		Certificate string `yaml:"cert"`
	} `yaml:"certificates"`
}

type Redis struct {
	ClientAddress  string `yaml:"address"`
	ClientPassword string `yaml:"password"`
	ClientDB       int    `yaml:"db"`
}

type Gateway struct {
	RedisConfig         *Redis         `yaml:"redis"`
	LimiterConfig       *Limiter       `yaml:"limiter"`
	AuthenticatorConfig *Authenticator `yaml:"certificates"`
	EndpointConfig      []Endpoint     `yaml:"endpoints"`
	ListenLocation      string         `yaml:"listen"`
}

var ErrNoRedis = errors.New("no redis config provided, but the config requires it.")
var ErrNoEndpoints = errors.New("no endpoints provided.")
var ErrNoTarget = errors.New("no target provided.")
var ErrNoAuthConfig = errors.New("no auth config provided, but endpoints require it.")
var ErrUnmatchedKey = errors.New("no key matching endpoint config in authenticator.")

// We need two functions, one that validates if the gateway would be able to build, and then one that builds it.
// The builder function should call the validator first, to ensure we will not have any issues with the build.
func Validate(gatewayConfig Gateway) error {
	redisRequired, err := ValidateLimiterConfig(gatewayConfig.LimiterConfig)
	if err != nil {
		return err
	}

	if redisRequired {
		if gatewayConfig.RedisConfig == nil {
			return ErrNoRedis
		}
	}

	// Go through the endpoints and validate them. Build a list of keys that we need.
	keysNeeded := make([]string, 0)
	if len(gatewayConfig.EndpointConfig) == 0 {
		return ErrNoEndpoints
	}

	for e := range gatewayConfig.EndpointConfig {
		if gatewayConfig.EndpointConfig[e].Target == "" {
			return ErrNoTarget
		}

		if gatewayConfig.EndpointConfig[e].AuthKey != "" {
			keysNeeded = append(keysNeeded, gatewayConfig.EndpointConfig[e].AuthKey)
		}
	}

	if len(keysNeeded) > 0 && (gatewayConfig.AuthenticatorConfig == nil || len(gatewayConfig.AuthenticatorConfig.Certificates) == 0) {
		return ErrNoAuthConfig
	}

	// Validate the authenticator config, and a simple list of key names to check against the endpoint config.
	if len(keysNeeded) > 0 && gatewayConfig.AuthenticatorConfig != nil {
		for k := range keysNeeded {
			matched := false
			for c := range gatewayConfig.AuthenticatorConfig.Certificates {
				if gatewayConfig.AuthenticatorConfig.Certificates[c].Name == keysNeeded[k] {
					matched = true
				}
			}

			if !matched {
				return ErrUnmatchedKey
			}
		}
	}

	return nil
}

func ValidateLimiterConfig(limiterConfig *Limiter) (bool, error) {
	redisRequired := false

	// Check the buckets first. If we encounter a redis bucket, check that we have a redis config as well.
	if limiterConfig != nil {
		if limiterConfig.GlobalBucket != nil {
			if err := ValidateBucketConfig(limiterConfig.GlobalBucket); err != nil {
				return false, err
			}

			if limiterConfig.GlobalBucket.Type == RedisBucket {
				redisRequired = true
			}
		}

		if limiterConfig.HostBucket != nil {
			if err := ValidateBucketConfig(limiterConfig.HostBucket); err != nil {
				return false, err
			}

			if limiterConfig.HostBucket.Type == RedisBucket {
				redisRequired = true
			}
		}

		if limiterConfig.PathBucket != nil {
			if err := ValidateBucketConfig(limiterConfig.PathBucket); err != nil {
				return false, err
			}

			if limiterConfig.PathBucket.Type == RedisBucket {
				redisRequired = true
			}
		}
	}

	return redisRequired, nil
}

var ErrLowBucketCapacity = errors.New("bucket capacity must be >= 1.")
var ErrLowBucketRefill = errors.New("bucket refill must be > 0.")
var ErrInvalidBucketType = errors.New("bucket type invalid. Must be \"redis\" or \"mem\".")

func ValidateBucketConfig(bucketConfig *Bucket) error {
	if bucketConfig.Capacity < 1 {
		return ErrLowBucketCapacity
	}

	if bucketConfig.Refill <= 0 {
		return ErrLowBucketCapacity
	}

	if bucketConfig.Type != RedisBucket && bucketConfig.Type != MemoryBucket {
		return ErrInvalidBucketType
	}

	return nil
}

func (gtwyCfg Gateway) RedisRequired() bool {
	redisReq := false
	if gtwyCfg.LimiterConfig != nil {
		if gtwyCfg.LimiterConfig.GlobalBucket != nil && gtwyCfg.LimiterConfig.GlobalBucket.Type == "redis" {
			redisReq = true
		}

		if gtwyCfg.LimiterConfig.HostBucket != nil && gtwyCfg.LimiterConfig.HostBucket.Type == "redis" {
			redisReq = true
		}

		if gtwyCfg.LimiterConfig.PathBucket != nil && gtwyCfg.LimiterConfig.PathBucket.Type == "redis" {
			redisReq = true
		}
	}

	return redisReq
}
