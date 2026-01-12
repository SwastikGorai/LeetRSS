package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"leetcode-rss/internal/leetcode"
	"leetcode-rss/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"
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

type PublicFeedHandlers struct {
	store    store.Store
	lc       *leetcode.Client
	sfGroup  singleflight.Group
	cacheTTL time.Duration
}

func NewPublicFeedHandlers(s store.Store, lc *leetcode.Client, cacheTTL time.Duration) *PublicFeedHandlers {
	return &PublicFeedHandlers{
		store:    s,
		lc:       lc,
		cacheTTL: cacheTTL,
	}
}

// GET /f/:feedID/:secret.xml
func (h *PublicFeedHandlers) PublicFeed(c *gin.Context) {
	feedID := c.Param("feedID")
	secretParam := c.Param("secret")
	secret := strings.TrimSuffix(secretParam, ".xml")

	if !isValidUUID(feedID) {
		c.Status(http.StatusNotFound)
		return
	}

	ctx := c.Request.Context()

	feed, err := h.store.GetFeedByIDAndSecret(ctx, feedID, secret)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.Status(http.StatusNotFound)
			return
		}
		log.Printf("error fetching feed %s: %v", feedID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": "internal_error", "message": "failed to fetch feed"},
		})
		return
	}

	if !feed.Enabled {
		c.Status(http.StatusNotFound)
		return
	}

	cache, cacheErr := h.store.GetFeedCache(ctx, feedID)
	hasFreshCache := cacheErr == nil && cache != nil && cache.ExpiresAt.After(time.Now())

	if hasFreshCache {
		h.serveCachedFeed(c, cache, false)
		return
	}

	hasStaleCache := cacheErr == nil && cache != nil

	result, err, _ := h.sfGroup.Do(feedID, func() (interface{}, error) {
		return h.refreshFeed(ctx, feed, selfURLFromRequest(c))
	})

	if err != nil {
		log.Printf("error refreshing feed %s: %v", feedID, err)
		if hasStaleCache {
			h.serveCachedFeed(c, cache, true)
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{
			"error": gin.H{"code": "upstream_error", "message": err.Error()},
		})
		return
	}

	newCache := result.(*store.FeedCache)
	h.serveCachedFeed(c, newCache, false)
}

func (h *PublicFeedHandlers) serveCachedFeed(c *gin.Context, cache *store.FeedCache, stale bool) {
	if etag := c.GetHeader("If-None-Match"); etag != "" && etag == cache.ETag {
		c.Status(http.StatusNotModified)
		return
	}

	if modifiedSince := c.GetHeader("If-Modified-Since"); modifiedSince != "" {
		t, err := http.ParseTime(modifiedSince)
		if err == nil && !cache.LastBuiltAt.After(t) {
			c.Status(http.StatusNotModified)
			return
		}
	}

	if stale {
		c.Header("Warning", `110 - "Response is stale"`)
	}
	c.Header("ETag", cache.ETag)
	c.Header("Last-Modified", cache.LastBuiltAt.UTC().Format(http.TimeFormat))
	c.Header("Cache-Control", fmt.Sprintf("public, max-age=%d", int(h.cacheTTL.Seconds())))
	c.Data(http.StatusOK, "application/rss+xml", cache.XML)
}

func (h *PublicFeedHandlers) refreshFeed(ctx context.Context, feed *store.Feed, selfURL string) (*store.FeedCache, error) {
	svc := UGCFeedService{
		Usernames: feed.Usernames,
		LC:        h.lc,
		First:     feed.FirstPerUser,
	}

	xml, err := svc.Build(ctx, selfURL)
	if err != nil {
		errStr := err.Error()
		_ = h.store.SetFeedCache(ctx, &store.FeedCache{
			FeedID:    feed.ID,
			LastError: &errStr,
		})
		return nil, err
	}

	now := time.Now()
	cache := &store.FeedCache{
		FeedID:      feed.ID,
		XML:         xml,
		ETag:        generateETag(xml),
		LastBuiltAt: now,
		ExpiresAt:   now.Add(h.cacheTTL),
		LastError:   nil,
	}

	if err := h.store.SetFeedCache(ctx, cache); err != nil {
		log.Printf("warning: failed to cache feed %s: %v", feed.ID, err)
	}

	return cache, nil
}

func generateETag(xml []byte) string {
	h := sha256.Sum256(xml)
	return fmt.Sprintf(`"%s"`, hex.EncodeToString(h[:8]))
}

func isValidUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

type FeedServiceBuilder interface {
	Build(ctx context.Context, usernames []string, first int, selfURL string) ([]byte, error)
}
