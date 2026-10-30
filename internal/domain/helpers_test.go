package domain

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// countingHandler records how many times it was called and the last request it saw.
type countingHandler struct {
	calls   int
	lastReq *http.Request
}

func (h *countingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.calls++
	h.lastReq = r
	w.WriteHeader(http.StatusOK)
}

// serve runs the handler and turns a panic into a test failure, so one
// panicking handler doesn't take down the whole test binary.
func serve(t *testing.T, h http.Handler, req *http.Request) (rec *httptest.ResponseRecorder) {
	t.Helper()
	rec = httptest.NewRecorder()
	defer func() {
		if p := recover(); p != nil {
			t.Errorf("handler panicked: %v", p)
		}
	}()
	h.ServeHTTP(rec, req)
	return rec
}

// withEndpoint attaches an endpoint to the request context the same way RouterMiddleware does.
func withEndpoint(r *http.Request, ep *Endpoint) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), "endpoint", ep))
}

func newRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}
	return key
}

func signToken(t *testing.T, key *rsa.PrivateKey, exp time.Time) string {
	t.Helper()
	claims := jwt.RegisteredClaims{
		Subject:   "test-user",
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(exp),
	}
	s, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(key)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return s
}

// writePublicKeyPEM writes key's public half as a PEM file and returns its path.
func writePublicKeyPEM(t *testing.T, dir, name string, key *rsa.PrivateKey) string {
	t.Helper()
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}
	path := filepath.Join(dir, name)
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
	if err := os.WriteFile(path, pemBytes, 0o600); err != nil {
		t.Fatalf("write pem: %v", err)
	}
	return path
}
