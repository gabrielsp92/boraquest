package unit_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/health"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/controllers"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/response"
)

type fakeChecker struct {
	report health.Report
	err    error
}

func (f fakeChecker) Check(context.Context) (health.Report, error) { return f.report, f.err }

func serveHealth(t *testing.T, checker controllers.HealthChecker) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)

	controllers.NewHealthController(checker).Get(c)
	return rec
}

func TestHealthControllerGetOK(t *testing.T) {
	at := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	rec := serveHealth(t, fakeChecker{report: health.Report{Status: health.StatusUp, Version: "1.0.0", CheckedAt: at}})

	require.Equal(t, http.StatusOK, rec.Code)
	var body controllers.HealthResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, controllers.HealthResponse{Status: "up", Version: "1.0.0", CheckedAt: at}, body)
}

func TestHealthControllerGetError(t *testing.T) {
	rec := serveHealth(t, fakeChecker{err: errors.New("boom")})

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	var body response.ErrorBody
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "service unavailable", body.Error)
}
