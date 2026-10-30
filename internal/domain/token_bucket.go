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
