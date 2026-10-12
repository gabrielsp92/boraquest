package unit_test

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/app/service"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/entry"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/guild"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/rule"
)

type fakeEntryRepo struct {
	got       entry.Entry
	getErr    error
	createErr error
	deleteErr error
	listed    []entry.Entry
	listErr   error
	summed    []service.MemberTotal
	sumErr    error
	photoSet  entry.Entry
	photoErr  error

	created   entry.Entry
	getArgs   [2]string
	deleteArg [2]string
	listArgs  struct {
		guildID, memberID string
		from, to          time.Time
	}
	sumArgs struct {
		guildID  string
		from, to time.Time
	}
	photoArgs [4]string
}

func (f *fakeEntryRepo) SetPhoto(_ context.Context, guildID, id, photoURL, photoPublicID string) (entry.Entry, error) {
	f.photoArgs = [4]string{guildID, id, photoURL, photoPublicID}
	return f.photoSet, f.photoErr
}

// fakePhotoUploader records what it was asked to store and returns url/storedID/err.
type fakePhotoUploader struct {
	url, storedID string
	err           error

	called           bool
	guildID, entryID string
	body             string
}

func (f *fakePhotoUploader) Upload(_ context.Context, guildID, entryID string, file io.Reader) (string, string, error) {
	f.called, f.guildID, f.entryID = true, guildID, entryID
	b, err := io.ReadAll(file)
	if err != nil {
		return "", "", err
	}
	f.body = string(b)
	return f.url, f.storedID, f.err
}

func (f *fakeEntryRepo) Create(_ context.Context, e entry.Entry) error {
	f.created = e
	return f.createErr
}

func (f *fakeEntryRepo) Get(_ context.Context, guildID, id string) (entry.Entry, error) {
	f.getArgs = [2]string{guildID, id}
	return f.got, f.getErr
}

func (f *fakeEntryRepo) Delete(_ context.Context, guildID, id string) error {
	f.deleteArg = [2]string{guildID, id}
	return f.deleteErr
}

func (f *fakeEntryRepo) ListByMember(_ context.Context, guildID, memberID string, from, to time.Time) ([]entry.Entry, error) {
	f.listArgs.guildID, f.listArgs.memberID, f.listArgs.from, f.listArgs.to = guildID, memberID, from, to
	return f.listed, f.listErr
}

func (f *fakeEntryRepo) SumByGuild(_ context.Context, guildID string, from, to time.Time) ([]service.MemberTotal, error) {
	f.sumArgs.guildID, f.sumArgs.from, f.sumArgs.to = guildID, from, to
	return f.summed, f.sumErr
}

var saoPaulo = mustLoadLocation()

func mustLoadLocation() *time.Location {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		panic(err)
	}
	return loc
}

func newEntryService(entries *fakeEntryRepo, rules *fakeRuleRepo, guilds fakeGuilds, now time.Time) *service.EntryService {
	return newEntryServiceWithPhotos(entries, rules, guilds, now, &fakePhotoUploader{})
}

func newEntryServiceWithPhotos(entries *fakeEntryRepo, rules *fakeRuleRepo, guilds fakeGuilds, now time.Time, photos *fakePhotoUploader) *service.EntryService {
	return service.NewEntryService(entries, rules, guilds, fakeIDs{id: "new-entry"}, fakeClock{now: now}, saoPaulo, photos)
}

var dailySumRule = rule.Rule{ID: "r1", GuildID: "g1", Name: "Água", Frequency: rule.FrequencyDaily, ScoreType: rule.ScoreTypeSum, Score: 10}
var weeklySumRule = rule.Rule{ID: "r2", GuildID: "g1", Name: "Treino", Frequency: rule.FrequencyWeekly, ScoreType: rule.ScoreTypeSum, Score: 20}
var decreaseRule = rule.Rule{ID: "r3", GuildID: "g1", Name: "Fast-food", Frequency: rule.FrequencyDaily, ScoreType: rule.ScoreTypeDecrease, Score: 15}

