package controllers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/guild"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/prize"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/middleware"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/response"
)

// PrizeUseCase is the use-case the prize controller depends on.
type PrizeUseCase interface {
	Get(ctx context.Context, userID string) (prize.Prizes, error)
	Set(ctx context.Context, userID string, period prize.Period, text string) (prize.Prizes, error)
}

// SetPrizeRequest is the JSON body of PUT /prizes/week and PUT /prizes/month.
type SetPrizeRequest struct {
	Text string `json:"text"`
}

// PrizeResponse is the JSON representation of a guild's prizes.
type PrizeResponse struct {
	Week      string    `json:"week"`
	Month     string    `json:"month"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// PrizeController serves the prizes endpoints.
type PrizeController struct {
	prizes PrizeUseCase
}

// NewPrizeController builds a PrizeController.
func NewPrizeController(prizes PrizeUseCase) *PrizeController {
	return &PrizeController{prizes: prizes}
}

// Get handles GET /api/v1/prizes.
func (h *PrizeController) Get(c *gin.Context) {
	p, err := h.prizes.Get(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		writePrizeError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, toPrizeResponse(p))
}

// SetWeek handles PUT /api/v1/prizes/week.
func (h *PrizeController) SetWeek(c *gin.Context) {
	h.set(c, prize.PeriodWeek)
}

// SetMonth handles PUT /api/v1/prizes/month.
func (h *PrizeController) SetMonth(c *gin.Context) {
	h.set(c, prize.PeriodMonth)
}

func (h *PrizeController) set(c *gin.Context, period prize.Period) {
	var req SetPrizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	p, err := h.prizes.Set(c.Request.Context(), middleware.UserID(c), period, req.Text)
	if err != nil {
		writePrizeError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, toPrizeResponse(p))
}

func toPrizeResponse(p prize.Prizes) PrizeResponse {
	return PrizeResponse{Week: p.Week, Month: p.Month, UpdatedAt: p.UpdatedAt}
}

func writePrizeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, guild.ErrNotMember):
		response.Error(c, http.StatusForbidden, "you are not a member of any guild")
	case errors.Is(err, prize.ErrTooLong):
		response.Error(c, http.StatusBadRequest, err.Error())
	default:
		_ = c.Error(err)
		response.Error(c, http.StatusInternalServerError, "internal error")
	}
}
