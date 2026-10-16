package domain

import (
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func GatewayPinHandler(gateway *atomic.Pointer[Gateway]) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pinnedGateway := gateway.Load()
		pinnedGateway.Entry.ServeHTTP(w, r)
	})
}

func BenchmarkMiddlewareStack(b *testing.B) {
	// 1. Read and parse your local RSA private key from disk
	pemBytes, err := os.ReadFile("../../key.pem")
	if err != nil {
		b.Fatalf("failed to read key.pem: %v", err)
	}

	signKey, err := jwt.ParseRSAPrivateKeyFromPEM(pemBytes)
	if err != nil {
		b.Fatalf("failed to parse RSA private key: %v", err)
	}

	// 2. Build standard token claims (Valid for 1 hour from now)
	claims := jwt.MapClaims{
		"sub":  "1234567890",
		"name": "Gateway Benchmark User",
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(time.Hour).Unix(),
	}

	// 3. Generate and sign the RS256 token string
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(signKey)
	if err != nil {
		b.Fatalf("failed to sign token: %v", err)
	}

	// Setup a dummy final handler
	gtwy, err := LoadGateway("../../gateway.yaml")
	if err != nil {
		panic(err)
	}

	var atGate atomic.Pointer[Gateway]
	atGate.Store(gtwy)
	handler := GatewayPinHandler(&atGate)

	// Reusable request and recorder to minimize benchmark framework noise
	req := httptest.NewRequest("GET", "/api/v1/resource", nil)

	// 4. Inject the genuine token string with the Bearer prefix
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	b.ResetTimer() // Exclude the file-read, parsing, and signing operations from metrics
	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(w, req)
	}
}
