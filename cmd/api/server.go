package main

import (
	"fmt"
	"net/http"
	"time"
)

func (app *app) serve() error {
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", app.config.Server.Port),
		Handler:           app.routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	return server.ListenAndServe()
}
