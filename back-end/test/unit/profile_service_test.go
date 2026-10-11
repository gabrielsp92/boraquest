package unit_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/app/service"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/user"
)

type fakeProfileRepo struct {
	updated user.User
	err     error

	gotUserID, gotAvatarKey string
	called                  bool
}

func (f *fakeProfileRepo) UpdateAvatar(_ context.Context, userID, avatarKey string) (user.User, error) {
	f.called = true
	f.gotUserID, f.gotAvatarKey = userID, avatarKey
	return f.updated, f.err
}

func TestProfileServiceUpdateAvatar(t *testing.T) {
	repo := &fakeProfileRepo{updated: user.User{ID: "lia", AvatarKey: "duda"}}
	svc := service.NewProfileService(repo)

	u, err := svc.UpdateAvatar(context.Background(), "lia", "duda")

	require.NoError(t, err)
	assert.Equal(t, user.User{ID: "lia", AvatarKey: "duda"}, u)
	assert.Equal(t, "lia", repo.gotUserID)
	assert.Equal(t, "duda", repo.gotAvatarKey)
}

func TestProfileServiceUpdateAvatarRejectsUnknownKey(t *testing.T) {
	repo := &fakeProfileRepo{}
	svc := service.NewProfileService(repo)

	_, err := svc.UpdateAvatar(context.Background(), "lia", "astolfo")

	assert.ErrorIs(t, err, user.ErrInvalidAvatar)
	assert.False(t, repo.called, "an invalid key must short-circuit before hitting the repository")
}

func TestProfileServiceUpdateAvatarRepoError(t *testing.T) {
	repo := &fakeProfileRepo{err: errBoom}
	svc := service.NewProfileService(repo)

	_, err := svc.UpdateAvatar(context.Background(), "lia", "duda")

	assert.ErrorIs(t, err, errBoom)
}
