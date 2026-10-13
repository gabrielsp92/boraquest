package unit_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/app/service"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/entry"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/guild"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/push"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/reminder"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/rule"
)

type fakeGuildLister struct {
	guilds []guild.Guild
	err    error
	called bool
}

func (f *fakeGuildLister) ListAll(context.Context) ([]guild.Guild, error) {
	f.called = true
	return f.guilds, f.err
}

type fakeReminderRules struct {
	byGuild map[string][]rule.Rule
	err     error
}

func (f fakeReminderRules) ListByGuild(_ context.Context, guildID string) ([]rule.Rule, error) {
	return f.byGuild[guildID], f.err
}

type fakeReminderEntries struct {
	byMember map[string][]entry.Entry
	err      error
	from, to []time.Time
	members  []string
}

func (f *fakeReminderEntries) ListByMember(_ context.Context, _, memberID string, from, to time.Time) ([]entry.Entry, error) {
	f.members = append(f.members, memberID)
	f.from, f.to = append(f.from, from), append(f.to, to)
	return f.byMember[memberID], f.err
}

type claimCall struct {
	userID   string
	day, now time.Time
}

type fakeReminderLog struct {
	refuse map[string]bool // userID -> already claimed
	err    error
	calls  []claimCall
}

func (f *fakeReminderLog) Claim(_ context.Context, userID string, day, now time.Time) (bool, error) {
	f.calls = append(f.calls, claimCall{userID, day, now})
	return !f.refuse[userID], f.err
}

type sentPush struct {
	endpoint string
	payload  reminder.Payload
}

type fakePushSender struct {
	errs map[string]error // endpoint -> error to return
	sent []sentPush
}

func (f *fakePushSender) Send(_ context.Context, s push.Subscription, p reminder.Payload) error {
	f.sent = append(f.sent, sentPush{s.Endpoint, p})
	return f.errs[s.Endpoint]
}

// reminderFixture is one guild (lia, beto) with a daily sum quest, a weekly quest and a
// daily slip. Lia has nothing checked and two devices; Beto checked the daily quest.
type reminderFixture struct {
	guilds  *fakeGuildLister
	rules   fakeReminderRules
	entries *fakeReminderEntries
	subs    *fakePushSubs
	log     *fakeReminderLog
	sender  *fakePushSender
	now     time.Time
}

// reminderNow is 20:30 on 2026-10-13 in São Paulo.
var reminderNow = time.Date(2026, 10, 13, 23, 30, 0, 0, time.UTC)

func newReminderFixture() *reminderFixture {
	return &reminderFixture{
		guilds: &fakeGuildLister{guilds: []guild.Guild{{ID: "g1", UserIDs: []string{"beto", "lia"}}}},
		rules: fakeReminderRules{byGuild: map[string][]rule.Rule{"g1": {
			{ID: "daily", Frequency: rule.FrequencyDaily, ScoreType: rule.ScoreTypeSum},
			{ID: "weekly", Frequency: rule.FrequencyWeekly, ScoreType: rule.ScoreTypeSum},
			{ID: "slip", Frequency: rule.FrequencyDaily, ScoreType: rule.ScoreTypeDecrease},
		}}},
		entries: &fakeReminderEntries{byMember: map[string][]entry.Entry{
			"beto": {{RuleID: "daily", ScoreType: rule.ScoreTypeSum}},
			"lia":  {{RuleID: "daily", ScoreType: rule.ScoreTypeDecrease}}, // a slip never counts as done
		}},
		subs: &fakePushSubs{byUser: map[string][]push.Subscription{
			"lia":  {{UserID: "lia", Endpoint: "https://push.example.com/lia-phone"}, {UserID: "lia", Endpoint: "https://push.example.com/lia-tablet"}},
			"beto": {{UserID: "beto", Endpoint: "https://push.example.com/beto"}},
		}},
		log:    &fakeReminderLog{},
		sender: &fakePushSender{},
		now:    reminderNow,
	}
}

func (f *reminderFixture) run(opts service.RunOptions) (service.RunResult, error) {
	svc := service.NewReminderService(f.guilds, f.rules, f.entries, f.subs, f.log, f.sender, fakeClock{now: f.now}, saoPaulo)
	return svc.RunDaily(context.Background(), opts)
}

func (f *reminderFixture) sentTo() []string {
	var out []string
	for _, s := range f.sender.sent {
		out = append(out, s.endpoint)
	}
	return out
}

var reminderDay = time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)

func TestReminderServiceRemindsMembersWithUncheckedDailyQuests(t *testing.T) {
	f := newReminderFixture()

	res, err := f.run(service.RunOptions{})

	require.NoError(t, err)
	assert.Equal(t, service.RunResult{Day: reminderDay, UsersEligible: 1, UsersNotified: 1, DevicesSent: 2}, res)
	assert.Equal(t, []string{"https://push.example.com/lia-phone", "https://push.example.com/lia-tablet"}, f.sentTo())
	assert.Equal(t, reminder.Daily(), f.sender.sent[0].payload)
	assert.Equal(t, []claimCall{{"lia", reminderDay, reminderNow}}, f.log.calls)
	assert.Equal(t, []time.Time{reminderDay, reminderDay}, f.entries.from, "today only")
	assert.Equal(t, []time.Time{reminderDay, reminderDay}, f.entries.to)
}