func TestEntryServiceCreateSumForSelf(t *testing.T) {
	entries := &fakeEntryRepo{}
	rules := &fakeRuleRepo{got: dailySumRule}
	at := time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC) // noon UTC-3

	e, err := newEntryService(entries, rules, fakeGuilds{guild: testGuild}, at).Create(context.Background(), "lia", service.EntryInput{RuleID: "r1", MemberID: "lia"})

	require.NoError(t, err)
	assert.Equal(t, "new-entry", e.ID)
	assert.Equal(t, "lia", e.MemberID)
	assert.Equal(t, "lia", e.LoggedBy)
	assert.Equal(t, 10, e.Points)
	assert.Equal(t, e, entries.created)
}

func TestEntryServiceCreateDecreaseForOtherMember(t *testing.T) {
	entries := &fakeEntryRepo{}
	rules := &fakeRuleRepo{got: decreaseRule}
	g := guild.Guild{ID: "g1", Name: "Família", UserIDs: []string{"lia", "beto"}}

	e, err := newEntryService(entries, rules, fakeGuilds{guild: g}, time.Now()).Create(context.Background(), "lia", service.EntryInput{RuleID: "r3", MemberID: "beto"})

	require.NoError(t, err)
	assert.Equal(t, "beto", e.MemberID)
	assert.Equal(t, "lia", e.LoggedBy)
	assert.Equal(t, -15, e.Points)
}

func TestEntryServiceCreateWeeklyPeriodKey(t *testing.T) {
	entries := &fakeEntryRepo{}
	rules := &fakeRuleRepo{got: weeklySumRule}
	at := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC) // Thursday

	e, err := newEntryService(entries, rules, fakeGuilds{guild: testGuild}, at).Create(context.Background(), "lia", service.EntryInput{RuleID: "r2", MemberID: "lia"})

	require.NoError(t, err)
	assert.Equal(t, entry.WeekStart(entry.DateOnly(at.In(saoPaulo))), e.PeriodKey)
}

func TestEntryServiceCreateRejectsNonMember(t *testing.T) {
	entries := &fakeEntryRepo{}
	rules := &fakeRuleRepo{}

	_, err := newEntryService(entries, rules, fakeGuilds{err: guild.ErrNotMember}, time.Now()).Create(context.Background(), "x", service.EntryInput{RuleID: "r1", MemberID: "x"})

	assert.ErrorIs(t, err, guild.ErrNotMember)
	assert.Empty(t, entries.created.ID)
}

func TestEntryServiceCreateRuleNotFound(t *testing.T) {
	entries := &fakeEntryRepo{}
	rules := &fakeRuleRepo{getErr: rule.ErrNotFound}

	_, err := newEntryService(entries, rules, fakeGuilds{guild: testGuild}, time.Now()).Create(context.Background(), "lia", service.EntryInput{RuleID: "r1", MemberID: "lia"})

	assert.ErrorIs(t, err, rule.ErrNotFound)
}

func TestEntryServiceCreateMemberNotInGuild(t *testing.T) {
	entries := &fakeEntryRepo{}
	rules := &fakeRuleRepo{got: decreaseRule}

	_, err := newEntryService(entries, rules, fakeGuilds{guild: testGuild}, time.Now()).Create(context.Background(), "lia", service.EntryInput{RuleID: "r3", MemberID: "ghost"})

	assert.ErrorIs(t, err, entry.ErrMemberNotInGuild)
	assert.Empty(t, entries.created.ID)
}

func TestEntryServiceCreateMemberEmpty(t *testing.T) {
	entries := &fakeEntryRepo{}
	rules := &fakeRuleRepo{got: decreaseRule}

	_, err := newEntryService(entries, rules, fakeGuilds{guild: testGuild}, time.Now()).Create(context.Background(), "lia", service.EntryInput{RuleID: "r3", MemberID: ""})

	assert.ErrorIs(t, err, entry.ErrMemberNotInGuild)
}

func TestEntryServiceCreateWrongMember(t *testing.T) {
	entries := &fakeEntryRepo{}
	rules := &fakeRuleRepo{got: dailySumRule}
	g := guild.Guild{ID: "g1", Name: "Família", UserIDs: []string{"lia", "beto"}}

	_, err := newEntryService(entries, rules, fakeGuilds{guild: g}, time.Now()).Create(context.Background(), "lia", service.EntryInput{RuleID: "r1", MemberID: "beto"})

	assert.ErrorIs(t, err, entry.ErrWrongMember)
	assert.Empty(t, entries.created.ID)
}

