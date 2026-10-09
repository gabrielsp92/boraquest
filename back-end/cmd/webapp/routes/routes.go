// Package routes registers every HTTP route of the web app.
package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/controllers"
)

// Controllers groups every controller and middleware the router needs.
type Controllers struct {
	Health  *controllers.HealthController
	Auth    *controllers.AuthController
	Rules   *controllers.RuleController
	Prizes  *controllers.PrizeController
	Entries *controllers.EntryController
	// RequireAuth guards every route that needs a signed-in user.
	RequireAuth gin.HandlerFunc
}

// NewRouter builds the Gin engine with all routes registered.
func NewRouter(c Controllers) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	v1 := r.Group("/api/v1")
	v1.GET("/health", c.Health.Get)
	v1.POST("/auth/login", c.Auth.Login)

	rules := v1.Group("/rules", c.RequireAuth)
	rules.GET("", c.Rules.List)
	rules.POST("", c.Rules.Create)
	rules.GET("/:id", c.Rules.Get)
	rules.PUT("/:id", c.Rules.Update)
	rules.DELETE("/:id", c.Rules.Delete)

	prizes := v1.Group("/prizes", c.RequireAuth)
	prizes.GET("", c.Prizes.Get)
	prizes.PUT("/week", c.Prizes.SetWeek)
	prizes.PUT("/month", c.Prizes.SetMonth)

	entries := v1.Group("/entries", c.RequireAuth)
	entries.GET("", c.Entries.List)
	entries.POST("", c.Entries.Create)
	entries.DELETE("/:id", c.Entries.Delete)

	return r
}
