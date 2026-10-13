package unit_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/app/service"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/reminder"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/controllers"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/response"
)

type fakeReminderUseCase struct {
	res  service.RunResult
	err  error
	opts *service.RunOptions
}

func (f *fakeReminderUseCase) RunDaily(_ context.Context, opts service.RunOptions) (service.RunResult, error) {
	f.opts = &opts
	return f.res, f.err
}

func serveReminder(uc controllers.ReminderUseCase, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/internal/reminders/daily", controllers.NewReminderController(uc).RunDaily)
	req := httptest.NewRequest(http.MethodPost, "/internal/reminders/daily", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestReminderControllerRunDaily(t *testing.T) {
	uc := &fakeReminderUseCase{res: service.RunResult{
		Day: reminderDay, UsersEligible: 3, UsersNotified: 2, UsersSkipped: 1, DevicesSent: 3, DevicesFailed: 1, DevicesRemoved: 1,
	}}

	rec := serveReminder(uc, `{"force":true,"userId":"lia"}`)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, controllers.RunDailyResponse{
		Day: "2026-10-13", UsersEligible: 3, UsersNotified: 2, UsersSkipped: 1, DevicesSent: 3, DevicesFailed: 1, DevicesRemoved: 1,
	}, decode[controllers.RunDailyResponse](t, rec))
	assert.Equal(t, &service.RunOptions{Force: true, UserID: "lia"}, uc.opts)
}

func TestReminderControllerEmptyBodyMeansDefaults(t *testing.T) {
	uc := &fakeReminderUseCase{res: service.RunResult{Day: reminderDay}}

	rec := serveReminder(uc, "")

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, &service.RunOptions{}, uc.opts)
}

func TestReminderControllerErrors(t *testing.T) {
	cases := map[string]struct {
		body    string
		err     error
		status  int
		message string
	}{
		"malformed json": {`{nope`, nil, http.StatusBadRequest, "invalid request body"},
		"outside window": {`{}`, reminder.ErrOutsideWindow, http.StatusConflict, "outside the daily send window"},
		"unexpected":     {`{}`, errBoom, http.StatusInternalServerError, "internal error"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			rec := serveReminder(&fakeReminderUseCase{err: tc.err}, tc.body)

			require.Equal(t, tc.status, rec.Code)
			assert.Equal(t, tc.message, decode[response.ErrorBody](t, rec).Error)
		})
	}
}
