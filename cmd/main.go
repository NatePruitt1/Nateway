package main

import (
	"ApiGateway/internal/domain"
	"log"
	"net"
	"net/http"
	"sync/atomic"
)

func GatewayPinHandler(gateway *atomic.Pointer[domain.Gateway]) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pinnedGateway := gateway.Load()
		pinnedGateway.Entry.ServeHTTP(w, r)
	})
}

func main() {

	log.Println("Starting ApiGateway")
	gtwy, err := domain.LoadGateway("./gateway.yaml")
	if err != nil {
		panic(err)
	}
	if gtwy == nil {
		panic("Gateway load failure!")
	}

	var atGate atomic.Pointer[domain.Gateway]
	atGate.Store(gtwy)

	domain.WatchGateway("./gateway.yaml", &atGate)

	listener, err := net.Listen("tcp", gtwy.Config.Server.ListenLocation)
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	server := http.Server{
		Handler: GatewayPinHandler(&atGate),
	}

	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		log.Printf("Server runtime err %v\n", err)
	}
}
