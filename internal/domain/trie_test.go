package domain

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTrie_ExactMatch(t *testing.T) {
	trie := NewTrie()
	ep := &Endpoint{Address: "a"}
	trie.Add([]byte("/path"), ep)

	if got := trie.Find([]byte("/path")); got != ep {
		t.Errorf("Find(/path) = %v, want %v", got, ep)
	}
}

func TestTrie_LongestPrefixMatch(t *testing.T) {
	root := &Endpoint{Address: "root"}
	path := &Endpoint{Address: "path"}
	deep := &Endpoint{Address: "deep"}

	trie := NewTrie()
	trie.Add([]byte("/"), root)
	trie.Add([]byte("/path"), path)
	trie.Add([]byte("/path/deep"), deep)

	tests := []struct {
		key  string
		want *Endpoint
	}{
		{"/", root},
		{"/path", path},
		{"/path/", path},
		{"/path/users", path},
		{"/path/deep/x", deep},
		{"/other", root},
		// Design decision: matching on path segments means /pathology must not
		// match /path. If you choose byte-prefix matching instead, change this to path.
		{"/pathology", root},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := trie.Find([]byte(tt.key))
			if got != tt.want {
				t.Errorf("Find(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestTrie_NoMatch(t *testing.T) {
	trie := NewTrie()
	trie.Add([]byte("/api"), &Endpoint{Address: "api"})

	for _, key := range []string{"", "/", "/other", "/ap"} {
		if got := trie.Find([]byte(key)); got != nil {
			t.Errorf("Find(%q) = %v, want nil", key, got)
		}
	}
}

func TestTrie_AddOverwrites(t *testing.T) {
	trie := NewTrie()
	trie.Add([]byte("/a"), &Endpoint{Address: "old"})
	newer := &Endpoint{Address: "new"}
	trie.Add([]byte("/a"), newer)

	if got := trie.Find([]byte("/a")); got != newer {
		t.Errorf("Find(/a) = %v, want the most recently added endpoint", got)
	}
}

func TestTrieRouter_FromConfig(t *testing.T) {
	cfg := &GatewayConfig{}
	cfg.Server.Locations = []EndpointCFG{
		{Path: "/", Target: "root:80"},
		{Path: "/api", Target: "api:80", Auth: true, AuthKey: "key1"},
	}
	router := NewPrefixTreeRouter(cfg)

	t.Run("maps config fields", func(t *testing.T) {
		ep, err := router.GetEndpoint(httptest.NewRequest(http.MethodGet, "/api", nil))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := Endpoint{Address: "api:80", AuthRequired: true, AuthKey: "key1"}
		if *ep != want {
			t.Errorf("endpoint = %+v, want %+v", *ep, want)
		}
	})

	t.Run("each route keeps its own endpoint", func(t *testing.T) {
		ep, err := router.GetEndpoint(httptest.NewRequest(http.MethodGet, "/", nil))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ep.Address != "root:80" {
			t.Errorf("address = %q, want %q", ep.Address, "root:80")
		}
	})
}

func TestTrieRouter_NotFound(t *testing.T) {
	cfg := &GatewayConfig{}
	cfg.Server.Locations = []EndpointCFG{{Path: "/api", Target: "api:80"}}
	router := NewPrefixTreeRouter(cfg)

	ep, err := router.GetEndpoint(httptest.NewRequest(http.MethodGet, "/nope", nil))
	if !errors.Is(err, NoRouteFoundErr) {
		t.Errorf("err = %v, want NoRouteFoundErr", err)
	}
	if ep != nil {
		t.Errorf("endpoint = %v, want nil", ep)
	}
}