func TestEntryServiceCreateAlreadyLogged(t *testing.T) {
	entries := &fakeEntryRepo{createErr: entry.ErrAlreadyLogged}
	rules := &fakeRuleRepo{got: dailySumRule}

	_, err := newEntryService(entries, rules, fakeGuilds{guild: testGuild}, time.Now()).Create(context.Background(), "lia", service.EntryInput{RuleID: "r1", MemberID: "lia"})

	assert.ErrorIs(t, err, entry.ErrAlreadyLogged)
}

// TestEntryServiceCreateTimezoneConversion is the one bug most likely to be
// gotten wrong: a UTC time that has already rolled into "tomorrow" must still
// be "today" in America/Sao_Paulo (UTC-3), and vice versa.
func TestEntryServiceCreateTimezoneConversion(t *testing.T) {
	entries := &fakeEntryRepo{}
	rules := &fakeRuleRepo{got: dailySumRule}
	// 02:30 UTC on the 9th is 23:30 on the 8th in America/Sao_Paulo.
	at := time.Date(2026, 10, 9, 2, 30, 0, 0, time.UTC)

	e, err := newEntryService(entries, rules, fakeGuilds{guild: testGuild}, at).Create(context.Background(), "lia", service.EntryInput{RuleID: "r1", MemberID: "lia"})

	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC), e.OccurredOn, "must be the 8th in America/Sao_Paulo, not the 9th in raw UTC")
}

func TestEntryServiceDelete(t *testing.T) {
	at := time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC) // 12:00 in Sao Paulo, same day
	existing, err := entry.New("e1", "g1", "r1", "Água", rule.ScoreTypeSum, 10, "lia", "lia", rule.FrequencyDaily, at.In(saoPaulo))
	require.NoError(t, err)
	entries := &fakeEntryRepo{got: existing}

	err = newEntryService(entries, &fakeRuleRepo{}, fakeGuilds{guild: testGuild}, at).Delete(context.Background(), "lia", "e1")

	require.NoError(t, err)
	assert.Equal(t, [2]string{"g1", "e1"}, entries.deleteArg)
}

func TestEntryServiceDeleteRejectsNonMember(t *testing.T) {
	entries := &fakeEntryRepo{}

	err := newEntryService(entries, &fakeRuleRepo{}, fakeGuilds{err: guild.ErrNotMember}, time.Now()).Delete(context.Background(), "x", "e1")

	assert.ErrorIs(t, err, guild.ErrNotMember)
}

func TestEntryServiceDeleteNotFound(t *testing.T) {
	entries := &fakeEntryRepo{getErr: entry.ErrNotFound}

	err := newEntryService(entries, &fakeRuleRepo{}, fakeGuilds{guild: testGuild}, time.Now()).Delete(context.Background(), "lia", "e1")

	assert.ErrorIs(t, err, entry.ErrNotFound)
}

func TestEntryServiceDeleteNotOwn(t *testing.T) {
	entries := &fakeEntryRepo{got: entry.Entry{MemberID: "beto", ScoreType: rule.ScoreTypeSum}}

	err := newEntryService(entries, &fakeRuleRepo{}, fakeGuilds{guild: testGuild}, time.Now()).Delete(context.Background(), "lia", "e1")

	assert.ErrorIs(t, err, entry.ErrNotOwn)
}

func TestEntryServiceDeleteNotRemovable(t *testing.T) {
	entries := &fakeEntryRepo{got: entry.Entry{MemberID: "lia", ScoreType: rule.ScoreTypeDecrease}}

	err := newEntryService(entries, &fakeRuleRepo{}, fakeGuilds{guild: testGuild}, time.Now()).Delete(context.Background(), "lia", "e1")

	assert.ErrorIs(t, err, entry.ErrNotRemovable)
}

func TestEntryServiceDeleteNotToday(t *testing.T) {
	at := time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC)
	yesterday := entry.DateOnly(at.In(saoPaulo).AddDate(0, 0, -1))
	entries := &fakeEntryRepo{got: entry.Entry{MemberID: "lia", ScoreType: rule.ScoreTypeSum, OccurredOn: yesterday}}

	err := newEntryService(entries, &fakeRuleRepo{}, fakeGuilds{guild: testGuild}, at).Delete(context.Background(), "lia", "e1")

	assert.ErrorIs(t, err, entry.ErrNotToday)
}

