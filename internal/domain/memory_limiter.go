package domain

import (
	"context"
	"sync"
	"time"
)

type SingleMemoryLimiter struct {
	lock   sync.Mutex
	tokens float64

	refill_rate float64
	capacity    float64
	last_refill time.Time
}

type AddressableMemoryLimiter struct {
	lock     sync.Mutex
	limiters map[string]*SingleMemoryLimiter

	refill_rate float64
	capacity    float64
}

func NewSingleMemoryLimiter(capacity, refill_rate float64) *SingleMemoryLimiter {
	return &SingleMemoryLimiter{
		lock:        sync.Mutex{},
		tokens:      capacity,
		capacity:    capacity,
		refill_rate: refill_rate,
		last_refill: time.Now(),
	}
}

func (sml *SingleMemoryLimiter) Allow(ctx context.Context) (bool, float64, error) {
	sml.lock.Lock()
	defer sml.lock.Unlock()
	now := time.Now()
	timeSince := now.Sub(sml.last_refill).Seconds()
	fillAmount := timeSince * sml.refill_rate
	sml.tokens = min(sml.capacity, sml.tokens+fillAmount)
	sml.last_refill = now

	allowed := false

	if sml.tokens >= 1.0 {
		sml.tokens -= 1.0
		allowed = true
	}

	return allowed, sml.tokens, nil
}

func (aml *AddressableMemoryLimiter) Allow(ctx context.Context, address string) (bool, float64, error) {
	// Get the address
	aml.lock.Lock()
	defer aml.lock.Unlock()

	addr, ok := aml.limiters[address]
	if !ok {
		addr = NewSingleMemoryLimiter(aml.capacity, aml.refill_rate)
		aml.limiters[address] = addr
	}

	return addr.Allow(ctx)
}

func (aml *AddressableMemoryLimiter) pruneAddresses(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(1 * time.Minute):
			aml.lock.Lock()
			now := time.Now()
			for k := range aml.limiters {
				sml := aml.limiters[k]
				sml.lock.Lock()
				timePassed := now.Sub(aml.limiters[k].last_refill).Seconds()
				amount := timePassed * aml.refill_rate
				if aml.limiters[k].tokens+amount >= aml.capacity {
					delete(aml.limiters, k)
				} else {
					sml.lock.Unlock()
				}
			}
			aml.lock.Unlock()
		}
		time.Sleep(1 * time.Second)

	}
}
