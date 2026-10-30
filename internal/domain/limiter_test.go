package domain

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeBucket struct {
	allow bool
	err   error
	calls int
}

func (f *fakeBucket) Allow(context.Context) (bool, float64, error) {
	f.calls++
	return f.allow, 0, f.err
}

type fakeAddrBucket struct {
	allow     bool
	err       error
	addresses []string
}

func (f *fakeAddrBucket) Allow(_ context.Context, addr string) (bool, float64, error) {
	f.addresses = append(f.addresses, addr)
	return f.allow, 0, f.err
}

func newLimitedRequest(remoteAddr string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = remoteAddr
	return req
}

func TestRateLimiter_AllowsWhenBothBucketsAllow(t *testing.T) {
	gb, ipb := &fakeBucket{allow: true}, &fakeAddrBucket{allow: true}
	next := &countingHandler{}

	rec := serve(t, RateLimiterMiddleware(gb, ipb)(next), newLimitedRequest("1.2.3.4:5555"))

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if next.calls != 1 {
		t.Errorf("next called %d times, want 1", next.calls)
	}
}

func TestRateLimiter_LimitedReturns429(t *testing.T) {
	tests := []struct {
		name string
		gb   *fakeBucket
		ipb  *fakeAddrBucket
	}{
		{"general bucket empty", &fakeBucket{allow: false}, &fakeAddrBucket{allow: true}},
		{"ip bucket empty", &fakeBucket{allow: true}, &fakeAddrBucket{allow: false}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next := &countingHandler{}
			rec := serve(t, RateLimiterMiddleware(tt.gb, tt.ipb)(next), newLimitedRequest("1.2.3.4:5555"))

			if rec.Code != http.StatusTooManyRequests {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusTooManyRequests)
			}
			if next.calls != 0 {
				t.Errorf("next called %d times, want 0", next.calls)
			}
		})
	}
}

func TestRateLimiter_KeysByIPNotPort(t *testing.T) {
	gb, ipb := &fakeBucket{allow: true}, &fakeAddrBucket{allow: true}
	h := RateLimiterMiddleware(gb, ipb)(&countingHandler{})

	serve(t, h, newLimitedRequest("1.2.3.4:1111"))
	serve(t, h, newLimitedRequest("1.2.3.4:2222"))
	serve(t, h, newLimitedRequest("[2001:db8::1]:3333"))

	want := []string{"1.2.3.4", "1.2.3.4", "2001:db8::1"}
	for i, addr := range ipb.addresses {
		if addr != want[i] {
			t.Errorf("request %d keyed as %q, want %q", i, addr, want[i])
		}
	}
}

func TestRateLimiter_GeneralLimitedSkipsIPBucket(t *testing.T) {
	gb, ipb := &fakeBucket{allow: false}, &fakeAddrBucket{allow: true}

	serve(t, RateLimiterMiddleware(gb, ipb)(&countingHandler{}), newLimitedRequest("1.2.3.4:5555"))

	if len(ipb.addresses) != 0 {
		t.Errorf("ip bucket consumed %d tokens after general limit hit, want 0", len(ipb.addresses))
	}
}

// Currently the limiter fails closed: a Redis error blocks the request.
// If you decide to fail open instead, flip this test to expect next to be called.
func TestRateLimiter_BucketErrorBlocksRequest(t *testing.T) {
	tests := []struct {
		name string
		gb   *fakeBucket
		ipb  *fakeAddrBucket
	}{
		{"general bucket error", &fakeBucket{allow: true, err: errors.New("redis down")}, &fakeAddrBucket{allow: true}},
		{"ip bucket error", &fakeBucket{allow: true}, &fakeAddrBucket{allow: true, err: errors.New("redis down")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next := &countingHandler{}
			rec := serve(t, RateLimiterMiddleware(tt.gb, tt.ipb)(next), newLimitedRequest("1.2.3.4:5555"))

			if next.calls != 0 {
				t.Errorf("next called %d times, want 0", next.calls)
			}
			if rec.Code < 400 {
				t.Errorf("status = %d, want an error status", rec.Code)
			}
		})
	}
}
