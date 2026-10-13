package integration_test

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/push"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/reminder"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/controllers"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/response"
)

var (
	testP256dh = base64.RawURLEncoding.EncodeToString([]byte(strings.Repeat("p", 65)))
	testAuth   = base64.RawURLEncoding.EncodeToString([]byte(strings.Repeat("a", 16)))
)

func subscription(endpoint string) controllers.PushSubscriptionRequest {
	return controllers.PushSubscriptionRequest{Endpoint: endpoint, Keys: controllers.PushKeys{P256dh: testP256dh, Auth: testAuth}}
}

func subscribe(t *testing.T, baseURL, token, endpoint string) {
	t.Helper()
	res := call(t, http.MethodPut, baseURL+"/api/v1/me/push-subscription", token, subscription(endpoint))
	require.Equal(t, http.StatusNoContent, res.StatusCode)
}

// runDaily calls the cron endpoint with the test secret and returns the response.
func runDaily(t *testing.T, baseURL string, body any) *http.Response {
	t.Helper()
	return call(t, http.MethodPost, baseURL+"/api/v1/internal/reminders/daily", testCronSecret, body)
}

func runDailyOK(t *testing.T, baseURL string, body any) controllers.RunDailyResponse {
	t.Helper()
	res := runDaily(t, baseURL, body)
	require.Equal(t, http.StatusOK, res.StatusCode)
	return decodeBody[controllers.RunDailyResponse](t, res)
}

// subscriptionOwners maps every stored endpoint to its user id.
func subscriptionOwners(t *testing.T) map[string]string {
	t.Helper()
	rows, err := pool.Query(context.Background(), `SELECT endpoint, user_id FROM push_subscriptions`)
	require.NoError(t, err)
	defer rows.Close()
	owners := map[string]string{}
	for rows.Next() {
		var endpoint, userID string
		require.NoError(t, rows.Scan(&endpoint, &userID))
		owners[endpoint] = userID
	}
	require.NoError(t, rows.Err())
	return owners
}

func resetReminders(t *testing.T) {
	t.Helper()
	resetRules(t)
	resetEntries(t)
	resetPush(t)
}

func TestDailyReminderRun(t *testing.T) {
	resetReminders(t)
	srv, sender := newReminderServer(t, todayAt(20, 30))
	liaToken := login(t, srv.URL, "lia@boraquest.dev")
	betoToken := login(t, srv.URL, "beto@boraquest.dev")
	r := createRule(t, srv.URL, liaToken, dailySumRuleReq)
	subscribe(t, srv.URL, liaToken, "https://push.test/lia")
	today := todayAt(0, 0).Format("2006-01-02")

	// Lia has an unchecked daily quest: exactly one push to her device.
	got := runDailyOK(t, srv.URL, nil)
	assert.Equal(t, controllers.RunDailyResponse{Day: today, UsersEligible: 1, UsersNotified: 1, DevicesSent: 1}, got)
	assert.Equal(t, []string{"https://push.test/lia"}, sender.sentTo())
	assert.Equal(t, reminder.Daily(), sender.last)

	// Same day again: claimed already, nothing is sent.
	got = runDailyOK(t, srv.URL, map[string]any{})
	assert.Equal(t, controllers.RunDailyResponse{Day: today, UsersEligible: 1, UsersSkipped: 1}, got)
	assert.Len(t, sender.sentTo(), 1)

	// force + userId re-sends to Lia only, even though Beto is eligible too.
	subscribe(t, srv.URL, betoToken, "https://push.test/beto")
	got = runDailyOK(t, srv.URL, map[string]any{"force": true, "userId": "lia"})
	assert.Equal(t, controllers.RunDailyResponse{Day: today, UsersEligible: 1, UsersNotified: 1, DevicesSent: 1}, got)
	assert.Equal(t, []string{"https://push.test/lia", "https://push.test/lia"}, sender.sentTo())

	// Once Lia checks every daily quest, she gets nothing.
	res := call(t, http.MethodPost, srv.URL+"/api/v1/entries", liaToken, controllers.EntryRequest{RuleID: r.ID, MemberID: "lia"})
	require.Equal(t, http.StatusCreated, res.StatusCode)
	got = runDailyOK(t, srv.URL, map[string]any{"force": true, "userId": "lia"})
	assert.Equal(t, controllers.RunDailyResponse{Day: today}, got)
	assert.Len(t, sender.sentTo(), 2)
}

