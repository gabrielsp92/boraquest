package unit_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/health"
)

func TestStatusValidate(t *testing.T) {
	assert.NoError(t, health.StatusUp.Validate())
	assert.NoError(t, health.StatusDown.Validate())
	assert.ErrorIs(t, health.Status("sideways").Validate(), health.ErrInvalidStatus)
}

func TestNewReport(t *testing.T) {
	at := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)

	report, err := health.NewReport(health.StatusUp, "1.0.0", at)
	require.NoError(t, err)
	assert.Equal(t, health.Report{Status: health.StatusUp, Version: "1.0.0", CheckedAt: at}, report)

	_, err = health.NewReport("bogus", "1.0.0", at)
	assert.ErrorIs(t, err, health.ErrInvalidStatus)
}
