package unit_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/infrastructure/clock"
)

func TestSystemClockNow(t *testing.T) {
	before := time.Now()
	now := clock.SystemClock{}.Now()

	assert.Equal(t, time.UTC, now.Location())
	assert.False(t, now.Before(before.Add(-time.Second)))
	assert.False(t, now.After(time.Now().Add(time.Second)))
}
