package main

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", strings.Join([]string{
				"Authorization",
				"Content-Type",
				"Accept",
				"Origin",
				"X-Requested-With",
			}, ", "))
			c.Header("Access-Control-Allow-Methods", strings.Join([]string{
				"GET",
				"POST",
				"PATCH",
				"DELETE",
				"OPTIONS",
			}, ", "))
		}

		if c.Request.Method == http.MethodOptions {
			c.Status(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
