package main

import (
	"ApiGateway/internal/gateway"
	"log"
	"net"
	"net/http"
)

func GatewayPinHandler(gateway *gateway.Gateway) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gateway.Entry.ServeHTTP(w, r)
	})
}

func main() {

	log.Println("Starting ApiGateway")
	gtwy, err := gateway.LoadGateway("./gateway.yaml")
	if err != nil {
		panic(err)
	}
	if gtwy == nil {
		panic("Gateway load failure!")
	}

	listener, err := net.Listen("tcp", gtwy.ListenLocation.String())
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	server := http.Server{
		Handler: GatewayPinHandler(gtwy),
	}

	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		log.Printf("Server runtime err %v\n", err)
	}
}
