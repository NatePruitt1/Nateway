package main

import (
	"io"
	"log"
	"net/http"
	"os"
)

func echoHandler(w http.ResponseWriter, r *http.Request) {
	// Dynamically captures and prints whatever path the user sent
	log.Printf("[Incoming] Method: %s | Full Path: %s", r.Method, r.URL.Path)

	// Echo the body back to the client
	_, err := io.Copy(w, r.Body)
	if err != nil {
		log.Printf("Error echoing body: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func main() {
	// The "/" pattern catches all sub-paths (e.g., /users, /api/v1/test, etc.)
	http.HandleFunc("/", echoHandler)

	log.Printf("Multi-path Echo server is running on %s...\n", os.Args[1])
	log.Fatal(http.ListenAndServe(os.Args[1], nil))
}
