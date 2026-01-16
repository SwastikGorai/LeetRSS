package main

import (
	"context"
	"net/http"

	"leetcode-rss/internal/api"

	clerkhttp "github.com/clerk/clerk-sdk-go/v2/http"
	"github.com/gin-gonic/gin"
)

func (app *app) routes() http.Handler {
	g := gin.Default()

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
		protected.Use(app.clerkAuthMiddleware())
		protected.Use(api.RequireAuth(app.store))
		{
			// TODO: Add protected endpoints here
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

func (app *app) clerkAuthMiddleware() gin.HandlerFunc {
	passthrough := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	clerkHandler := clerkhttp.RequireHeaderAuthorization()(passthrough)
	return gin.WrapH(clerkHandler)
}
