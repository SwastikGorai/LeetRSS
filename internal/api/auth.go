package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"leetcode-rss/internal/store"

	"github.com/clerk/clerk-sdk-go/v2"
	clerkhttp "github.com/clerk/clerk-sdk-go/v2/http"
	"github.com/clerk/clerk-sdk-go/v2/user"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ctxKey string

const userIDKey ctxKey = "userID"

func ClerkAuthMiddleware(s store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		authFailed := false
		authFailureHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authFailed = true
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":{"code":"` + ErrorCodeUnauthorized + `","message":"Authentication required"}}`))
		})

		var updatedReq *http.Request
		captureRequest := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			updatedReq = r
		})

		clerkHandler := clerkhttp.WithHeaderAuthorization(
			clerkhttp.AuthorizationFailureHandler(authFailureHandler),
		)(captureRequest)

		clerkHandler.ServeHTTP(c.Writer, c.Request)

		if authFailed {
			c.Abort()
			return
		}

		if updatedReq != nil {
			c.Request = updatedReq
		}

		claims, ok := clerk.SessionClaimsFromContext(c.Request.Context())
		if !ok {
			AbortJSONError(c, http.StatusUnauthorized, ErrorCodeUnauthorized, "invalid token")
			return
		}

		localUser, err := getOrCreateUser(c.Request.Context(), s, claims.Subject)
		if err != nil {
			log.Printf("error provisioning user %s: %v", claims.Subject, err)
			AbortJSONError(c, http.StatusInternalServerError, ErrorCodeInternal, "failed to provision user")
			return
		}

		c.Set(string(userIDKey), localUser.ID)
		c.Next()
	}
}

func GetUserID(c *gin.Context) (string, bool) {
	val, ok := c.Get(string(userIDKey))
	if !ok {
		return "", false
	}
	userID, ok := val.(string)
	return userID, ok
}

func getOrCreateUser(ctx context.Context, s store.Store, clerkUserID string) (*store.User, error) {
	u, err := s.GetUserByProvider(ctx, "clerk", clerkUserID)
	if err == nil {
		return u, nil
	}
	if !errors.Is(err, store.ErrNotFound) {
		return nil, fmt.Errorf("lookup user by provider: %w", err)
	}

	clerkUser, err := user.Get(ctx, clerkUserID)
	if err != nil {
		return nil, fmt.Errorf("fetch clerk user: %w", err)
	}

	email := ""
	if len(clerkUser.EmailAddresses) > 0 {
		email = clerkUser.EmailAddresses[0].EmailAddress
	}
	if email == "" {
		return nil, fmt.Errorf("clerk user %s has no email address", clerkUserID)
	}

	provider := "clerk"
	now := time.Now()
	newUser := &store.User{
		ID:              uuid.NewString(),
		Email:           email,
		AuthProvider:    &provider,
		ProviderSubject: &clerkUserID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.CreateUser(ctx, newUser); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	log.Printf("provisioned new user: id=%s email=%s clerk_id=%s", newUser.ID, newUser.Email, clerkUserID)
	return newUser, nil
}
