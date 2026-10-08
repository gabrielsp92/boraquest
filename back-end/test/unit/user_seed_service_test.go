package unit_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/app/service"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/guild"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/user"
)

type fakeUserStore struct {
	found     user.User
	findErr   error
	createErr error
	email     string
	created   []user.User
	guildIDs  []string
}

func (f *fakeUserStore) FindByEmail(_ context.Context, email string) (user.User, error) {
	f.email = email
	return f.found, f.findErr
}

func (f *fakeUserStore) Create(_ context.Context, u user.User, guildID string) error {
	f.created = append(f.created, u)
	f.guildIDs = append(f.guildIDs, guildID)
	return f.createErr
}

type fakeGuildFinder struct {
	err error
	id  string
}

func (f *fakeGuildFinder) Get(_ context.Context, id string) (guild.Guild, error) {
	f.id = id
	return testGuild, f.err
}

type fakeHasher struct {
	err   error
	calls int
}

func (f *fakeHasher) Hash(password string) (string, error) {
	f.calls++
	return "hashed:" + password, f.err
}

var newcomer = service.SeedUserInput{Name: " Ana ", Email: " ANA@boraquest.dev ", Password: "s3cret-pass", GuildID: " g1 "}

func TestUserSeedServiceCreates(t *testing.T) {
	store := &fakeUserStore{findErr: user.ErrNotFound}
	guilds := &fakeGuildFinder{}
	svc := service.NewUserSeedService(store, guilds, &fakeHasher{}, fakeIDs{id: "u1"})

	res, err := svc.Seed(context.Background(), newcomer)

	require.NoError(t, err)
	want := user.User{ID: "u1", Name: "Ana", Email: "ana@boraquest.dev", PasswordHash: "hashed:s3cret-pass"}
	assert.Equal(t, service.SeedUserResult{User: want, Guild: testGuild, Outcome: service.SeedCreated}, res)
	assert.Equal(t, "g1", guilds.id, "guild id is trimmed")
	assert.Equal(t, "ana@boraquest.dev", store.email)
	assert.Equal(t, []user.User{want}, store.created)
	assert.Equal(t, []string{testGuild.ID}, store.guildIDs)
}

func TestUserSeedServiceDryRun(t *testing.T) {
	store := &fakeUserStore{findErr: user.ErrNotFound}
	hasher := &fakeHasher{}
	svc := service.NewUserSeedService(store, &fakeGuildFinder{}, hasher, fakeIDs{id: "u1"})
	in := newcomer
	in.DryRun = true

	res, err := svc.Seed(context.Background(), in)

	require.NoError(t, err)
	assert.Equal(t, service.SeedWouldCreate, res.Outcome)
	assert.Equal(t, testGuild, res.Guild)
	assert.Equal(t, user.User{ID: "u1", Name: "Ana", Email: "ana@boraquest.dev"}, res.User)
	assert.Empty(t, store.created, "a dry run must not write")
	assert.Zero(t, hasher.calls)
}

func TestUserSeedServiceAlreadyExists(t *testing.T) {
	for _, dryRun := range []bool{false, true} {
		store := &fakeUserStore{found: lia}
		hasher := &fakeHasher{}
		svc := service.NewUserSeedService(store, &fakeGuildFinder{}, hasher, fakeIDs{id: "u1"})
		in := newcomer
		in.DryRun = dryRun

		res, err := svc.Seed(context.Background(), in)

		require.NoError(t, err)
		assert.Equal(t, service.SeedUserResult{User: lia, Outcome: service.SeedAlreadyExists}, res)
		assert.Empty(t, store.created)
		assert.Zero(t, hasher.calls)
	}
}

func TestUserSeedServiceInvalidInput(t *testing.T) {
	cases := map[string]struct {
		in   service.SeedUserInput
		want error
	}{
		"short password": {service.SeedUserInput{Name: "Ana", Email: "ana@b.dev", Password: "short"}, user.ErrInvalidPassword},
		"blank name":     {service.SeedUserInput{Name: " ", Email: "ana@b.dev", Password: "s3cret-pass"}, user.ErrInvalidName},
		"bad email":      {service.SeedUserInput{Name: "Ana", Email: "nope", Password: "s3cret-pass"}, user.ErrInvalidEmail},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			store := &fakeUserStore{findErr: user.ErrNotFound}
			svc := service.NewUserSeedService(store, &fakeGuildFinder{}, &fakeHasher{}, fakeIDs{id: "u1"})

			_, err := svc.Seed(context.Background(), tc.in)

			assert.ErrorIs(t, err, tc.want)
			assert.Empty(t, store.email, "must not hit the repository")
		})
	}
}

func TestUserSeedServiceGuildError(t *testing.T) {
	for _, dryRun := range []bool{false, true} {
		store := &fakeUserStore{findErr: user.ErrNotFound}
		svc := service.NewUserSeedService(store, &fakeGuildFinder{err: guild.ErrNotFound}, &fakeHasher{}, fakeIDs{id: "u1"})
		in := newcomer
		in.DryRun = dryRun

		_, err := svc.Seed(context.Background(), in)

		assert.ErrorIs(t, err, guild.ErrNotFound)
		assert.Empty(t, store.created)
	}
}

func TestUserSeedServiceFindError(t *testing.T) {
	store := &fakeUserStore{findErr: errBoom}
	svc := service.NewUserSeedService(store, &fakeGuildFinder{}, &fakeHasher{}, fakeIDs{id: "u1"})

	_, err := svc.Seed(context.Background(), newcomer)

	assert.ErrorIs(t, err, errBoom)
	assert.Empty(t, store.created)
}

func TestUserSeedServiceHashError(t *testing.T) {
	store := &fakeUserStore{findErr: user.ErrNotFound}
	svc := service.NewUserSeedService(store, &fakeGuildFinder{}, &fakeHasher{err: errBoom}, fakeIDs{id: "u1"})

	_, err := svc.Seed(context.Background(), newcomer)

	assert.ErrorIs(t, err, errBoom)
	assert.Empty(t, store.created)
}

func TestUserSeedServiceCreateError(t *testing.T) {
	store := &fakeUserStore{findErr: user.ErrNotFound, createErr: user.ErrAlreadyExists}
	svc := service.NewUserSeedService(store, &fakeGuildFinder{}, &fakeHasher{}, fakeIDs{id: "u1"})

	_, err := svc.Seed(context.Background(), newcomer)

	assert.ErrorIs(t, err, user.ErrAlreadyExists)
}
