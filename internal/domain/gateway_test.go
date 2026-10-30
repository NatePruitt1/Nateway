package domain

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

const testConfig = `
limits:
  algorithm:
    address: 'localhost:6379'
  general:
    capacity: 100
    refill: 10
  ip:
    capacity: 10
    refill: 1
gateway:
  listen: ':9000'
  locations:
    - path: '/'
      target: 'localhost:8080'
    - path: '/api'
      target: '${API_TARGET}'
      auth_required: true
      auth_key: "key1"
`

// setupGatewayDir creates a temp dir holding gateway.yaml and the cert.pem that
// LoadGateway reads from the working directory, and makes it the working directory.
func setupGatewayDir(t *testing.T, config string) string {
	t.Helper()
	dir := t.TempDir()
	writePublicKeyPEM(t, dir, "cert.pem", newRSAKey(t))
	path := filepath.Join(dir, "gateway.yaml")
	if err := os.WriteFile(path, []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	return path
}

func TestLoadGateway_ParsesConfig(t *testing.T) {
	t.Setenv("API_TARGET", "api.internal:9090")
	path := setupGatewayDir(t, testConfig)

	gw, err := LoadGateway(path)
	if err != nil {
		t.Fatalf("LoadGateway: %v", err)
	}

	if gw.Config.Server.ListenLocation != ":9000" {
		t.Errorf("listen = %q, want %q", gw.Config.Server.ListenLocation, ":9000")
	}
	if got := gw.Config.Limits.General.Capacity; got != 100 {
		t.Errorf("general capacity = %d, want 100", got)
	}
	if gw.Entry == nil || gw.Router == nil {
		t.Fatal("gateway missing Entry or Router")
	}

	ep := gw.Router.(*TrieRouter).PrefixTree.Find([]byte("/api"))
	if ep == nil {
		t.Fatal("route /api not registered")
	}
	if ep.Address != "api.internal:9090" {
		t.Errorf("env var not expanded: address = %q", ep.Address)
	}
	if !ep.AuthRequired || ep.AuthKey != "key1" {
		t.Errorf("auth settings not loaded: %+v", *ep)
	}
}

func TestLoadGateway_Errors(t *testing.T) {
	t.Run("missing config file", func(t *testing.T) {
		setupGatewayDir(t, testConfig)
		if _, err := LoadGateway("does-not-exist.yaml"); err == nil {
			t.Error("expected an error")
		}
	})

	t.Run("invalid yaml", func(t *testing.T) {
		path := setupGatewayDir(t, "gateway: [unclosed")
		if _, err := LoadGateway(path); err == nil {
			t.Error("expected an error")
		}
	})

	t.Run("missing cert.pem", func(t *testing.T) {
		path := setupGatewayDir(t, testConfig)
		os.Remove("cert.pem")
		if _, err := LoadGateway(path); err == nil {
			t.Error("expected an error")
		}
	})
}

// waitFor polls cond until it returns true or the timeout passes.
func waitFor(timeout time.Duration, cond func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return cond()
}

func TestWatchGateway_ReloadsValidChange(t *testing.T) {
	path := setupGatewayDir(t, testConfig)
	gw, err := LoadGateway(path)
	if err != nil {
		t.Fatal(err)
	}
	var ptr atomic.Pointer[Gateway]
	ptr.Store(gw)
	WatchGateway(path, &ptr)

	updated := []byte(testConfig[:len(testConfig)-1] + "\n    - path: '/new'\n      target: 'new:80'\n")
	if err := os.WriteFile(path, updated, 0o600); err != nil {
		t.Fatal(err)
	}

	reloaded := waitFor(2*time.Second, func() bool {
		g := ptr.Load()
		return g != nil && g != gw && len(g.Config.Server.Locations) == 3
	})
	if !reloaded {
		t.Error("gateway was not reloaded after a valid config change")
	}
}

func TestWatchGateway_InvalidChangeKeepsOldGateway(t *testing.T) {
	path := setupGatewayDir(t, testConfig)
	gw, err := LoadGateway(path)
	if err != nil {
		t.Fatal(err)
	}
	var ptr atomic.Pointer[Gateway]
	ptr.Store(gw)
	WatchGateway(path, &ptr)

	if err := os.WriteFile(path, []byte("gateway: [unclosed"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Give the watcher time to react, failing as soon as the gateway is replaced.
	replaced := waitFor(500*time.Millisecond, func() bool { return ptr.Load() != gw })
	if replaced {
		t.Errorf("gateway was replaced after an invalid config change (now %v)", ptr.Load())
	}
}
