package unit_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/infrastructure/security"
)

const jwtSecret = "test-secret"

var jwtNow = time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)

func signClaims(t *testing.T, method jwt.SigningMethod, key any, claims jwt.Claims) string {
	t.Helper()
	token, err := jwt.NewWithClaims(method, claims).SignedString(key)
	require.NoError(t, err)
	return token
}

func validClaims() jwt.RegisteredClaims {
	return jwt.RegisteredClaims{
		Subject:   "lia",
		Issuer:    "boraquest",
		IssuedAt:  jwt.NewNumericDate(jwtNow),
		ExpiresAt: jwt.NewNumericDate(jwtNow.Add(time.Hour)),
	}
}

func TestJWTIssueAndVerify(t *testing.T) {
	signer := security.NewJWT(jwtSecret, time.Hour, fakeClock{now: jwtNow})

	token, err := signer.Issue("lia")
	require.NoError(t, err)

	userID, err := signer.Verify(token)
	require.NoError(t, err)
	assert.Equal(t, "lia", userID)
}

func TestJWTExpires(t *testing.T) {
	token, err := security.NewJWT(jwtSecret, time.Hour, fakeClock{now: jwtNow}).Issue("lia")
	require.NoError(t, err)

	later := security.NewJWT(jwtSecret, time.Hour, fakeClock{now: jwtNow.Add(2 * time.Hour)})
	_, err = later.Verify(token)

	assert.ErrorIs(t, err, security.ErrInvalidToken)
}

func TestJWTVerifyRejects(t *testing.T) {
	noSubject := validClaims()
	noSubject.Subject = ""
	otherIssuer := validClaims()
	otherIssuer.Issuer = "someone-else"
	noExpiry := validClaims()
	noExpiry.ExpiresAt = nil

	cases := map[string]string{
		"garbage":       "not-a-token",
		"wrong secret":  signClaims(t, jwt.SigningMethodHS256, []byte("other-secret"), validClaims()),
		"wrong alg":     signClaims(t, jwt.SigningMethodHS512, []byte(jwtSecret), validClaims()),
		"alg none":      signClaims(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, validClaims()),
		"empty subject": signClaims(t, jwt.SigningMethodHS256, []byte(jwtSecret), noSubject),
		"other issuer":  signClaims(t, jwt.SigningMethodHS256, []byte(jwtSecret), otherIssuer),
		"no expiry":     signClaims(t, jwt.SigningMethodHS256, []byte(jwtSecret), noExpiry),
	}
	signer := security.NewJWT(jwtSecret, time.Hour, fakeClock{now: jwtNow})
	for name, token := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := signer.Verify(token)
			assert.ErrorIs(t, err, security.ErrInvalidToken)
		})
	}
}
