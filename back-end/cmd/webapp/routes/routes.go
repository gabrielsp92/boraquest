// Package routes registers every HTTP route of the web app.
package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/controllers"
)

// Controllers groups every controller the router needs.
type Controllers struct {
	Health *controllers.HealthController
}

// NewRouter builds the Gin engine with all routes registered.
func NewRouter(c Controllers) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	v1 := r.Group("/api/v1")
	v1.GET("/health", c.Health.Get)

	return r
}
