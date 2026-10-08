package integration_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/controllers"
)

// call sends a JSON request and returns the response; body may be nil.
func call(t *testing.T, method, url, token string, body any) *http.Response {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, r)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { res.Body.Close() })
	return res
}

func decodeBody[T any](t *testing.T, res *http.Response) T {
	t.Helper()
	var v T
	require.NoError(t, json.NewDecoder(res.Body).Decode(&v))
	return v
}

// login signs email in with the dev password and returns the access token.
func login(t *testing.T, baseURL, email string) string {
	t.Helper()
	res := call(t, http.MethodPost, baseURL+"/api/v1/auth/login", "", controllers.LoginRequest{Email: email, Password: devPassword})
	require.Equal(t, http.StatusOK, res.StatusCode)
	return decodeBody[controllers.LoginResponse](t, res).Token
}
