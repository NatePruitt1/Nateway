package domain

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type stubRouter struct {
	ep  *Endpoint
	err error
}

func (s stubRouter) GetEndpoint(*http.Request) (*Endpoint, error) { return s.ep, s.err }

func TestRouterMiddleware_AttachesEndpoint(t *testing.T) {
	ep := &Endpoint{Address: "backend:80"}
	next := &countingHandler{}

	rec := serve(t, RouterMiddleware(stubRouter{ep: ep})(next), httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if next.calls != 1 {
		t.Fatalf("next called %d times, want 1", next.calls)
	}
	if got, _ := next.lastReq.Context().Value("endpoint").(*Endpoint); got != ep {
		t.Errorf("endpoint in context = %v, want %v", got, ep)
	}
}

func TestRouterMiddleware_NoRouteReturns404(t *testing.T) {
	tests := map[string]stubRouter{
		"error":        {err: NoRouteFoundErr},
		"nil endpoint": {},
	}
	for name, router := range tests {
		t.Run(name, func(t *testing.T) {
			next := &countingHandler{}
			rec := serve(t, RouterMiddleware(router)(next), httptest.NewRequest(http.MethodGet, "/x", nil))

			if rec.Code != http.StatusNotFound {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
			}
			if next.calls != 0 {
				t.Errorf("next called %d times, want 0", next.calls)
			}
		})
	}
}