func TestEntryServiceDeleteRepoError(t *testing.T) {
	entries := &fakeEntryRepo{got: entry.Entry{MemberID: "lia", ScoreType: rule.ScoreTypeSum, OccurredOn: entry.DateOnly(time.Now().In(saoPaulo))}, deleteErr: errBoom}

	err := newEntryService(entries, &fakeRuleRepo{}, fakeGuilds{guild: testGuild}, time.Now()).Delete(context.Background(), "lia", "e1")

	assert.ErrorIs(t, err, errBoom)
}

func TestEntryServiceListToday(t *testing.T) {
	entries := &fakeEntryRepo{listed: []entry.Entry{{ID: "e1"}}}
	at := time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC)

	got, from, to, today, err := newEntryService(entries, &fakeRuleRepo{}, fakeGuilds{guild: testGuild}, at).List(context.Background(), "lia", service.PeriodToday)

	require.NoError(t, err)
	assert.Equal(t, []entry.Entry{{ID: "e1"}}, got)
	wantDay := time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, wantDay, from)
	assert.Equal(t, wantDay, to)
	assert.Equal(t, wantDay, today)
	assert.Equal(t, "g1", entries.listArgs.guildID)
	assert.Equal(t, "lia", entries.listArgs.memberID)
}

func TestEntryServiceListWeek(t *testing.T) {
	entries := &fakeEntryRepo{}
	at := time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC) // Thursday in Sao Paulo

	_, from, to, today, err := newEntryService(entries, &fakeRuleRepo{}, fakeGuilds{guild: testGuild}, at).List(context.Background(), "lia", service.PeriodWeek)

	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC), from)
	assert.Equal(t, time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC), to)
	assert.Equal(t, time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC), today)
}

func TestEntryServiceListRejectsNonMember(t *testing.T) {
	entries := &fakeEntryRepo{}

	_, _, _, _, err := newEntryService(entries, &fakeRuleRepo{}, fakeGuilds{err: guild.ErrNotMember}, time.Now()).List(context.Background(), "x", service.PeriodToday)

	assert.ErrorIs(t, err, guild.ErrNotMember)
}

func TestEntryServiceListRepoError(t *testing.T) {
	entries := &fakeEntryRepo{listErr: errBoom}

	_, _, _, _, err := newEntryService(entries, &fakeRuleRepo{}, fakeGuilds{guild: testGuild}, time.Now()).List(context.Background(), "lia", service.PeriodToday)

	assert.ErrorIs(t, err, errBoom)
}

// photoAt is 12:00 in Sao Paulo; todaysSumEntry is lia's own quest from that day.
var photoAt = time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC)

var todaysSumEntry = entry.Entry{
	ID: "e1", GuildID: "g1", ScoreType: rule.ScoreTypeSum, MemberID: "lia", LoggedBy: "lia",
	OccurredOn: time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC),
}

func attachPhoto(entries *fakeEntryRepo, guilds fakeGuilds, photos *fakePhotoUploader, contentType string, size int64) (entry.Entry, error) {
	svc := newEntryServiceWithPhotos(entries, &fakeRuleRepo{}, guilds, photoAt, photos)
	return svc.AttachPhoto(context.Background(), "lia", "e1", contentType, size, strings.NewReader("img-bytes"))
}

func TestEntryServiceAttachPhoto(t *testing.T) {
	withPhoto := todaysSumEntry
	withPhoto.PhotoURL, withPhoto.PhotoPublicID = "https://cdn/e1.jpg", "boraquest/g1/entries/e1"
	entries := &fakeEntryRepo{got: todaysSumEntry, photoSet: withPhoto}
	photos := &fakePhotoUploader{url: "https://cdn/e1.jpg", storedID: "boraquest/g1/entries/e1"}

	got, err := attachPhoto(entries, fakeGuilds{guild: testGuild}, photos, "image/jpeg", 9)

	require.NoError(t, err)
	assert.Equal(t, withPhoto, got)
	assert.Equal(t, [2]string{"g1", "e1"}, entries.getArgs)
	assert.Equal(t, "g1", photos.guildID)
	assert.Equal(t, "e1", photos.entryID)
	assert.Equal(t, "img-bytes", photos.body)
	assert.Equal(t, [4]string{"g1", "e1", "https://cdn/e1.jpg", "boraquest/g1/entries/e1"}, entries.photoArgs)
}

