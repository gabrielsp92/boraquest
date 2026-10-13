package unit_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/middleware"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/response"
)

func serveCron(header string) (*httptest.ResponseRecorder, bool) {
	gin.SetMode(gin.TestMode)
	reached := false
	r := gin.New()
	r.POST("/cron", middleware.RequireCronSecret("s3cret"), func(c *gin.Context) {
		reached = true
		c.Status(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodPost, "/cron", nil)
	if header != "" {
		req.Header.Set("Authorization", header)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec, reached
}

func TestRequireCronSecretAccepts(t *testing.T) {
	rec, reached := serveCron("Bearer s3cret")

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, reached)
}

func TestRequireCronSecretRejects(t *testing.T) {
	for name, header := range map[string]string{
		"missing":      "",
		"empty token":  "Bearer ",
		"wrong secret": "Bearer nope",
		"prefix only":  "Bearer s3cre",
		"not bearer":   "Basic s3cret",
	} {
		t.Run(name, func(t *testing.T) {
			rec, reached := serveCron(header)

			assert.Equal(t, http.StatusUnauthorized, rec.Code)
			assert.Equal(t, "unauthorized", decode[response.ErrorBody](t, rec).Error)
			assert.False(t, reached)
		})
	}
}
