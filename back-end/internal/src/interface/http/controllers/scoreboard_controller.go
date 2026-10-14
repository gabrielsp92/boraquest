package controllers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/app/service"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/entry"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/guild"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/rule"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/middleware"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/response"
)

// ScoreboardUseCase is the use-case the scoreboard controller depends on.
type ScoreboardUseCase interface {
	List(ctx context.Context, callerID string, period service.ScoreboardPeriod) (standings []service.Standing, from, to, today time.Time, err error)
	Breakdown(ctx context.Context, callerID, memberID string, period service.ScoreboardPeriod) (items []service.QuestTotal, from, to, today time.Time, err error)
}

// StandingResponse is the JSON representation of one ranked row.
type StandingResponse struct {
	MemberID  string `json:"memberId"`
	Points    int    `json:"points"`
	Completed int    `json:"completed"`
}

// ScoreboardResponse is the JSON body of GET /scoreboard.
type ScoreboardResponse struct {
	Standings []StandingResponse `json:"standings"`
	From      string             `json:"from"`
	To        string             `json:"to"`
	Today     string             `json:"today"`
}

// QuestTotalResponse is the JSON representation of one quest's aggregate.
type QuestTotalResponse struct {
	RuleID    string `json:"ruleId"`
	RuleName  string `json:"ruleName"`
	ScoreType string `json:"scoreType"`
	Count     int    `json:"count"`
	Points    int    `json:"points"`
}

// ScoreBreakdownResponse is the JSON body of GET /scoreboard/members/:memberId.
type ScoreBreakdownResponse struct {
	MemberID  string               `json:"memberId"`
	Items     []QuestTotalResponse `json:"items"`
	Total     int                  `json:"total"`
	Completed int                  `json:"completed"`
	From      string               `json:"from"`
	To        string               `json:"to"`
	Today     string               `json:"today"`
}

// ScoreboardController serves the scoreboard endpoint.
type ScoreboardController struct {
	scoreboard ScoreboardUseCase
}

// NewScoreboardController builds a ScoreboardController.
func NewScoreboardController(scoreboard ScoreboardUseCase) *ScoreboardController {
	return &ScoreboardController{scoreboard: scoreboard}
}

// List handles GET /api/v1/scoreboard.
func (h *ScoreboardController) List(c *gin.Context) {
	period, ok := bindScoreboardPeriod(c)
	if !ok {
		return
	}
	standings, from, to, today, err := h.scoreboard.List(c.Request.Context(), middleware.UserID(c), period)
	if err != nil {
		writeScoreboardError(c, err)
		return
	}
	body := ScoreboardResponse{
		Standings: make([]StandingResponse, 0, len(standings)),
		From:      from.Format(dateLayout),
		To:        to.Format(dateLayout),
		Today:     today.Format(dateLayout),
	}
	for _, s := range standings {
		body.Standings = append(body.Standings, StandingResponse{MemberID: s.MemberID, Points: s.Points, Completed: s.Completed})
	}
	response.JSON(c, http.StatusOK, body)
}

// Breakdown handles GET /api/v1/scoreboard/members/:memberId.
func (h *ScoreboardController) Breakdown(c *gin.Context) {
	period, ok := bindScoreboardPeriod(c)
	if !ok {
		return
	}
	memberID := c.Param("memberId")
	items, from, to, today, err := h.scoreboard.Breakdown(c.Request.Context(), middleware.UserID(c), memberID, period)
	if err != nil {
		writeScoreboardError(c, err)
		return
	}
	body := ScoreBreakdownResponse{
		MemberID: memberID,
		Items:    make([]QuestTotalResponse, 0, len(items)),
		From:     from.Format(dateLayout),
		To:       to.Format(dateLayout),
		Today:    today.Format(dateLayout),
	}
	for _, it := range items {
		body.Items = append(body.Items, QuestTotalResponse{RuleID: it.RuleID, RuleName: it.RuleName, ScoreType: string(it.ScoreType), Count: it.Count, Points: it.Points})
		body.Total += it.Points
		if it.ScoreType == rule.ScoreTypeSum {
			body.Completed += it.Count
		}
	}
	response.JSON(c, http.StatusOK, body)
}

func bindScoreboardPeriod(c *gin.Context) (service.ScoreboardPeriod, bool) {
	period := service.ScoreboardPeriod(c.Query("period"))
	switch period {
	case service.ScoreboardWeek, service.ScoreboardMonth:
		return period, true
	default:
		response.Error(c, http.StatusBadRequest, "period must be \"week\" or \"month\"")
		return "", false
	}
}

func writeScoreboardError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, guild.ErrNotMember):
		response.Error(c, http.StatusForbidden, "you are not a member of any guild")
	case errors.Is(err, entry.ErrMemberNotInGuild):
		response.Error(c, http.StatusNotFound, "member not found in your guild")
	default:
		_ = c.Error(err)
		response.Error(c, http.StatusInternalServerError, "internal error")
	}
}
