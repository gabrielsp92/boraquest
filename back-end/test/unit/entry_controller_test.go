package unit_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/app/service"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/entry"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/guild"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/rule"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/controllers"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/middleware"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/response"
)

// fakeEntryUseCase returns entry/err for every call and records what it received.
type fakeEntryUseCase struct {
	entries         []entry.Entry
	entry           entry.Entry
	from, to, today time.Time
	err             error
	userID          string
	id              string
	in              service.EntryInput
	period          service.EntryPeriod
}

func (f *fakeEntryUseCase) Create(_ context.Context, userID string, in service.EntryInput) (entry.Entry, error) {
	f.userID, f.in = userID, in
	return f.entry, f.err
}

func (f *fakeEntryUseCase) Delete(_ context.Context, userID, id string) error {
	f.userID, f.id = userID, id
	return f.err
}

func (f *fakeEntryUseCase) List(_ context.Context, userID string, period service.EntryPeriod) ([]entry.Entry, time.Time, time.Time, time.Time, error) {
	f.userID, f.period = userID, period
	return f.entries, f.from, f.to, f.today, f.err
}

var entryAt = time.Date(2026, 10, 8, 14, 3, 0, 0, time.UTC)

var sampleEntry = entry.Entry{
	ID: "e1", GuildID: "g1", RuleID: "r1", RuleName: "Beber 2 L de água", ScoreType: rule.ScoreTypeSum,
	Points: 10, MemberID: "lia", LoggedBy: "lia",
	OccurredOn: time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC), CreatedAt: entryAt,
}

var sampleEntryResponse = controllers.EntryResponse{
	ID: "e1", RuleID: "r1", RuleName: "Beber 2 L de água", ScoreType: "sum", Points: 10,
	MemberID: "lia", LoggedBy: "lia", OccurredOn: "2026-10-08", CreatedAt: entryAt,
}

func serveEntries(t *testing.T, uc controllers.EntryUseCase, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	h := controllers.NewEntryController(uc)
	r := gin.New()
	g := r.Group("/entries", middleware.RequireAuth(fakeVerifier{userID: "lia"}))
	g.GET("", h.List)
	g.POST("", h.Create)
	g.DELETE("/:id", h.Delete)

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer good")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

const validEntryBody = `{"ruleId":"r1","memberId":"lia"}`

func TestEntryControllerList(t *testing.T) {
	uc := &fakeEntryUseCase{
		entries: []entry.Entry{sampleEntry},
		from:    time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC),
		to:      time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC),
		today:   time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC),
	}

	rec := serveEntries(t, uc, http.MethodGet, "/entries?period=today", "")

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, controllers.EntryListResponse{
		Entries: []controllers.EntryResponse{sampleEntryResponse},
		From:    "2026-10-08", To: "2026-10-08", Today: "2026-10-08",
	}, decode[controllers.EntryListResponse](t, rec))
	assert.Equal(t, "lia", uc.userID)
	assert.Equal(t, service.PeriodToday, uc.period)
}

func TestEntryControllerListWeek(t *testing.T) {
	uc := &fakeEntryUseCase{}

	rec := serveEntries(t, uc, http.MethodGet, "/entries?period=week", "")

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, service.PeriodWeek, uc.period)
}

func TestEntryControllerListEmptyIsArray(t *testing.T) {
	rec := serveEntries(t, &fakeEntryUseCase{}, http.MethodGet, "/entries?period=today", "")

	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"entries":[],"from":"0001-01-01","to":"0001-01-01","today":"0001-01-01"}`, rec.Body.String())
}

func TestEntryControllerListMissingOrInvalidPeriod(t *testing.T) {
	for _, path := range []string{"/entries", "/entries?period=", "/entries?period=month"} {
		t.Run(path, func(t *testing.T) {
			uc := &fakeEntryUseCase{}
			rec := serveEntries(t, uc, http.MethodGet, path, "")

			require.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Empty(t, uc.userID, "use-case must not be called")
		})
	}
}

func TestEntryControllerCreate(t *testing.T) {
	uc := &fakeEntryUseCase{entry: sampleEntry}

	rec := serveEntries(t, uc, http.MethodPost, "/entries", validEntryBody)

	require.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, sampleEntryResponse, decode[controllers.EntryResponse](t, rec))
	assert.Equal(t, service.EntryInput{RuleID: "r1", MemberID: "lia"}, uc.in)
	assert.Equal(t, "lia", uc.userID)
}

func TestEntryControllerCreateInvalidBody(t *testing.T) {
	for name, body := range map[string]string{
		"not json":         `not json`,
		"missing ruleId":   `{"memberId":"lia"}`,
		"missing memberId": `{"ruleId":"r1"}`,
		"empty ruleId":     `{"ruleId":"","memberId":"lia"}`,
	} {
		t.Run(name, func(t *testing.T) {
			uc := &fakeEntryUseCase{}
			rec := serveEntries(t, uc, http.MethodPost, "/entries", body)

			require.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Empty(t, uc.userID, "use-case must not be called")
		})
	}
}

func TestEntryControllerDelete(t *testing.T) {
	uc := &fakeEntryUseCase{}

	rec := serveEntries(t, uc, http.MethodDelete, "/entries/e1", "")

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Empty(t, rec.Body.String())
	assert.Equal(t, "e1", uc.id)
	assert.Equal(t, "lia", uc.userID)
}

func TestEntryControllerErrorMapping(t *testing.T) {
	cases := []struct {
		err     error
		status  int
		message string
	}{
		{guild.ErrNotMember, http.StatusForbidden, "you are not a member of any guild"},
		{rule.ErrNotFound, http.StatusNotFound, "rule not found"},
		{entry.ErrMemberNotInGuild, http.StatusBadRequest, entry.ErrMemberNotInGuild.Error()},
		{entry.ErrWrongMember, http.StatusForbidden, entry.ErrWrongMember.Error()},
		{entry.ErrAlreadyLogged, http.StatusConflict, entry.ErrAlreadyLogged.Error()},
		{entry.ErrNotFound, http.StatusNotFound, "entry not found"},
		{entry.ErrNotOwn, http.StatusForbidden, entry.ErrNotOwn.Error()},
		{entry.ErrNotRemovable, http.StatusForbidden, entry.ErrNotRemovable.Error()},
		{entry.ErrNotToday, http.StatusForbidden, entry.ErrNotToday.Error()},
		{errBoom, http.StatusInternalServerError, "internal error"},
	}
	calls := [][3]string{
		{http.MethodGet, "/entries?period=today", ""},
		{http.MethodPost, "/entries", validEntryBody},
		{http.MethodDelete, "/entries/e1", ""},
	}
	for _, tc := range cases {
		for _, call := range calls {
			t.Run(fmt.Sprintf("%s %s %v", call[0], call[1], tc.err), func(t *testing.T) {
				rec := serveEntries(t, &fakeEntryUseCase{err: tc.err}, call[0], call[1], call[2])

				require.Equal(t, tc.status, rec.Code)
				assert.Equal(t, tc.message, decode[response.ErrorBody](t, rec).Error)
			})
		}
	}
}
