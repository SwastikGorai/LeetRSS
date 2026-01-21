package main

import (
	"context"
	"net/http"

	"leetcode-rss/internal/api"

	"github.com/gin-gonic/gin"
)

func (app *app) routes() http.Handler {
	g := gin.Default()
	g.Use(corsMiddleware())

	health := g.Group("/health")
	{
		health.GET("", app.healthHandler)
	}

	root := g.Group("/")
	{
		root.GET("", app.rootHandler)
		root.GET("/leetcode.xml", app.withTimeout(app.handlers.RSS))
	}

	if app.publicHandlers != nil {
		feeds := g.Group("/f")
		{
			feeds.GET("/:feedID/:secret", app.withTimeout(app.publicHandlers.PublicFeed))
		}
	}

	if app.config.Clerk.SecretKey != "" && app.store != nil {
		protected := g.Group("/")
		protected.Use(api.ClerkAuthMiddleware(app.store))
		{
			protected.GET("/me", app.getCurrentUser)
			protected.GET("/feeds", app.listFeeds)
			protected.POST("/feeds", app.createFeed)
			protected.GET("/feeds/:id", app.getFeed)
			protected.PATCH("/feeds/:id", app.updateFeed)
			protected.POST("/feeds/:id/rotate", app.rotateFeedSecret)
			protected.DELETE("/feeds/:id", app.deleteFeed)
		}
	}

	return g
}

func (app *app) healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "OK"})
}

func (app *app) rootHandler(c *gin.Context) {
	c.String(http.StatusOK, "OK. RSS at /leetcode.xml\n")
}

func (app *app) withTimeout(fn gin.HandlerFunc) gin.HandlerFunc {
	timeout := app.config.Server.HandlerTimeout
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		fn(c)
	}
}
