package main

import (
	"context"
	"net/http"
	"time"

	"leetcode-rss/internal/api"

	"github.com/gin-gonic/gin"
)

func routes(handlers *api.Handlers, publicHandlers *api.PublicFeedHandlers, handlerTimeout time.Duration) http.Handler {
	g := gin.Default()

	health := g.Group("/health")
	{
		health.GET("", healthHandler)
	}

	g.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "OK. RSS at /leetcode.xml\n")
	})

	g.GET("/leetcode.xml", withTimeout(handlerTimeout, handlers.RSS))

	if publicHandlers != nil {
		g.GET("/f/:feedID/:secret", withTimeout(handlerTimeout, publicHandlers.PublicFeed))
	}

	return g
}

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "OK"})
}

func withTimeout(d time.Duration, fn gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), d)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		fn(c)
	}
}
