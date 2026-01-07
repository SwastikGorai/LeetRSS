package main

import (
	"fmt"
	"net/http"
	"time"
)

func newServer(port int, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
}
