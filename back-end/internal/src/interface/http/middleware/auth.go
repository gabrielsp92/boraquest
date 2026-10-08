// Package middleware holds the Gin middlewares shared by the routes.
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/response"
)

const userIDKey = "auth.userID"

// TokenVerifier checks an access token and returns the user id it identifies.
type TokenVerifier interface {
	Verify(token string) (string, error)
}

// RequireAuth rejects requests without a valid "Authorization: Bearer <token>"
// header and stores the caller's user id for UserID.
func RequireAuth(verifier TokenVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		if !ok || token == "" {
			response.Error(c, http.StatusUnauthorized, "missing bearer token")
			return
		}
		userID, err := verifier.Verify(token)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, "invalid token")
			return
		}
		c.Set(userIDKey, userID)
		c.Next()
	}
}

// UserID returns the authenticated user id set by RequireAuth.
func UserID(c *gin.Context) string {
	return c.GetString(userIDKey)
}
