package service

import (
	"context"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/user"
)

// ProfileRepository persists a member's own profile settings.
type ProfileRepository interface {
	// UpdateAvatar sets userID's avatar_key and returns the updated user.
	// Returns user.ErrNotFound if userID doesn't exist (defensive — userID always
	// comes from a valid JWT, so this should not happen in practice).
	UpdateAvatar(ctx context.Context, userID, avatarKey string) (user.User, error)
}

// ProfileService manages a signed-in member's own profile.
type ProfileService struct {
	profiles ProfileRepository
}

// NewProfileService builds a ProfileService.
func NewProfileService(profiles ProfileRepository) *ProfileService {
	return &ProfileService{profiles: profiles}
}

// UpdateAvatar validates avatarKey and saves it as callerID's avatar.
func (s *ProfileService) UpdateAvatar(ctx context.Context, callerID, avatarKey string) (user.User, error) {
	if !user.ValidAvatarKey(avatarKey) {
		return user.User{}, user.ErrInvalidAvatar
	}
	return s.profiles.UpdateAvatar(ctx, callerID, avatarKey)
}
