package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/response"
)

// RequireCronSecret rejects requests whose "Authorization: Bearer <secret>" header
// doesn't match secret, compared in constant time. It guards machine-only routes
// called by the scheduler.
func RequireCronSecret(secret string) gin.HandlerFunc {
	want := []byte(secret)
	return func(c *gin.Context) {
		token, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		if !ok || token == "" || subtle.ConstantTimeCompare([]byte(token), want) != 1 {
			response.Error(c, http.StatusUnauthorized, "unauthorized")
			return
		}
		c.Next()
	}
}