func TestDailyReminderIgnoresWeeklyAndSlipRules(t *testing.T) {
	resetReminders(t)
	srv, sender := newReminderServer(t, todayAt(20, 30))
	liaToken := login(t, srv.URL, "lia@boraquest.dev")
	createRule(t, srv.URL, liaToken, weeklySumRuleReq)
	createRule(t, srv.URL, liaToken, decreaseRuleReq)
	subscribe(t, srv.URL, liaToken, "https://push.test/lia")

	got := runDailyOK(t, srv.URL, nil)

	assert.Zero(t, got.UsersEligible)
	assert.Empty(t, sender.sentTo())
}

func TestDailyReminderDeviceFailures(t *testing.T) {
	resetReminders(t)
	srv, sender := newReminderServer(t, todayAt(22, 59))
	liaToken := login(t, srv.URL, "lia@boraquest.dev")
	createRule(t, srv.URL, liaToken, dailySumRuleReq)
	for _, d := range []string{"gone", "broken", "ok"} {
		subscribe(t, srv.URL, liaToken, "https://push.test/"+d)
	}
	sender.errs["https://push.test/gone"] = push.ErrGone
	sender.errs["https://push.test/broken"] = errors.New("push: status 500")

	got := runDailyOK(t, srv.URL, nil)

	assert.Equal(t, 1, got.DevicesSent)
	assert.Equal(t, 1, got.DevicesFailed)
	assert.Equal(t, 1, got.DevicesRemoved)
	assert.ElementsMatch(t, []string{"https://push.test/gone", "https://push.test/broken", "https://push.test/ok"}, sender.sentTo())
	assert.Equal(t, map[string]string{"https://push.test/broken": "lia", "https://push.test/ok": "lia"}, subscriptionOwners(t),
		"a 404/410 deletes the row; any other failure keeps it")
}

func TestDailyReminderAuthAndWindow(t *testing.T) {
	resetReminders(t)
	srv, sender := newReminderServer(t, todayAt(20, 30))
	liaToken := login(t, srv.URL, "lia@boraquest.dev")
	createRule(t, srv.URL, liaToken, dailySumRuleReq)
	subscribe(t, srv.URL, liaToken, "https://push.test/lia")
	url := srv.URL + "/api/v1/internal/reminders/daily"

	for _, token := range []string{"", "wrong", liaToken} {
		res := call(t, http.MethodPost, url, token, nil)
		require.Equal(t, http.StatusUnauthorized, res.StatusCode)
		assert.Equal(t, "unauthorized", decodeBody[response.ErrorBody](t, res).Error)
	}

	assert.Empty(t, sender.sentTo(), "nothing is sent without the secret")

	// Malformed JSON is rejected before anything runs.
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader("{nope"))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+testCronSecret)
	bad, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer bad.Body.Close()
	assert.Equal(t, http.StatusBadRequest, bad.StatusCode)
	assert.Empty(t, sender.sentTo())

	runDailyOK(t, srv.URL, nil)
	assert.Len(t, sender.sentTo(), 1)

	for _, at := range []time.Time{todayAt(19, 59), todayAt(23, 0)} {
		early, earlySender := newReminderServer(t, at)
		res := runDaily(t, early.URL, nil)
		require.Equal(t, http.StatusConflict, res.StatusCode, at)
		assert.Equal(t, "outside the daily send window", decodeBody[response.ErrorBody](t, res).Error)
		assert.Empty(t, earlySender.sentTo())

		// force skips the window (the claim from 20:30 still holds today, so use userId).
		got := runDailyOK(t, early.URL, map[string]any{"force": true, "userId": "lia"})
		assert.Equal(t, 1, got.UsersNotified)
	}
}

func TestDailyReminderUnknownUser(t *testing.T) {
	resetReminders(t)
	srv, sender := newReminderServer(t, todayAt(20, 30))
	liaToken := login(t, srv.URL, "lia@boraquest.dev")
	createRule(t, srv.URL, liaToken, dailySumRuleReq)
	subscribe(t, srv.URL, liaToken, "https://push.test/lia")

	for _, id := range []string{outsiderID, "ghost"} {
		got := runDailyOK(t, srv.URL, map[string]any{"force": true, "userId": id})
		assert.Equal(t, controllers.RunDailyResponse{Day: todayAt(0, 0).Format("2006-01-02")}, got)
	}
	assert.Empty(t, sender.sentTo())
}

