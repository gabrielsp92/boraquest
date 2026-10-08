// Package service contains the application use-cases.
package service

import (
	"context"
	"time"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/health"
)

// Clock abstracts the current time so use-cases stay deterministic in tests.
type Clock interface {
	Now() time.Time
}

// HealthService reports the health of the running service.
type HealthService struct {
	clock   Clock
	version string
}

// NewHealthService builds a HealthService.
func NewHealthService(clock Clock, version string) *HealthService {
	return &HealthService{clock: clock, version: version}
}

// Check returns the current health report.
func (s *HealthService) Check(_ context.Context) (health.Report, error) {
	return health.NewReport(health.StatusUp, s.version, s.clock.Now())
}
