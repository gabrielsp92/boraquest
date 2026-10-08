package security

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const issuer = "boraquest"

// ErrInvalidToken is returned for any token that fails verification.
var ErrInvalidToken = errors.New("security: invalid token")

// Clock abstracts the current time so token expiry is testable.
type Clock interface {
	Now() time.Time
}

// JWT issues and verifies HS256-signed access tokens whose subject is the user id.
type JWT struct {
	secret []byte
	ttl    time.Duration
	clock  Clock
}

// NewJWT builds a JWT signer valid for ttl after issuance.
func NewJWT(secret string, ttl time.Duration, clock Clock) *JWT {
	return &JWT{secret: []byte(secret), ttl: ttl, clock: clock}
}

// Issue returns a signed token for userID.
func (j *JWT) Issue(userID string) (string, error) {
	now := j.clock.Now()
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		Issuer:    issuer,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(j.ttl)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(j.secret)
}

// Verify checks token and returns the user id it was issued for.
func (j *JWT) Verify(token string) (string, error) {
	var claims jwt.RegisteredClaims
	_, err := jwt.ParseWithClaims(token, &claims,
		func(*jwt.Token) (any, error) { return j.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(issuer),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(j.clock.Now),
	)
	if err != nil || claims.Subject == "" {
		return "", ErrInvalidToken
	}
	return claims.Subject, nil
}
