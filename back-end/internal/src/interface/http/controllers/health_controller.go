// Package controllers holds the HTTP handlers.
package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/health"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/response"
)

// HealthChecker is the use-case the health controller depends on.
type HealthChecker interface {
	Check(ctx context.Context) (health.Report, error)
}

// HealthResponse is the JSON body returned by GET /api/v1/health.
type HealthResponse struct {
	Status    string    `json:"status"`
	Version   string    `json:"version"`
	CheckedAt time.Time `json:"checkedAt"`
}

// HealthController serves the health endpoint.
type HealthController struct {
	checker HealthChecker
}

// NewHealthController builds a HealthController.
func NewHealthController(checker HealthChecker) *HealthController {
	return &HealthController{checker: checker}
}

// Get handles GET /api/v1/health.
func (h *HealthController) Get(c *gin.Context) {
	report, err := h.checker.Check(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusServiceUnavailable, "service unavailable")
		return
	}
	response.JSON(c, http.StatusOK, HealthResponse{
		Status:    string(report.Status),
		Version:   report.Version,
		CheckedAt: report.CheckedAt,
	})
}
