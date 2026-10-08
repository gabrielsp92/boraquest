package unit_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/middleware"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/response"
)

// fakeVerifier accepts only the token "good" and maps it to userID.
type fakeVerifier struct{ userID string }

func (f fakeVerifier) Verify(token string) (string, error) {
	if token != "good" {
		return "", errBoom
	}
	return f.userID, nil
}

func serveWithAuth(t *testing.T, authorization string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/me", middleware.RequireAuth(fakeVerifier{userID: "lia"}), func(c *gin.Context) {
		c.String(http.StatusOK, middleware.UserID(c))
	})
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestRequireAuthValidToken(t *testing.T) {
	rec := serveWithAuth(t, "Bearer good")

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "lia", rec.Body.String())
}

func TestRequireAuthRejects(t *testing.T) {
	cases := map[string]struct{ header, message string }{
		"no header":     {"", "missing bearer token"},
		"wrong scheme":  {"Basic good", "missing bearer token"},
		"empty token":   {"Bearer ", "missing bearer token"},
		"invalid token": {"Bearer bad", "invalid token"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			rec := serveWithAuth(t, tc.header)

			require.Equal(t, http.StatusUnauthorized, rec.Code)
			var body response.ErrorBody
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			assert.Equal(t, tc.message, body.Error)
		})
	}
}
