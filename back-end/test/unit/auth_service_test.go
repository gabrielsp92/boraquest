package unit_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/app/service"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/user"
)

type fakeUsers struct {
	user  user.User
	err   error
	email string
}

func (f *fakeUsers) FindByEmail(_ context.Context, email string) (user.User, error) {
	f.email = email
	return f.user, f.err
}

type fakeComparer struct {
	err    error
	hashes []string
}

func (f *fakeComparer) Compare(hash, _ string) error {
	f.hashes = append(f.hashes, hash)
	return f.err
}

type fakeIssuer struct {
	token string
	err   error
}

func (f fakeIssuer) Issue(userID string) (string, error) { return f.token + ":" + userID, f.err }

var lia = user.User{ID: "lia", Name: "Lia", Email: "lia@boraquest.dev", PasswordHash: "hash"}

func TestAuthServiceLogin(t *testing.T) {
	users := &fakeUsers{user: lia}
	comparer := &fakeComparer{}
	svc := service.NewAuthService(users, comparer, fakeIssuer{token: "tok"})

	token, u, err := svc.Login(context.Background(), "  LIA@boraquest.dev ", "secret")

	require.NoError(t, err)
	assert.Equal(t, "tok:lia", token)
	assert.Equal(t, lia, u)
	assert.Equal(t, "lia@boraquest.dev", users.email)
	assert.Equal(t, []string{"hash"}, comparer.hashes)
}

func TestAuthServiceLoginUnknownEmail(t *testing.T) {
	comparer := &fakeComparer{}
	svc := service.NewAuthService(&fakeUsers{err: user.ErrNotFound}, comparer, fakeIssuer{})

	_, _, err := svc.Login(context.Background(), "nobody@boraquest.dev", "secret")

	assert.ErrorIs(t, err, user.ErrInvalidCredentials)
	assert.Len(t, comparer.hashes, 1, "must still compare a hash to keep timing constant")
}

func TestAuthServiceLoginRepoError(t *testing.T) {
	svc := service.NewAuthService(&fakeUsers{err: errBoom}, &fakeComparer{}, fakeIssuer{})

	_, _, err := svc.Login(context.Background(), "lia@boraquest.dev", "secret")

	assert.ErrorIs(t, err, errBoom)
}

func TestAuthServiceLoginWrongPassword(t *testing.T) {
	svc := service.NewAuthService(&fakeUsers{user: lia}, &fakeComparer{err: errBoom}, fakeIssuer{})

	_, _, err := svc.Login(context.Background(), "lia@boraquest.dev", "wrong")

	assert.ErrorIs(t, err, user.ErrInvalidCredentials)
}

func TestAuthServiceLoginIssuerError(t *testing.T) {
	svc := service.NewAuthService(&fakeUsers{user: lia}, &fakeComparer{}, fakeIssuer{err: errBoom})

	_, _, err := svc.Login(context.Background(), "lia@boraquest.dev", "secret")

	assert.ErrorIs(t, err, errBoom)
}
