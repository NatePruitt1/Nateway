package domain

//Credit: https://github.com/redis/docs/blob/main/content/develop/use-cases/rate-limiter/go/token_bucket.go

import (
	"context"
	_ "embed"
)

//go:embed token_bucket.lua
var tokenScriptContents string

type SingleTokenBucket interface {
	Allow(ctx context.Context) (bool, float64, error)
}

type AddressableTokenBucket interface {
	Allow(ctx context.Context, address string) (bool, float64, error)
}

func checkBucketConfig(cfg BucketCFG) error {
	if cfg.Capacity <= 0 {
		return BucketCapacityInvalidError
	}

	if cfg.Refill <= 0 {
		return BucketRefillInvalidError
	}

	return nil
}