func TestReminderServiceOutsideWindow(t *testing.T) {
	for name, local := range map[string]time.Time{
		"19:59": time.Date(2026, 10, 13, 19, 59, 0, 0, saoPaulo),
		"23:00": time.Date(2026, 10, 13, 23, 0, 0, 0, saoPaulo),
	} {
		t.Run(name, func(t *testing.T) {
			f := newReminderFixture()
			f.now = local

			_, err := f.run(service.RunOptions{})

			assert.ErrorIs(t, err, reminder.ErrOutsideWindow)
			assert.False(t, f.guilds.called)
			assert.Empty(t, f.sender.sent)
		})
	}
}

func TestReminderServiceForceSkipsWindow(t *testing.T) {
	f := newReminderFixture()
	f.now = time.Date(2026, 10, 13, 12, 0, 0, 0, saoPaulo)

	res, err := f.run(service.RunOptions{Force: true})

	require.NoError(t, err)
	assert.Equal(t, 1, res.UsersNotified)
	assert.Len(t, f.log.calls, 1, "force alone still claims the day")
}

func TestReminderServiceDayIsComputedInSaoPaulo(t *testing.T) {
	f := newReminderFixture()
	f.now = time.Date(2026, 10, 14, 2, 30, 0, 0, time.UTC) // 23:30 on the 13th in São Paulo

	_, err := f.run(service.RunOptions{})
	assert.ErrorIs(t, err, reminder.ErrOutsideWindow)

	res, err := f.run(service.RunOptions{Force: true})
	require.NoError(t, err)
	assert.Equal(t, reminderDay, res.Day)
}

func TestReminderServiceIgnoresGuildWithoutDailySumQuests(t *testing.T) {
	f := newReminderFixture()
	f.rules.byGuild["g1"] = f.rules.byGuild["g1"][1:] // only weekly and slip

	res, err := f.run(service.RunOptions{})

	require.NoError(t, err)
	assert.Equal(t, service.RunResult{Day: reminderDay}, res)
	assert.Empty(t, f.entries.members, "entries are never read")
	assert.Empty(t, f.sender.sent)
}

func TestReminderServiceSkipsMemberWithoutDevices(t *testing.T) {
	f := newReminderFixture()
	delete(f.subs.byUser, "lia")

	res, err := f.run(service.RunOptions{})

	require.NoError(t, err)
	assert.Equal(t, service.RunResult{Day: reminderDay}, res)
	assert.Empty(t, f.log.calls)
}

func TestReminderServiceSkipsAlreadyClaimed(t *testing.T) {
	f := newReminderFixture()
	f.log.refuse = map[string]bool{"lia": true}

	res, err := f.run(service.RunOptions{})

	require.NoError(t, err)
	assert.Equal(t, service.RunResult{Day: reminderDay, UsersEligible: 1, UsersSkipped: 1}, res)
	assert.Empty(t, f.sender.sent)
}

func TestReminderServiceUserFilterSkipsClaim(t *testing.T) {
	f := newReminderFixture()
	f.log.refuse = map[string]bool{"lia": true} // already reminded today
	f.entries.byMember["beto"] = nil            // beto would be eligible too

	res, err := f.run(service.RunOptions{Force: true, UserID: "lia"})

	require.NoError(t, err)
	assert.Equal(t, service.RunResult{Day: reminderDay, UsersEligible: 1, UsersNotified: 1, DevicesSent: 2}, res)
	assert.Empty(t, f.log.calls, "no claim is written")
	assert.Equal(t, []string{"lia"}, f.entries.members, "nobody else is considered")
}

func TestReminderServiceHandlesSendFailures(t *testing.T) {
	f := newReminderFixture()
	f.sender.errs = map[string]error{
		"https://push.example.com/lia-phone":  push.ErrGone,
		"https://push.example.com/lia-tablet": errors.New("push: status 500"),
	}
	f.entries.byMember["beto"] = nil

	res, err := f.run(service.RunOptions{})

	require.NoError(t, err)
	assert.Equal(t, service.RunResult{Day: reminderDay, UsersEligible: 2, UsersNotified: 2, DevicesSent: 1, DevicesFailed: 1, DevicesRemoved: 1}, res)
	assert.Equal(t, [2]string{"lia", "https://push.example.com/lia-phone"}, f.subs.deletedFor, "only the member's own row")
	assert.Contains(t, f.sentTo(), "https://push.example.com/beto", "the run continues after a failure")
}

func TestReminderServiceRepositoryErrorsAbortTheRun(t *testing.T) {
	cases := map[string]func(f *reminderFixture){
		"list guilds":  func(f *reminderFixture) { f.guilds.err = errBoom },
		"list rules":   func(f *reminderFixture) { f.rules.err = errBoom },
		"list entries": func(f *reminderFixture) { f.entries.err = errBoom },
		"list subs":    func(f *reminderFixture) { f.subs.listErr = errBoom },
		"claim":        func(f *reminderFixture) { f.log.err = errBoom },
		"delete gone": func(f *reminderFixture) {
			f.sender.errs = map[string]error{"https://push.example.com/lia-phone": push.ErrGone}
			f.subs.deleteErr = errBoom
		},
	}
	for name, breakIt := range cases {
		t.Run(name, func(t *testing.T) {
			f := newReminderFixture()
			breakIt(f)

			_, err := f.run(service.RunOptions{})

			assert.ErrorIs(t, err, errBoom)
		})
	}
}
