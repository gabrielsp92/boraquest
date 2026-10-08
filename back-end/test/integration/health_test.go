package integration_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/cmd/webapp/routes"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/app/service"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/infrastructure/clock"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/controllers"
)

// newServer wires the app exactly like cmd/webapp/main.go, with real adapters.
func newServer(t *testing.T) *httptest.Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	healthService := service.NewHealthService(clock.SystemClock{}, "test")
	router := routes.NewRouter(routes.Controllers{
		Health: controllers.NewHealthController(healthService),
	})
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)
	return srv
}

func TestGetHealth(t *testing.T) {
	srv := newServer(t)

	res, err := http.Get(srv.URL + "/api/v1/health")
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, res.Header.Get("Content-Type"), "application/json")

	var body controllers.HealthResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&body))
	assert.Equal(t, "up", body.Status)
	assert.Equal(t, "test", body.Version)
	assert.WithinDuration(t, time.Now(), body.CheckedAt, 5*time.Second)
}

func TestUnknownRouteReturns404(t *testing.T) {
	srv := newServer(t)

	res, err := http.Get(srv.URL + "/api/v1/nope")
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, http.StatusNotFound, res.StatusCode)
}
