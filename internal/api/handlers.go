package api

import (
	"context"

	"github.com/gin-gonic/gin"
)

type FeedService interface {
	Build(ctx context.Context) ([]byte, error)
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

	b, err := h.svc.Build(c.Request.Context())
	if err != nil {
		c.String(502, err.Error())
		return
	}

	h.cache.Set(b)
	c.Data(200, "application/rss+xml", b)
}
