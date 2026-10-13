package controllers

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/guild"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/weekresult"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/middleware"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/response"
)

// WeekHistoryUseCase is the use-case the week history controller depends on.
type WeekHistoryUseCase interface {
	List(ctx context.Context, callerID string) ([]weekresult.WeekResult, error)
}

// WeekStandingResponse is the JSON representation of one frozen standing.
type WeekStandingResponse struct {
	MemberID  string `json:"memberId"`
	Rank      int    `json:"rank"`
	Points    int    `json:"points"`
	Completed int    `json:"completed"`
}

// WeekResultResponse is the JSON representation of one closed week.
type WeekResultResponse struct {
	From      string                 `json:"from"`
	To        string                 `json:"to"`
	Prize     string                 `json:"prize"`
	Winner    WeekStandingResponse   `json:"winner"`
	Standings []WeekStandingResponse `json:"standings"`
}

// WeekHistoryResponse is the JSON body of GET /weeks/history.
type WeekHistoryResponse struct {
	Weeks []WeekResultResponse `json:"weeks"`
}

// WeekHistoryController serves the closed-weeks history endpoint.
type WeekHistoryController struct {
	history WeekHistoryUseCase
}

// NewWeekHistoryController builds a WeekHistoryController.
func NewWeekHistoryController(history WeekHistoryUseCase) *WeekHistoryController {
	return &WeekHistoryController{history: history}
}

// List handles GET /api/v1/weeks/history.
func (h *WeekHistoryController) List(c *gin.Context) {
	weeks, err := h.history.List(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		writeWeekHistoryError(c, err)
		return
	}
	body := WeekHistoryResponse{Weeks: make([]WeekResultResponse, 0, len(weeks))}
	for _, w := range weeks {
		item := WeekResultResponse{
			From:      w.WeekStart.Format(dateLayout),
			To:        w.WeekEnd.Format(dateLayout),
			Prize:     w.Prize,
			Standings: make([]WeekStandingResponse, 0, len(w.Standings)),
		}
		for _, s := range w.Standings {
			item.Standings = append(item.Standings, toWeekStandingResponse(s))
		}
		if winner, ok := w.Winner(); ok {
			item.Winner = toWeekStandingResponse(winner)
		}
		body.Weeks = append(body.Weeks, item)
	}
	response.JSON(c, http.StatusOK, body)
}

func toWeekStandingResponse(s weekresult.Standing) WeekStandingResponse {
	return WeekStandingResponse{MemberID: s.MemberID, Rank: s.Rank, Points: s.Points, Completed: s.Completed}
}

func writeWeekHistoryError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, guild.ErrNotMember):
		response.Error(c, http.StatusForbidden, "you are not a member of any guild")
	default:
		_ = c.Error(err)
		response.Error(c, http.StatusInternalServerError, "internal error")
	}
}
