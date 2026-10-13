package main

import (
	"ApiGateway/internal/domain"
	"log"
	"net"
	"net/http"
	"strconv"

	"github.com/redis/go-redis/v9"
)

func main() {
	client := redis.NewClient(
		&redis.Options{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
		},
	)

	log.Println("Starting ApiGateway")
	gtwy, err := domain.LoadGateway("./gateway.yaml", client)
	if err != nil {
		panic(err)
	}

	listener, err := net.Listen("tcp", "localhost:"+strconv.Itoa(gtwy.Config.Server.ListenPort))
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	server := http.Server{
		Handler: gtwy.Entry,
	}

	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		log.Printf("Server runtime err %v\n", err)
	}
}