func TestPushPublicKey(t *testing.T) {
	srv := newServer(t)

	res := call(t, http.MethodGet, srv.URL+"/api/v1/push/public-key", "", nil)

	require.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, controllers.PublicKeyResponse{PublicKey: testVAPIDPublicKey}, decodeBody[controllers.PublicKeyResponse](t, res))
}

func TestPushSubscriptionSaveAndReassign(t *testing.T) {
	resetPush(t)
	srv := newServer(t)
	liaToken := login(t, srv.URL, "lia@boraquest.dev")
	betoToken := login(t, srv.URL, "beto@boraquest.dev")
	url := srv.URL + "/api/v1/me/push-subscription"

	res := call(t, http.MethodPut, url, "", subscription("https://push.test/shared"))
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode)

	subscribe(t, srv.URL, liaToken, "https://push.test/shared")
	assert.Equal(t, map[string]string{"https://push.test/shared": "lia"}, subscriptionOwners(t))

	// A shared tablet changes hands: the same endpoint now belongs to Beto.
	subscribe(t, srv.URL, betoToken, "https://push.test/shared")
	assert.Equal(t, map[string]string{"https://push.test/shared": "beto"}, subscriptionOwners(t))

	// A user in no guild may subscribe; they are just never reminded.
	subscribe(t, srv.URL, login(t, srv.URL, "intruso@boraquest.dev"), "https://push.test/outsider")

	invalid := []controllers.PushSubscriptionRequest{
		subscription("http://push.test/plain"),
		{Endpoint: "https://push.test/x", Keys: controllers.PushKeys{P256dh: testAuth, Auth: testAuth}},
		{Endpoint: "https://push.test/x", Keys: controllers.PushKeys{P256dh: testP256dh, Auth: testP256dh}},
	}
	for _, body := range invalid {
		res := call(t, http.MethodPut, url, liaToken, body)
		require.Equal(t, http.StatusBadRequest, res.StatusCode, body)
		assert.Equal(t, "invalid subscription", decodeBody[response.ErrorBody](t, res).Error)
	}
}

func TestPushSubscriptionKeepsTenPerUser(t *testing.T) {
	resetPush(t)
	srv := newServer(t)
	liaToken := login(t, srv.URL, "lia@boraquest.dev")
	for i := 1; i <= push.MaxPerUser; i++ {
		subscribe(t, srv.URL, liaToken, fmt.Sprintf("https://push.test/%02d", i))
	}
	// Refresh the oldest one: it becomes the most recently updated.
	subscribe(t, srv.URL, liaToken, "https://push.test/01")
	require.Len(t, subscriptionOwners(t), push.MaxPerUser, "re-saving never evicts")

	subscribe(t, srv.URL, liaToken, "https://push.test/11")

	owners := subscriptionOwners(t)
	assert.Len(t, owners, push.MaxPerUser)
	assert.NotContains(t, owners, "https://push.test/02", "the least recently updated is evicted")
	assert.Contains(t, owners, "https://push.test/01")
	assert.Contains(t, owners, "https://push.test/11")
}

func TestPushSubscriptionDelete(t *testing.T) {
	resetPush(t)
	srv := newServer(t)
	liaToken := login(t, srv.URL, "lia@boraquest.dev")
	betoToken := login(t, srv.URL, "beto@boraquest.dev")
	url := srv.URL + "/api/v1/me/push-subscription"
	subscribe(t, srv.URL, betoToken, "https://push.test/beto")
	subscribe(t, srv.URL, liaToken, "https://push.test/lia")

	// Lia can't remove Beto's device.
	res := call(t, http.MethodDelete, url, liaToken, map[string]string{"endpoint": "https://push.test/beto"})
	require.Equal(t, http.StatusNoContent, res.StatusCode)
	assert.Equal(t, "beto", subscriptionOwners(t)["https://push.test/beto"])

	res = call(t, http.MethodDelete, url, liaToken, map[string]string{"endpoint": "https://push.test/lia"})
	require.Equal(t, http.StatusNoContent, res.StatusCode)
	assert.Equal(t, map[string]string{"https://push.test/beto": "beto"}, subscriptionOwners(t))

	res = call(t, http.MethodDelete, url, liaToken, map[string]string{"endpoint": ""})
	require.Equal(t, http.StatusBadRequest, res.StatusCode)
	assert.Equal(t, "endpoint is required", decodeBody[response.ErrorBody](t, res).Error)

	res = call(t, http.MethodDelete, url, "", map[string]string{"endpoint": "https://push.test/beto"})
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
}
