package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"leetcode-rss/internal/api"
	"leetcode-rss/internal/leetcode"
	"leetcode-rss/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	defaultFirstPerUser = 15
	minFirstPerUser     = 1
	maxFirstPerUser     = 50
	maxFeedNameLength   = 100
	secretBytes         = 32
)

func (app *app) getCurrentUser(c *gin.Context) {
	userID, ok := api.GetUserID(c)
	if !ok {
		api.AbortJSONError(c, http.StatusInternalServerError, api.ErrorCodeInternal, "missing user context")
		return
	}

	user, err := app.store.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			api.AbortJSONError(c, http.StatusNotFound, api.ErrorCodeNotFound, "user not found")
			return
		}
		api.AbortJSONError(c, http.StatusInternalServerError, api.ErrorCodeInternal, "failed to fetch user")
		return
	}

	feedCount, err := app.store.CountFeedsByUserID(c.Request.Context(), userID)
	if err != nil {
		api.AbortJSONError(c, http.StatusInternalServerError, api.ErrorCodeInternal, "failed to count feeds")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":          user.ID,
		"email":       user.Email,
		"created_at":  user.CreatedAt.Format(time.RFC3339),
		"feeds_count": feedCount,
	})
}

func (app *app) listFeeds(c *gin.Context) {
	userID, ok := api.GetUserID(c)
	if !ok {
		api.AbortJSONError(c, http.StatusInternalServerError, api.ErrorCodeInternal, "missing user context")
		return
	}

	feeds, err := app.store.ListFeedsByUserID(c.Request.Context(), userID)
	if err != nil {
		api.AbortJSONError(c, http.StatusInternalServerError, api.ErrorCodeInternal, "failed to list feeds")
		return
	}

	result := make([]gin.H, 0, len(feeds))
	for _, feed := range feeds {
		result = append(result, gin.H{
			"id":             feed.ID,
			"name":           feed.Name,
			"usernames":      feed.Usernames,
			"first_per_user": feed.FirstPerUser,
			"enabled":        feed.Enabled,
			"url":            app.feedURL(feed.ID, feed.Secret),
			"created_at":     feed.CreatedAt.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, result)
}

func (app *app) createFeed(c *gin.Context) {
	userID, ok := api.GetUserID(c)
	if !ok {
		api.AbortJSONError(c, http.StatusInternalServerError, api.ErrorCodeInternal, "missing user context")
		return
	}

	feedCount, err := app.store.CountFeedsByUserID(c.Request.Context(), userID)
	if err != nil {
		api.AbortJSONError(c, http.StatusInternalServerError, api.ErrorCodeInternal, "failed to count feeds")
		return
	}
	if feedCount >= app.config.Limits.MaxFeedsPerUser {
		api.AbortJSONError(c, http.StatusForbidden, api.ErrorCodeQuota, "Feed limit reached")
		return
	}

	var req struct {
		Name         string   `json:"name"`
		Usernames    []string `json:"usernames"`
		FirstPerUser *int     `json:"first_per_user"`
		Enabled      *bool    `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		api.AbortJSONErrorWithDetails(c, http.StatusBadRequest, api.ErrorCodeValidation, "invalid request body", err.Error())
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		api.AbortJSONError(c, http.StatusBadRequest, api.ErrorCodeValidation, "name is required")
		return
	}
	if len(req.Name) > maxFeedNameLength {
		api.AbortJSONError(c, http.StatusBadRequest, api.ErrorCodeValidation, fmt.Sprintf("name must be at most %d characters", maxFeedNameLength))
		return
	}

	if len(req.Usernames) == 0 {
		api.AbortJSONError(c, http.StatusBadRequest, api.ErrorCodeValidation, "at least one username is required")
		return
	}

	validUsernames := make([]string, 0, len(req.Usernames))
	seen := make(map[string]struct{})
	invalidUsernames := make([]string, 0)

	for _, username := range req.Usernames {
		username = strings.TrimSpace(username)
		if username == "" {
			continue
		}
		if err := leetcode.ValidateUsername(username); err != nil {
			invalidUsernames = append(invalidUsernames, username)
			continue
		}
		if _, exists := seen[username]; !exists {
			seen[username] = struct{}{}
			validUsernames = append(validUsernames, username)
		}
	}

	if len(invalidUsernames) > 0 {
		api.AbortJSONErrorWithDetails(c, http.StatusBadRequest, api.ErrorCodeValidation, "invalid usernames", invalidUsernames)
		return
	}

	if len(validUsernames) == 0 {
		api.AbortJSONError(c, http.StatusBadRequest, api.ErrorCodeValidation, "at least one valid username is required")
		return
	}

	if len(validUsernames) > app.config.Limits.MaxUsernamesPerFeed {
		api.AbortJSONError(c, http.StatusBadRequest, api.ErrorCodeValidation, fmt.Sprintf("maximum %d usernames per feed", app.config.Limits.MaxUsernamesPerFeed))
		return
	}

	firstPerUser := defaultFirstPerUser
	if req.FirstPerUser != nil {
		firstPerUser = clampInt(*req.FirstPerUser, minFirstPerUser, maxFirstPerUser)
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	secret, err := generateSecret()
	if err != nil {
		api.AbortJSONError(c, http.StatusInternalServerError, api.ErrorCodeInternal, "failed to generate secret")
		return
	}

	now := time.Now()
	feed := &store.Feed{
		ID:           uuid.NewString(),
		UserID:       userID,
		Name:         req.Name,
		Secret:       secret,
		Usernames:    validUsernames,
		FirstPerUser: firstPerUser,
		Enabled:      enabled,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := app.store.CreateFeed(c.Request.Context(), feed); err != nil {
		api.AbortJSONError(c, http.StatusInternalServerError, api.ErrorCodeInternal, "failed to create feed")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":             feed.ID,
		"name":           feed.Name,
		"usernames":      feed.Usernames,
		"first_per_user": feed.FirstPerUser,
		"enabled":        feed.Enabled,
		"url":            app.feedURL(feed.ID, feed.Secret),
		"created_at":     feed.CreatedAt.Format(time.RFC3339),
	})
}

func (app *app) getFeed(c *gin.Context) {
	userID, ok := api.GetUserID(c)
	if !ok {
		api.AbortJSONError(c, http.StatusInternalServerError, api.ErrorCodeInternal, "missing user context")
		return
	}

	feedID := c.Param("id")
	feed, err := app.store.GetFeedByID(c.Request.Context(), feedID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			api.AbortJSONError(c, http.StatusNotFound, api.ErrorCodeNotFound, "feed not found")
			return
		}
		api.AbortJSONError(c, http.StatusInternalServerError, api.ErrorCodeInternal, "failed to fetch feed")
		return
	}

	if feed.UserID != userID {
		api.AbortJSONError(c, http.StatusForbidden, api.ErrorCodeForbidden, "you do not own this feed")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":             feed.ID,
		"name":           feed.Name,
		"usernames":      feed.Usernames,
		"first_per_user": feed.FirstPerUser,
		"enabled":        feed.Enabled,
		"url":            app.feedURL(feed.ID, feed.Secret),
		"created_at":     feed.CreatedAt.Format(time.RFC3339),
		"updated_at":     feed.UpdatedAt.Format(time.RFC3339),
	})
}

func (app *app) updateFeed(c *gin.Context) {
	userID, ok := api.GetUserID(c)
	if !ok {
		api.AbortJSONError(c, http.StatusInternalServerError, api.ErrorCodeInternal, "missing user context")
		return
	}

	feedID := c.Param("id")
	feed, err := app.store.GetFeedByID(c.Request.Context(), feedID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			api.AbortJSONError(c, http.StatusNotFound, api.ErrorCodeNotFound, "feed not found")
			return
		}
		api.AbortJSONError(c, http.StatusInternalServerError, api.ErrorCodeInternal, "failed to fetch feed")
		return
	}

	if feed.UserID != userID {
		api.AbortJSONError(c, http.StatusForbidden, api.ErrorCodeForbidden, "you do not own this feed")
		return
	}

	var req struct {
		Name         *string  `json:"name"`
		Usernames    []string `json:"usernames"`
		FirstPerUser *int     `json:"first_per_user"`
		Enabled      *bool    `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		api.AbortJSONErrorWithDetails(c, http.StatusBadRequest, api.ErrorCodeValidation, "invalid request body", err.Error())
		return
	}

	needsCacheInvalidation := false

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			api.AbortJSONError(c, http.StatusBadRequest, api.ErrorCodeValidation, "name cannot be empty")
			return
		}
		if len(name) > maxFeedNameLength {
			api.AbortJSONError(c, http.StatusBadRequest, api.ErrorCodeValidation, fmt.Sprintf("name must be at most %d characters", maxFeedNameLength))
			return
		}
		feed.Name = name
	}

	if req.Usernames != nil {
		if len(req.Usernames) == 0 {
			api.AbortJSONError(c, http.StatusBadRequest, api.ErrorCodeValidation, "at least one username is required")
			return
		}

		validUsernames := make([]string, 0, len(req.Usernames))
		seen := make(map[string]struct{})
		invalidUsernames := make([]string, 0)

		for _, username := range req.Usernames {
			username = strings.TrimSpace(username)
			if username == "" {
				continue
			}
			if err := leetcode.ValidateUsername(username); err != nil {
				invalidUsernames = append(invalidUsernames, username)
				continue
			}
			if _, exists := seen[username]; !exists {
				seen[username] = struct{}{}
				validUsernames = append(validUsernames, username)
			}
		}

		if len(invalidUsernames) > 0 {
			api.AbortJSONErrorWithDetails(c, http.StatusBadRequest, api.ErrorCodeValidation, "invalid usernames", invalidUsernames)
			return
		}

		if len(validUsernames) > app.config.Limits.MaxUsernamesPerFeed {
			api.AbortJSONError(c, http.StatusBadRequest, api.ErrorCodeValidation, fmt.Sprintf("maximum %d usernames per feed", app.config.Limits.MaxUsernamesPerFeed))
			return
		}

		if len(validUsernames) == 0 {
			api.AbortJSONError(c, http.StatusBadRequest, api.ErrorCodeValidation, "at least one valid username is required")
			return
		}

		feed.Usernames = validUsernames
		needsCacheInvalidation = true
	}

	if req.FirstPerUser != nil {
		newFirstPerUser := clampInt(*req.FirstPerUser, minFirstPerUser, maxFirstPerUser)
		if newFirstPerUser != feed.FirstPerUser {
			feed.FirstPerUser = newFirstPerUser
			needsCacheInvalidation = true
		}
	}

	if req.Enabled != nil {
		feed.Enabled = *req.Enabled
	}

	feed.UpdatedAt = time.Now()

	if err := app.store.UpdateFeed(c.Request.Context(), feed); err != nil {
		api.AbortJSONError(c, http.StatusInternalServerError, api.ErrorCodeInternal, "failed to update feed")
		return
	}

	if needsCacheInvalidation {
		_ = app.store.InvalidateFeedCache(c.Request.Context(), feed.ID)
	}

	c.JSON(http.StatusOK, gin.H{
		"id":             feed.ID,
		"name":           feed.Name,
		"usernames":      feed.Usernames,
		"first_per_user": feed.FirstPerUser,
		"enabled":        feed.Enabled,
		"url":            app.feedURL(feed.ID, feed.Secret),
		"created_at":     feed.CreatedAt.Format(time.RFC3339),
		"updated_at":     feed.UpdatedAt.Format(time.RFC3339),
	})
}

