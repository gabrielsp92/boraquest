package controllers

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/push"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/middleware"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/response"
)

// PushUseCase is the use-case the push controller depends on.
type PushUseCase interface {
	PublicKey() string
	Subscribe(ctx context.Context, userID, endpoint, p256dh, auth string) error
	Unsubscribe(ctx context.Context, userID, endpoint string) error
}

// PushKeys holds a subscription's encryption keys, as in PushSubscription.toJSON().
type PushKeys struct {
	P256dh string `json:"p256dh"`
	Auth   string `json:"auth"`
}

// PushSubscriptionRequest is the JSON body of PUT /me/push-subscription: the shape of
// PushSubscription.toJSON(). Extra fields such as expirationTime are ignored.
type PushSubscriptionRequest struct {
	Endpoint string   `json:"endpoint"`
	Keys     PushKeys `json:"keys"`
}

// DeletePushSubscriptionRequest is the JSON body of DELETE /me/push-subscription.
type DeletePushSubscriptionRequest struct {
	Endpoint string `json:"endpoint" binding:"required"`
}

// PublicKeyResponse is the JSON body of GET /push/public-key.
type PublicKeyResponse struct {
	PublicKey string `json:"publicKey"`
}

// PushController serves the Web Push subscription endpoints.
type PushController struct {
	push PushUseCase
}

// NewPushController builds a PushController.
func NewPushController(push PushUseCase) *PushController {
	return &PushController{push: push}
}

// PublicKey handles GET /api/v1/push/public-key.
func (h *PushController) PublicKey(c *gin.Context) {
	response.JSON(c, http.StatusOK, PublicKeyResponse{PublicKey: h.push.PublicKey()})
}

// Subscribe handles PUT /api/v1/me/push-subscription.
func (h *PushController) Subscribe(c *gin.Context) {
	var req PushSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid subscription")
		return
	}
	err := h.push.Subscribe(c.Request.Context(), middleware.UserID(c), req.Endpoint, req.Keys.P256dh, req.Keys.Auth)
	switch {
	case err == nil:
		c.Status(http.StatusNoContent)
	case errors.Is(err, push.ErrInvalidSubscription):
		response.Error(c, http.StatusBadRequest, "invalid subscription")
	default:
		writeInternalError(c, err)
	}
}

// Unsubscribe handles DELETE /api/v1/me/push-subscription. It only ever removes the
// caller's own row, and answers 204 whether or not one existed.
func (h *PushController) Unsubscribe(c *gin.Context) {
	var req DeletePushSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "endpoint is required")
		return
	}
	if err := h.push.Unsubscribe(c.Request.Context(), middleware.UserID(c), req.Endpoint); err != nil {
		writeInternalError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func writeInternalError(c *gin.Context, err error) {
	_ = c.Error(err)
	response.Error(c, http.StatusInternalServerError, "internal error")
}
