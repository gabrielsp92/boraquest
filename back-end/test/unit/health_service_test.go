package unit_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/app/service"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/health"
)

type fakeClock struct{ now time.Time }

func (f fakeClock) Now() time.Time { return f.now }

func TestHealthServiceCheck(t *testing.T) {
	at := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	svc := service.NewHealthService(fakeClock{now: at}, "1.2.3")

	report, err := svc.Check(context.Background())

	require.NoError(t, err)
	assert.Equal(t, health.StatusUp, report.Status)
	assert.Equal(t, "1.2.3", report.Version)
	assert.Equal(t, at, report.CheckedAt)
}