func (app *app) rotateFeedSecret(c *gin.Context) {
	userID, ok := api.GetUserID(c)
	if !ok {
		api.AbortJSONError(c, http.StatusInternalServerError, api.ErrorCodeInternal, "missing user context")
		return
	}

	feedID := c.Param("id")
	feed, err := app.store.GetFeedByID(c.Request.Context(), feedID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			api.AbortJSONError(c, http.StatusNotFound, api.ErrorCodeNotFound, "feed not found")
			return
		}
		api.AbortJSONError(c, http.StatusInternalServerError, api.ErrorCodeInternal, "failed to fetch feed")
		return
	}

	if feed.UserID != userID {
		api.AbortJSONError(c, http.StatusForbidden, api.ErrorCodeForbidden, "you do not own this feed")
		return
	}

	newSecret, err := generateSecret()
	if err != nil {
		api.AbortJSONError(c, http.StatusInternalServerError, api.ErrorCodeInternal, "failed to generate secret")
		return
	}

	feed.Secret = newSecret
	feed.UpdatedAt = time.Now()

	if err := app.store.UpdateFeed(c.Request.Context(), feed); err != nil {
		api.AbortJSONError(c, http.StatusInternalServerError, api.ErrorCodeInternal, "failed to update feed")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":             feed.ID,
		"name":           feed.Name,
		"usernames":      feed.Usernames,
		"first_per_user": feed.FirstPerUser,
		"enabled":        feed.Enabled,
		"url":            app.feedURL(feed.ID, feed.Secret),
		"created_at":     feed.CreatedAt.Format(time.RFC3339),
		"updated_at":     feed.UpdatedAt.Format(time.RFC3339),
	})
}

