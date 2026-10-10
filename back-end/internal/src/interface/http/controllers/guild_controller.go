package controllers

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/app/service"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/guild"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/middleware"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/response"
)

// GuildUseCase is the use-case the guild controller depends on.
type GuildUseCase interface {
	ListMembers(ctx context.Context, callerID string) ([]service.GuildMember, error)
}

// GuildMemberResponse is the JSON representation of a guild member.
type GuildMemberResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// GuildMemberListResponse is the JSON body of GET /guild/members.
type GuildMemberListResponse struct {
	Members []GuildMemberResponse `json:"members"`
}

// GuildController serves the guild endpoints.
type GuildController struct {
	guilds GuildUseCase
}

// NewGuildController builds a GuildController.
func NewGuildController(g GuildUseCase) *GuildController {
	return &GuildController{guilds: g}
}

// ListMembers handles GET /api/v1/guild/members.
func (h *GuildController) ListMembers(c *gin.Context) {
	members, err := h.guilds.ListMembers(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		writeGuildError(c, err)
		return
	}
	body := GuildMemberListResponse{Members: make([]GuildMemberResponse, 0, len(members))}
	for _, m := range members {
		body.Members = append(body.Members, GuildMemberResponse{ID: m.ID, Name: m.Name})
	}
	response.JSON(c, http.StatusOK, body)
}

func writeGuildError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, guild.ErrNotMember):
		response.Error(c, http.StatusForbidden, "you are not a member of any guild")
	default:
		_ = c.Error(err)
		response.Error(c, http.StatusInternalServerError, "internal error")
	}
}
