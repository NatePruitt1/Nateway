package domain

import (
	"io"
	"log"
	"net/http"
)

type Middleware func(http.Handler) http.Handler

func Chain(handler http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}

	return handler
}

func Logger() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Request: %s", r.URL.Path)
			next.ServeHTTP(w, r)
		})
	}
}

func ProxyHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Send the request to the endpoint.
		endpoint := r.Context().Value("endpoint").(*Endpoint)

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
	})
}
