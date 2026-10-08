// Package response holds the HTTP response helpers shared by all controllers.
package response

import "github.com/gin-gonic/gin"

// ErrorBody is the JSON shape of every error response.
type ErrorBody struct {
	Error string `json:"error"`
}

// JSON writes a success payload.
func JSON(c *gin.Context, status int, payload any) {
	c.JSON(status, payload)
}

// Error writes an error payload.
func Error(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, ErrorBody{Error: message})
}
