package api

import (
	"github.com/gin-gonic/gin"
)

const (
	ErrorCodeValidation   = "validation_error"
	ErrorCodeNotFound     = "not_found"
	ErrorCodeUnauthorized = "unauthorized"
	ErrorCodeForbidden    = "forbidden"
	ErrorCodeRateLimited  = "rate_limited"
	ErrorCodeUpstream     = "upstream_error"
	ErrorCodeInternal     = "internal_error"
	ErrorCodeQuota        = "quota_exceeded"
)

type ErrorDetails struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message,omitempty"`
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func JSONError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}

func JSONErrorWithDetails(c *gin.Context, status int, code, message string, details any) {
	if details == nil {
		JSONError(c, status, code, message)
		return
	}
	switch v := details.(type) {
	case []ErrorDetails:
		if len(v) == 0 {
			JSONError(c, status, code, message)
			return
		}
	}
	c.JSON(status, gin.H{
		"error": ErrorResponse{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

func AbortJSONError(c *gin.Context, status int, code, message string) {
	JSONError(c, status, code, message)
	c.Abort()
}

func AbortJSONErrorWithDetails(c *gin.Context, status int, code, message string, details any) {
	JSONErrorWithDetails(c, status, code, message, details)
	c.Abort()
}
