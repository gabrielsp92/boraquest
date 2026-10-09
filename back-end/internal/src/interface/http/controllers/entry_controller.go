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

const dateLayout = "2006-01-02"

// EntryUseCase is the use-case the entry controller depends on.
type EntryUseCase interface {
	Create(ctx context.Context, callerID string, in service.EntryInput) (entry.Entry, error)
	Delete(ctx context.Context, callerID, id string) error
	List(ctx context.Context, callerID string, period service.EntryPeriod) (entries []entry.Entry, from, to, today time.Time, err error)
}

// EntryRequest is the JSON body of POST /entries.
type EntryRequest struct {
	RuleID   string `json:"ruleId"`
	MemberID string `json:"memberId"`
}

// EntryResponse is the JSON representation of an entry.
type EntryResponse struct {
	ID         string    `json:"id"`
	RuleID     string    `json:"ruleId"`
	RuleName   string    `json:"ruleName"`
	ScoreType  string    `json:"scoreType"`
	Points     int       `json:"points"`
	MemberID   string    `json:"memberId"`
	LoggedBy   string    `json:"loggedBy"`
	OccurredOn string    `json:"occurredOn"`
	CreatedAt  time.Time `json:"createdAt"`
}

// EntryListResponse is the JSON body of GET /entries.
type EntryListResponse struct {
	Entries []EntryResponse `json:"entries"`
	From    string          `json:"from"`
	To      string          `json:"to"`
	Today   string          `json:"today"`
}

// EntryController serves the entries endpoints.
type EntryController struct {
	entries EntryUseCase
}

// NewEntryController builds an EntryController.
func NewEntryController(entries EntryUseCase) *EntryController {
	return &EntryController{entries: entries}
}

// List handles GET /api/v1/entries.
func (h *EntryController) List(c *gin.Context) {
	period, ok := bindEntryPeriod(c)
	if !ok {
		return
	}
	entries, from, to, today, err := h.entries.List(c.Request.Context(), middleware.UserID(c), period)
	if err != nil {
		writeEntryError(c, err)
		return
	}
	body := EntryListResponse{
		Entries: make([]EntryResponse, 0, len(entries)),
		From:    from.Format(dateLayout),
		To:      to.Format(dateLayout),
		Today:   today.Format(dateLayout),
	}
	for _, e := range entries {
		body.Entries = append(body.Entries, toEntryResponse(e))
	}
	response.JSON(c, http.StatusOK, body)
}

// Create handles POST /api/v1/entries.
func (h *EntryController) Create(c *gin.Context) {
	in, ok := bindEntryInput(c)
	if !ok {
		return
	}
	e, err := h.entries.Create(c.Request.Context(), middleware.UserID(c), in)
	if err != nil {
		writeEntryError(c, err)
		return
	}
	response.JSON(c, http.StatusCreated, toEntryResponse(e))
}

// Delete handles DELETE /api/v1/entries/:id.
func (h *EntryController) Delete(c *gin.Context) {
	if err := h.entries.Delete(c.Request.Context(), middleware.UserID(c), c.Param("id")); err != nil {
		writeEntryError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func bindEntryPeriod(c *gin.Context) (service.EntryPeriod, bool) {
	period := service.EntryPeriod(c.Query("period"))
	switch period {
	case service.PeriodToday, service.PeriodWeek:
		return period, true
	default:
		response.Error(c, http.StatusBadRequest, "period must be \"today\" or \"week\"")
		return "", false
	}
}

func bindEntryInput(c *gin.Context) (service.EntryInput, bool) {
	var req EntryRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.RuleID == "" || req.MemberID == "" {
		response.Error(c, http.StatusBadRequest, "ruleId and memberId are required")
		return service.EntryInput{}, false
	}
	return service.EntryInput{RuleID: req.RuleID, MemberID: req.MemberID}, true
}

func toEntryResponse(e entry.Entry) EntryResponse {
	return EntryResponse{
		ID:         e.ID,
		RuleID:     e.RuleID,
		RuleName:   e.RuleName,
		ScoreType:  string(e.ScoreType),
		Points:     e.Points,
		MemberID:   e.MemberID,
		LoggedBy:   e.LoggedBy,
		OccurredOn: e.OccurredOn.Format(dateLayout),
		CreatedAt:  e.CreatedAt.UTC(),
	}
}

func writeEntryError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, guild.ErrNotMember):
		response.Error(c, http.StatusForbidden, "you are not a member of any guild")
	case errors.Is(err, rule.ErrNotFound):
		response.Error(c, http.StatusNotFound, "rule not found")
	case errors.Is(err, entry.ErrMemberNotInGuild):
		response.Error(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, entry.ErrWrongMember):
		response.Error(c, http.StatusForbidden, err.Error())
	case errors.Is(err, entry.ErrAlreadyLogged):
		response.Error(c, http.StatusConflict, err.Error())
	case errors.Is(err, entry.ErrNotFound):
		response.Error(c, http.StatusNotFound, "entry not found")
	case errors.Is(err, entry.ErrNotOwn):
		response.Error(c, http.StatusForbidden, err.Error())
	case errors.Is(err, entry.ErrNotRemovable):
		response.Error(c, http.StatusForbidden, err.Error())
	case errors.Is(err, entry.ErrNotToday):
		response.Error(c, http.StatusForbidden, err.Error())
	default:
		_ = c.Error(err)
		response.Error(c, http.StatusInternalServerError, "internal error")
	}
}
