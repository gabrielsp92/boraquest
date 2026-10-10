package unit_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/app/service"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/guild"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/controllers"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/middleware"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/response"
)

// fakeGuildUseCase returns members/err for every call and records who called it.
type fakeGuildUseCase struct {
	members []service.GuildMember
	err     error
	userID  string
}

func (f *fakeGuildUseCase) ListMembers(_ context.Context, callerID string) ([]service.GuildMember, error) {
	f.userID = callerID
	return f.members, f.err
}

func serveGuild(t *testing.T, uc controllers.GuildUseCase, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	h := controllers.NewGuildController(uc)
	r := gin.New()
	g := r.Group("/guild", middleware.RequireAuth(fakeVerifier{userID: "lia"}))
	g.GET("/members", h.ListMembers)

	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Authorization", "Bearer good")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestGuildControllerListMembers(t *testing.T) {
	uc := &fakeGuildUseCase{members: []service.GuildMember{{ID: "lia", Name: "Lia"}, {ID: "beto", Name: "Beto"}}}

	rec := serveGuild(t, uc, http.MethodGet, "/guild/members")

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, controllers.GuildMemberListResponse{
		Members: []controllers.GuildMemberResponse{{ID: "lia", Name: "Lia"}, {ID: "beto", Name: "Beto"}},
	}, decode[controllers.GuildMemberListResponse](t, rec))
	assert.Equal(t, "lia", uc.userID)
}

func TestGuildControllerListMembersEmptyIsArray(t *testing.T) {
	rec := serveGuild(t, &fakeGuildUseCase{}, http.MethodGet, "/guild/members")

	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"members":[]}`, rec.Body.String())
}

func TestGuildControllerErrorMapping(t *testing.T) {
	cases := []struct {
		err     error
		status  int
		message string
	}{
		{guild.ErrNotMember, http.StatusForbidden, "you are not a member of any guild"},
		{errBoom, http.StatusInternalServerError, "internal error"},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%v", tc.err), func(t *testing.T) {
			rec := serveGuild(t, &fakeGuildUseCase{err: tc.err}, http.MethodGet, "/guild/members")

			require.Equal(t, tc.status, rec.Code)
			assert.Equal(t, tc.message, decode[response.ErrorBody](t, rec).Error)
		})
	}
}
