package controllers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/app/service"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/guild"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/rule"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/middleware"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/response"
)

// RuleUseCase is the use-case the rule controller depends on.
type RuleUseCase interface {
	List(ctx context.Context, userID string) ([]rule.Rule, error)
	Get(ctx context.Context, userID, id string) (rule.Rule, error)
	Create(ctx context.Context, userID string, in service.RuleInput) (rule.Rule, error)
	Update(ctx context.Context, userID, id string, in service.RuleInput) (rule.Rule, error)
	Delete(ctx context.Context, userID, id string) error
}

// RuleRequest is the JSON body of POST /rules and PUT /rules/:id.
type RuleRequest struct {
	Name      string `json:"name"`
	Frequency string `json:"frequency"`
	ScoreType string `json:"scoreType"`
	Score     int    `json:"score"`
}

// RuleResponse is the JSON representation of a rule.
type RuleResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Frequency string    `json:"frequency"`
	ScoreType string    `json:"scoreType"`
	Score     int       `json:"score"`
	CreatedBy string    `json:"createdBy"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// RuleListResponse is the JSON body of GET /rules.
type RuleListResponse struct {
	Rules []RuleResponse `json:"rules"`
	Limit int            `json:"limit"`
}

// RuleController serves the rules endpoints.
type RuleController struct {
	rules RuleUseCase
}

// NewRuleController builds a RuleController.
func NewRuleController(rules RuleUseCase) *RuleController {
	return &RuleController{rules: rules}
}

// List handles GET /api/v1/rules.
func (h *RuleController) List(c *gin.Context) {
	rules, err := h.rules.List(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		writeRuleError(c, err)
		return
	}
	body := RuleListResponse{Rules: make([]RuleResponse, 0, len(rules)), Limit: rule.MaxPerGuild}
	for _, r := range rules {
		body.Rules = append(body.Rules, toRuleResponse(r))
	}
	response.JSON(c, http.StatusOK, body)
}

// Get handles GET /api/v1/rules/:id.
func (h *RuleController) Get(c *gin.Context) {
	r, err := h.rules.Get(c.Request.Context(), middleware.UserID(c), c.Param("id"))
	if err != nil {
		writeRuleError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, toRuleResponse(r))
}

// Create handles POST /api/v1/rules.
func (h *RuleController) Create(c *gin.Context) {
	in, ok := bindRuleInput(c)
	if !ok {
		return
	}
	r, err := h.rules.Create(c.Request.Context(), middleware.UserID(c), in)
	if err != nil {
		writeRuleError(c, err)
		return
	}
	response.JSON(c, http.StatusCreated, toRuleResponse(r))
}

// Update handles PUT /api/v1/rules/:id.
func (h *RuleController) Update(c *gin.Context) {
	in, ok := bindRuleInput(c)
	if !ok {
		return
	}
	r, err := h.rules.Update(c.Request.Context(), middleware.UserID(c), c.Param("id"), in)
	if err != nil {
		writeRuleError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, toRuleResponse(r))
}

// Delete handles DELETE /api/v1/rules/:id.
func (h *RuleController) Delete(c *gin.Context) {
	if err := h.rules.Delete(c.Request.Context(), middleware.UserID(c), c.Param("id")); err != nil {
		writeRuleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func bindRuleInput(c *gin.Context) (service.RuleInput, bool) {
	var req RuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return service.RuleInput{}, false
	}
	return service.RuleInput{
		Name:      req.Name,
		Frequency: rule.Frequency(req.Frequency),
		ScoreType: rule.ScoreType(req.ScoreType),
		Score:     req.Score,
	}, true
}

func toRuleResponse(r rule.Rule) RuleResponse {
	return RuleResponse{
		ID:        r.ID,
		Name:      r.Name,
		Frequency: string(r.Frequency),
		ScoreType: string(r.ScoreType),
		Score:     r.Score,
		CreatedBy: r.CreatedBy,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

func writeRuleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, rule.ErrInvalidName), errors.Is(err, rule.ErrInvalidFrequency),
		errors.Is(err, rule.ErrInvalidScoreType), errors.Is(err, rule.ErrInvalidScore):
		response.Error(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, guild.ErrNotMember):
		response.Error(c, http.StatusForbidden, "you are not a member of any guild")
	case errors.Is(err, rule.ErrNotFound):
		response.Error(c, http.StatusNotFound, "rule not found")
	case errors.Is(err, rule.ErrLimitReached):
		response.Error(c, http.StatusConflict, fmt.Sprintf("a guild can have at most %d rules", rule.MaxPerGuild))
	default:
		_ = c.Error(err)
		response.Error(c, http.StatusInternalServerError, "internal error")
	}
}
