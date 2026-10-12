package domain

import (
	"io"
	"log"
	"net"
	"net/http"
)

type Gateway struct {
	Limiter *RedisRateLimiterLoader
	Router  *LongestMatchRouterLoader
}

func (g *Gateway) StartGateway(listener net.Listener) {
	server := http.Server{
		Handler: http.HandlerFunc(g.HandleRequest),
	}

	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		log.Printf("Server runtime error: %v\n", err)
	}
}

func (g *Gateway) HandleRequest(w http.ResponseWriter, r *http.Request) {
	// Get pinned configurations
	limiter := g.Limiter.Pin()
	router := g.Router.Pin()

	ok, err := limiter.CheckGeneralRequestAllowed(r)
	if !ok {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	ok, err = limiter.CheckIpRequestAllowed(r)
	if !ok {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	endpoint, err := router.GetEndpoint(r)
	if err == NoRouteFoundErr {
		w.WriteHeader(404)
	} else if err != nil {
		log.Println(err.Error())
		w.WriteHeader(502)
		return
	}

	// Send the request to the endpoint.
	outReq := r.Clone(r.Context())
	outReq.URL.Scheme = "http"
	outReq.URL.Host = endpoint.Address
	outReq.URL.Path = r.URL.Path
	outReq.URL.RawQuery = r.URL.RawQuery

	outReq.RequestURI = ""

	client := &http.Client{}
	resp, err := client.Do(outReq)
	if err != nil {
		log.Println(err.Error())
		w.WriteHeader(502)
		return
	}
	// 5. Copy headers from the backend response to the client response
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// 6. Copy the status code (Must happen AFTER setting headers, but BEFORE writing the body)
	w.WriteHeader(resp.StatusCode)

	// 7. Stream the body chunks directly to the client
	// io.Copy uses a 32KB internal buffer to read and write concurrently
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		// The status header has already been sent, so we can't send an http.Error anymore.
		// Instead, log the connection failure.
		println("Error streaming response body:", err.Error())
	}
}