func (app *app) deleteFeed(c *gin.Context) {
	userID, ok := api.GetUserID(c)
	if !ok {
		api.AbortJSONError(c, http.StatusInternalServerError, api.ErrorCodeInternal, "missing user context")
		return
	}

	feedID := c.Param("id")
	feed, err := app.store.GetFeedByID(c.Request.Context(), feedID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			api.AbortJSONError(c, http.StatusNotFound, api.ErrorCodeNotFound, "feed not found")
			return
		}
		api.AbortJSONError(c, http.StatusInternalServerError, api.ErrorCodeInternal, "failed to fetch feed")
		return
	}

	if feed.UserID != userID {
		api.AbortJSONError(c, http.StatusForbidden, api.ErrorCodeForbidden, "you do not own this feed")
		return
	}

	if err := app.store.DeleteFeed(c.Request.Context(), feedID); err != nil {
		api.AbortJSONError(c, http.StatusInternalServerError, api.ErrorCodeInternal, "failed to delete feed")
		return
	}

	c.Status(http.StatusNoContent)
}

func (app *app) feedURL(feedID, secret string) string {
	return fmt.Sprintf("%s/f/%s/%s.xml", app.config.Database.PublicBaseURL, feedID, secret)
}

func generateSecret() (string, error) {
	b := make([]byte, secretBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
