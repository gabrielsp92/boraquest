package controllers

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/app/service"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/reminder"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/response"
)

// ReminderUseCase is the use-case the reminder controller depends on.
type ReminderUseCase interface {
	RunDaily(ctx context.Context, opts service.RunOptions) (service.RunResult, error)
}

// RunDailyRequest is the optional JSON body of POST /internal/reminders/daily.
type RunDailyRequest struct {
	Force  bool   `json:"force"`
	UserID string `json:"userId"`
}

// RunDailyResponse reports what one daily run did.
type RunDailyResponse struct {
	Day            string `json:"day"`
	UsersEligible  int    `json:"usersEligible"`
	UsersNotified  int    `json:"usersNotified"`
	UsersSkipped   int    `json:"usersSkipped"`
	DevicesSent    int    `json:"devicesSent"`
	DevicesFailed  int    `json:"devicesFailed"`
	DevicesRemoved int    `json:"devicesRemoved"`
}

// ReminderController serves the scheduler-only reminder endpoints.
type ReminderController struct {
	reminders ReminderUseCase
}

// NewReminderController builds a ReminderController.
func NewReminderController(reminders ReminderUseCase) *ReminderController {
	return &ReminderController{reminders: reminders}
}

// RunDaily handles POST /api/v1/internal/reminders/daily. An empty body means {}.
func (h *ReminderController) RunDaily(c *gin.Context) {
	var req RunDailyRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	// Finish the run even if the scheduler disconnects: a member claimed for today but
	// cut off mid-send would otherwise be skipped by the scheduler's retry.
	ctx := context.WithoutCancel(c.Request.Context())
	res, err := h.reminders.RunDaily(ctx, service.RunOptions{Force: req.Force, UserID: req.UserID})
	switch {
	case errors.Is(err, reminder.ErrOutsideWindow):
		response.Error(c, http.StatusConflict, "outside the daily send window")
		return
	case err != nil:
		writeInternalError(c, err)
		return
	}
	log.Printf("daily reminder: day=%s eligible=%d notified=%d skipped=%d sent=%d failed=%d removed=%d",
		res.Day.Format(dateLayout), res.UsersEligible, res.UsersNotified, res.UsersSkipped,
		res.DevicesSent, res.DevicesFailed, res.DevicesRemoved)
	response.JSON(c, http.StatusOK, RunDailyResponse{
		Day:            res.Day.Format(dateLayout),
		UsersEligible:  res.UsersEligible,
		UsersNotified:  res.UsersNotified,
		UsersSkipped:   res.UsersSkipped,
		DevicesSent:    res.DevicesSent,
		DevicesFailed:  res.DevicesFailed,
		DevicesRemoved: res.DevicesRemoved,
	})
}