func TestEntryServiceAttachPhotoUsesGuildTimezone(t *testing.T) {
	// 02:30 UTC on the 9th is still the 8th in Sao Paulo, so the 8th's entry is "today".
	entries := &fakeEntryRepo{got: todaysSumEntry}
	svc := newEntryServiceWithPhotos(entries, &fakeRuleRepo{}, fakeGuilds{guild: testGuild}, time.Date(2026, 10, 9, 2, 30, 0, 0, time.UTC), &fakePhotoUploader{})

	_, err := svc.AttachPhoto(context.Background(), "lia", "e1", "image/png", 9, strings.NewReader("x"))

	assert.NoError(t, err)
}

func TestEntryServiceAttachPhotoErrors(t *testing.T) {
	decrease := todaysSumEntry
	decrease.ScoreType = rule.ScoreTypeDecrease
	others := todaysSumEntry
	others.MemberID = "beto"
	hasPhoto := todaysSumEntry
	hasPhoto.PhotoURL = "https://cdn/old.jpg"
	yesterday := todaysSumEntry
	yesterday.OccurredOn = yesterday.OccurredOn.AddDate(0, 0, -1)

	cases := map[string]struct {
		entries     *fakeEntryRepo
		guilds      fakeGuilds
		photos      *fakePhotoUploader
		contentType string
		size        int64
		want        error
		uploads     bool
	}{
		"not in a guild":  {&fakeEntryRepo{}, fakeGuilds{err: guild.ErrNotMember}, &fakePhotoUploader{}, "image/jpeg", 9, guild.ErrNotMember, false},
		"entry not found": {&fakeEntryRepo{getErr: entry.ErrNotFound}, fakeGuilds{guild: testGuild}, &fakePhotoUploader{}, "image/jpeg", 9, entry.ErrNotFound, false},
		"slip entry":      {&fakeEntryRepo{got: decrease}, fakeGuilds{guild: testGuild}, &fakePhotoUploader{}, "image/jpeg", 9, entry.ErrPhotoNotAllowed, false},
		"not own entry":   {&fakeEntryRepo{got: others}, fakeGuilds{guild: testGuild}, &fakePhotoUploader{}, "image/jpeg", 9, entry.ErrPhotoNotOwn, false},
		"already a photo": {&fakeEntryRepo{got: hasPhoto}, fakeGuilds{guild: testGuild}, &fakePhotoUploader{}, "image/jpeg", 9, entry.ErrPhotoExists, false},
		"past day":        {&fakeEntryRepo{got: yesterday}, fakeGuilds{guild: testGuild}, &fakePhotoUploader{}, "image/jpeg", 9, entry.ErrPhotoTooLate, false},
		"wrong type":      {&fakeEntryRepo{got: todaysSumEntry}, fakeGuilds{guild: testGuild}, &fakePhotoUploader{}, "image/gif", 9, entry.ErrInvalidPhoto, false},
		"too big":         {&fakeEntryRepo{got: todaysSumEntry}, fakeGuilds{guild: testGuild}, &fakePhotoUploader{}, "image/jpeg", entry.MaxPhotoBytes + 1, entry.ErrInvalidPhoto, false},
		"upload fails":    {&fakeEntryRepo{got: todaysSumEntry}, fakeGuilds{guild: testGuild}, &fakePhotoUploader{err: errBoom}, "image/jpeg", 9, errBoom, true},
		"lost a race":     {&fakeEntryRepo{got: todaysSumEntry, photoErr: entry.ErrPhotoExists}, fakeGuilds{guild: testGuild}, &fakePhotoUploader{}, "image/jpeg", 9, entry.ErrPhotoExists, true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := attachPhoto(tc.entries, tc.guilds, tc.photos, tc.contentType, tc.size)

			assert.ErrorIs(t, err, tc.want)
			assert.Equal(t, tc.uploads, tc.photos.called, "uploader called")
			if !tc.uploads || tc.photos.err != nil {
				assert.Empty(t, tc.entries.photoArgs[0], "SetPhoto must not be called")
			}
		})
	}
}
