package domain

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { client.Close() })
	return mr, client
}

func TestRedisTokenBucket_AllowsUpToCapacity(t *testing.T) {
	_, client := newTestRedis(t)
	tb := NewRedisTokenBucket(3, 0.0001, client)
	ctx := context.Background()

	for i := range 3 {
		ok, _, err := tb.Allow(ctx, "k")
		if err != nil {
			t.Fatalf("call %d: unexpected error: %v", i, err)
		}
		if !ok {
			t.Fatalf("call %d: denied, want allowed", i)
		}
	}

	ok, _, err := tb.Allow(ctx, "k")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("call beyond capacity was allowed")
	}
}

func TestRedisTokenBucket_Refills(t *testing.T) {
	_, client := newTestRedis(t)
	tb := NewRedisTokenBucket(1, 100, client) // 1 token every 10ms
	ctx := context.Background()

	if ok, _, _ := tb.Allow(ctx, "k"); !ok {
		t.Fatal("first call denied")
	}
	if ok, _, _ := tb.Allow(ctx, "k"); ok {
		t.Fatal("second immediate call allowed, bucket should be empty")
	}

	time.Sleep(50 * time.Millisecond)

	if ok, _, err := tb.Allow(ctx, "k"); !ok || err != nil {
		t.Errorf("call after refill: ok=%v err=%v, want allowed", ok, err)
	}
}

func TestRedisTokenBucket_NeverExceedsCapacity(t *testing.T) {
	_, client := newTestRedis(t)
	tb := NewRedisTokenBucket(2, 50, client) // 1 token every 20ms
	ctx := context.Background()

	tb.Allow(ctx, "k")
	time.Sleep(200 * time.Millisecond) // enough time to refill far past capacity

	allowed := 0
	for range 5 {
		// Back-to-back calls refill well under one token between them.
		if ok, _, _ := tb.Allow(ctx, "k"); ok {
			allowed++
		}
	}
	// 2 from the full bucket, plus at most 1 refilled during the loop.
	if allowed > 3 {
		t.Errorf("allowed %d requests, want at most capacity (2) plus refill slack (1)", allowed)
	}
}

func TestRedisTokenBucket_KeysAreIndependent(t *testing.T) {
	_, client := newTestRedis(t)
	ctx := context.Background()
	ipb := &RedisTokenBucketAddressable{*NewRedisTokenBucket(1, 0.0001, client)}
	gb := &RedisTokenBucketGeneral{*NewRedisTokenBucket(1, 0.0001, client)}

	if ok, _, _ := ipb.Allow(ctx, "1.1.1.1"); !ok {
		t.Fatal("first IP denied")
	}
	if ok, _, _ := ipb.Allow(ctx, "2.2.2.2"); !ok {
		t.Error("second IP denied: it shares a bucket with the first")
	}
	if ok, _, _ := gb.Allow(ctx); !ok {
		t.Error("general bucket denied: it shares a bucket with the IP buckets")
	}
}

func TestRedisTokenBucket_RecoversAfterScriptFlush(t *testing.T) {
	_, client := newTestRedis(t)
	tb := NewRedisTokenBucket(10, 1, client)
	ctx := context.Background()

	if _, _, err := tb.Allow(ctx, "k"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := client.ScriptFlush(ctx).Err(); err != nil {
		t.Fatalf("script flush: %v", err)
	}
	if ok, _, err := tb.Allow(ctx, "k"); !ok || err != nil {
		t.Errorf("after flush: ok=%v err=%v, want allowed", ok, err)
	}
}

func TestRedisTokenBucket_RedisDownReturnsError(t *testing.T) {
	mr, client := newTestRedis(t)
	tb := NewRedisTokenBucket(10, 1, client)
	mr.Close()

	ok, _, err := tb.Allow(context.Background(), "k")
	if err == nil {
		t.Error("expected an error when Redis is unreachable")
	}
	if ok {
		t.Error("request allowed while Redis is unreachable")
	}
}

// Run with -race: Allow is called from many request goroutines at once.
func TestRedisTokenBucket_ConcurrentAllow(t *testing.T) {
	_, client := newTestRedis(t)
	tb := NewRedisTokenBucket(50, 0.0001, client)
	ctx := context.Background()

	var wg sync.WaitGroup
	var mu sync.Mutex
	allowed := 0
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if ok, _, _ := tb.Allow(ctx, "k"); ok {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if allowed != 50 {
		t.Errorf("allowed %d of 100 concurrent requests, want exactly 50", allowed)
	}
}
