package domain

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func TestChain_RunsMiddlewaresInOrder(t *testing.T) {
	var order []string
	mark := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name)
				next.ServeHTTP(w, r)
			})
		}
	}
	final := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { order = append(order, "handler") })

	serve(t, Chain(final, mark("a"), mark("b"), mark("c")), httptest.NewRequest(http.MethodGet, "/", nil))

	want := []string{"a", "b", "c", "handler"}
	if !reflect.DeepEqual(order, want) {
		t.Errorf("order = %v, want %v", order, want)
	}
}

func TestChain_NoMiddlewares(t *testing.T) {
	next := &countingHandler{}
	serve(t, Chain(next), httptest.NewRequest(http.MethodGet, "/", nil))

	if next.calls != 1 {
		t.Errorf("handler called %d times, want 1", next.calls)
	}
}

// backendHost returns the host:port of a test server, the format Endpoint.Address uses.
func backendHost(t *testing.T, srv *httptest.Server) string {
	t.Helper()
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	return u.Host
}

func TestProxyHandler_ForwardsRequestAndResponse(t *testing.T) {
	var got struct {
		method, path, query, body, header string
	}
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got.method, got.path, got.query, got.body = r.Method, r.URL.Path, r.URL.RawQuery, string(b)
		got.header = r.Header.Get("X-Test")

		w.Header().Set("X-Backend", "yes")
		w.WriteHeader(http.StatusCreated)
		io.WriteString(w, "created")
	}))
	defer backend.Close()

	req := httptest.NewRequest(http.MethodPost, "/items?id=7", strings.NewReader("payload"))
	req.Header.Set("X-Test", "forwarded")
	req = withEndpoint(req, &Endpoint{Address: backendHost(t, backend)})

	rec := serve(t, ProxyHandler(), req)

	if got.method != http.MethodPost || got.path != "/items" || got.query != "id=7" || got.body != "payload" {
		t.Errorf("backend saw %+v", got)
	}
	if got.header != "forwarded" {
		t.Errorf("backend X-Test header = %q, want %q", got.header, "forwarded")
	}
	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if rec.Header().Get("X-Backend") != "yes" {
		t.Errorf("response missing backend header")
	}
	if rec.Body.String() != "created" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "created")
	}
}

func TestProxyHandler_BackendErrorStatusPassesThrough(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer backend.Close()

	req := withEndpoint(httptest.NewRequest(http.MethodGet, "/", nil), &Endpoint{Address: backendHost(t, backend)})
	rec := serve(t, ProxyHandler(), req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestProxyHandler_UnreachableBackendReturns502(t *testing.T) {
	backend := httptest.NewServer(http.NotFoundHandler())
	addr := backendHost(t, backend)
	backend.Close()

	req := withEndpoint(httptest.NewRequest(http.MethodGet, "/", nil), &Endpoint{Address: addr})
	rec := serve(t, ProxyHandler(), req)

	if rec.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadGateway)
	}
}
