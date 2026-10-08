package integration_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/app/service"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/user"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/infrastructure/idgen"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/infrastructure/postgres"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/infrastructure/security"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/controllers"
)

// newSeeder wires the seed service like cmd/seeduser/main.go, against the test database.
func newSeeder() *service.UserSeedService {
	return service.NewUserSeedService(postgres.NewUserRepository(pool), security.BcryptHasher{Cost: bcrypt.MinCost}, idgen.UUIDGenerator{})
}

// The seeded user can sign in through the API; a dry run writes nothing; re-seeding is a no-op.
func TestSeedUserThenLogin(t *testing.T) {
	ctx := context.Background()
	const email = "seeded@boraquest.dev"
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM users WHERE email = $1`, email) })
	in := service.SeedUserInput{Name: "Seeded", Email: "  Seeded@Boraquest.dev ", Password: "seeded-pass", DryRun: true}

	dry, err := newSeeder().Seed(ctx, in)
	require.NoError(t, err)
	assert.Equal(t, service.SeedWouldCreate, dry.Outcome)
	_, err = postgres.NewUserRepository(pool).FindByEmail(ctx, email)
	require.ErrorIs(t, err, user.ErrNotFound, "dry run must not write")

	in.DryRun = false
	created, err := newSeeder().Seed(ctx, in)
	require.NoError(t, err)
	assert.Equal(t, service.SeedCreated, created.Outcome)

	res := call(t, http.MethodPost, newServer(t).URL+"/api/v1/auth/login", "", controllers.LoginRequest{Email: email, Password: "seeded-pass"})
	require.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, controllers.UserResponse{ID: created.User.ID, Name: "Seeded", Email: email}, decodeBody[controllers.LoginResponse](t, res).User)

	in.Password = "another-pass"
	again, err := newSeeder().Seed(ctx, in)
	require.NoError(t, err)
	assert.Equal(t, service.SeedAlreadyExists, again.Outcome)
	assert.Equal(t, created.User, again.User, "existing user, including its password hash, is left untouched")
}
