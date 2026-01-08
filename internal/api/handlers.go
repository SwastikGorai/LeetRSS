package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

type FeedService interface {
	Build(ctx context.Context, selfURL string) ([]byte, error)
}

type Handlers struct {
	svc   FeedService
	cache *Cache
}

func NewHandlers(svc FeedService, cache *Cache) *Handlers {
	return &Handlers{
		svc:   svc,
		cache: cache,
	}
}

func (h *Handlers) RSS(c *gin.Context) {
	if b, ok := h.cache.Get(); ok {
		c.Data(200, "application/rss+xml", b)
		return
	}

	b, err := h.svc.Build(c.Request.Context(), selfURLFromRequest(c))
	if err != nil {
		c.String(502, err.Error())
		return
	}

	h.cache.Set(b)
	c.Data(200, "application/rss+xml", b)
}

func selfURLFromRequest(c *gin.Context) string {
	scheme := forwardedFirst(c.GetHeader("X-Forwarded-Proto"))
	if scheme == "" {
		if c.Request.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}

	host := forwardedFirst(c.GetHeader("X-Forwarded-Host"))
	if host == "" {
		host = c.Request.Host
	}

	return fmt.Sprintf("%s://%s%s", scheme, host, c.Request.URL.Path)
}

func forwardedFirst(v string) string {
	if v == "" {
		return ""
	}
	if i := strings.IndexByte(v, ','); i >= 0 {
		return strings.TrimSpace(v[:i])
	}
	return strings.TrimSpace(v)
}
