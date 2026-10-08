package controllers

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/user"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/response"
)

// Authenticator is the use-case the auth controller depends on.
type Authenticator interface {
	Login(ctx context.Context, email, password string) (string, user.User, error)
}

// LoginRequest is the JSON body of POST /auth/login.
type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// UserResponse is the public JSON representation of a user.
type UserResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// LoginResponse is the JSON body returned by POST /auth/login.
type LoginResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

// AuthController serves the authentication endpoints.
type AuthController struct {
	auth Authenticator
}

// NewAuthController builds an AuthController.
func NewAuthController(auth Authenticator) *AuthController {
	return &AuthController{auth: auth}
}

// Login handles POST /api/v1/auth/login.
func (h *AuthController) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "email and password are required")
		return
	}
	token, u, err := h.auth.Login(c.Request.Context(), req.Email, req.Password)
	switch {
	case errors.Is(err, user.ErrInvalidCredentials):
		response.Error(c, http.StatusUnauthorized, "invalid email or password")
	case err != nil:
		_ = c.Error(err)
		response.Error(c, http.StatusInternalServerError, "internal error")
	default:
		response.JSON(c, http.StatusOK, LoginResponse{Token: token, User: UserResponse{ID: u.ID, Name: u.Name, Email: u.Email}})
	}
}
