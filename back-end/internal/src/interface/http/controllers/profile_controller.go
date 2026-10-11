package controllers

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/user"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/middleware"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/response"
)

// ProfileUseCase is the use-case the profile controller depends on.
type ProfileUseCase interface {
	UpdateAvatar(ctx context.Context, callerID, avatarKey string) (user.User, error)
}

// UpdateAvatarRequest is the JSON body of PUT /me/avatar.
type UpdateAvatarRequest struct {
	AvatarKey string `json:"avatarKey" binding:"required"`
}

// ProfileController serves the signed-in member's own profile endpoints.
type ProfileController struct {
	profiles ProfileUseCase
}

// NewProfileController builds a ProfileController.
func NewProfileController(profiles ProfileUseCase) *ProfileController {
	return &ProfileController{profiles: profiles}
}

// UpdateAvatar handles PUT /api/v1/me/avatar.
func (h *ProfileController) UpdateAvatar(c *gin.Context) {
	var req UpdateAvatarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "avatarKey is required")
		return
	}
	u, err := h.profiles.UpdateAvatar(c.Request.Context(), middleware.UserID(c), req.AvatarKey)
	if err != nil {
		writeProfileError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, UserResponse{ID: u.ID, Name: u.Name, Email: u.Email, AvatarKey: u.AvatarKey})
}

func writeProfileError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, user.ErrInvalidAvatar):
		response.Error(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, user.ErrNotFound):
		response.Error(c, http.StatusNotFound, "user not found")
	default:
		_ = c.Error(err)
		response.Error(c, http.StatusInternalServerError, "internal error")
	}
}
