package domain

import (
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func newAuthUnderTest(t *testing.T) (Middleware, *rsa.PrivateKey) {
	t.Helper()
	key := newRSAKey(t)
	auth := Authenticator{Keys: map[string]*rsa.PublicKey{"key1": &key.PublicKey}}
	return AuthenticatorMiddleware(auth), key
}

func TestAuth_ValidTokenCallsNextOnce(t *testing.T) {
	mw, key := newAuthUnderTest(t)
	next := &countingHandler{}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+signToken(t, key, time.Now().Add(time.Hour)))
	req = withEndpoint(req, &Endpoint{AuthRequired: true, AuthKey: "key1"})

	rec := serve(t, mw(next), req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if next.calls != 1 {
		t.Errorf("next called %d times, want 1", next.calls)
	}
}

func TestAuth_NotRequiredCallsNextOnce(t *testing.T) {
	mw, _ := newAuthUnderTest(t)
	next := &countingHandler{}

	req := withEndpoint(httptest.NewRequest(http.MethodGet, "/", nil), &Endpoint{AuthRequired: false})
	rec := serve(t, mw(next), req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if next.calls != 1 {
		t.Errorf("next called %d times, want 1", next.calls)
	}
}

func TestAuth_RejectsBadTokens(t *testing.T) {
	_, otherKey := newAuthUnderTest(t)

	tests := []struct {
		name    string
		header  func(t *testing.T, key *rsa.PrivateKey) string
		authKey string
	}{
		{
			name:    "missing header",
			header:  func(*testing.T, *rsa.PrivateKey) string { return "" },
			authKey: "key1",
		},
		{
			name: "wrong scheme",
			header: func(t *testing.T, key *rsa.PrivateKey) string {
				return "Basic " + signToken(t, key, time.Now().Add(time.Hour))
			},
			authKey: "key1",
		},
		{
			name:    "malformed token",
			header:  func(*testing.T, *rsa.PrivateKey) string { return "Bearer not.a.jwt" },
			authKey: "key1",
		},
		{
			name: "expired token",
			header: func(t *testing.T, key *rsa.PrivateKey) string {
				return "Bearer " + signToken(t, key, time.Now().Add(-time.Hour))
			},
			authKey: "key1",
		},
		{
			name: "signed by a different key",
			header: func(t *testing.T, _ *rsa.PrivateKey) string {
				return "Bearer " + signToken(t, otherKey, time.Now().Add(time.Hour))
			},
			authKey: "key1",
		},
		{
			// Algorithm confusion: an HMAC token must never be accepted by an RSA verifier.
			name: "HS256 token",
			header: func(t *testing.T, _ *rsa.PrivateKey) string {
				s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
				}).SignedString([]byte("secret"))
				if err != nil {
					t.Fatalf("sign hs256: %v", err)
				}
				return "Bearer " + s
			},
			authKey: "key1",
		},
		{
			name: "endpoint references unknown key",
			header: func(t *testing.T, key *rsa.PrivateKey) string {
				return "Bearer " + signToken(t, key, time.Now().Add(time.Hour))
			},
			authKey: "does-not-exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw, key := newAuthUnderTest(t)
			next := &countingHandler{}

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if h := tt.header(t, key); h != "" {
				req.Header.Set("Authorization", h)
			}
			req = withEndpoint(req, &Endpoint{AuthRequired: true, AuthKey: tt.authKey})

			rec := serve(t, mw(next), req)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
			}
			if next.calls != 0 {
				t.Errorf("next called %d times, want 0", next.calls)
			}
		})
	}
}

func TestAuth_MissingEndpointInContextReturns500(t *testing.T) {
	mw, _ := newAuthUnderTest(t)
	next := &countingHandler{}

	rec := serve(t, mw(next), httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if next.calls != 0 {
		t.Errorf("next called %d times, want 0", next.calls)
	}
}

func TestExtractToken(t *testing.T) {
	tests := []struct {
		header, want string
	}{
		{"Bearer abc.def.ghi", "abc.def.ghi"},
		{"", ""},
		{"Bearer", ""},
		{"Basic abc", ""},
		{"Bearer a b", ""},
	}
	for _, tt := range tests {
		if got := extractToken(tt.header); got != tt.want {
			t.Errorf("extractToken(%q) = %q, want %q", tt.header, got, tt.want)
		}
	}
}

func TestLoadPublicKey(t *testing.T) {
	dir := t.TempDir()

	t.Run("valid PEM", func(t *testing.T) {
		key := newRSAKey(t)
		path := writePublicKeyPEM(t, dir, "valid.pem", key)

		got, err := loadPrivateKey(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil || got.N.Cmp(key.PublicKey.N) != 0 {
			t.Errorf("loaded key does not match the written key")
		}
	})

	t.Run("invalid PEM returns error", func(t *testing.T) {
		path := filepath.Join(dir, "garbage.pem")
		if err := os.WriteFile(path, []byte("not a key"), 0o600); err != nil {
			t.Fatal(err)
		}

		got, err := loadPrivateKey(path)
		if err == nil {
			t.Errorf("expected an error, got key=%v err=nil", got)
		}
	})

	t.Run("missing file returns error", func(t *testing.T) {
		if _, err := loadPrivateKey(filepath.Join(dir, "missing.pem")); err == nil {
			t.Error("expected an error for a missing file")
		}
	})
}
