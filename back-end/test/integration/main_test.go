package integration_test

import (
	"context"
	"io"
	"log"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/gabrielsp92/boraquest/back-end/cmd/webapp/routes"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/app/service"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/push"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/reminder"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/infrastructure/clock"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/infrastructure/idgen"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/infrastructure/postgres"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/infrastructure/security"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/controllers"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/middleware"
)

const (
	testJWTSecret = "integration-secret"
	// devPassword is the password of every user seeded by 0003_seed_dev_users.
	devPassword = "boraquest-dev"
	// outsiderID is a user that exists but belongs to no guild.
	outsiderID = "intruso"
	// testCronSecret guards POST /internal/reminders/daily in tests.
	testCronSecret = "integration-cron-secret"
	// testVAPIDPublicKey is served by GET /push/public-key in tests.
	testVAPIDPublicKey = "test-vapid-public-key"
)

var (
	// databaseURL points at the PostgreSQL container shared by every test.
	databaseURL string
	// pool is migrated and seeded before any test runs.
	pool *pgxpool.Pool
	// saoPaulo is the fixed timezone every entry's day/week boundary is computed in.
	saoPaulo *time.Location
)

// TestMain starts one throwaway PostgreSQL container for the whole package.
func TestMain(m *testing.M) {
	ctx := context.Background()
	ctr, err := tcpostgres.Run(ctx, "postgres:17-alpine",
		tcpostgres.WithDatabase("boraquest"),
		tcpostgres.WithUsername("boraquest"),
		tcpostgres.WithPassword("boraquest"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		log.Fatalf("start postgres container (is Docker running?): %v", err)
	}
	code := run(ctx, m, ctr)
	if err := ctr.Terminate(ctx); err != nil {
		log.Printf("terminate postgres container: %v", err)
	}
	os.Exit(code)
}

func run(ctx context.Context, m *testing.M, ctr *tcpostgres.PostgresContainer) int {
	var err error
	saoPaulo, err = time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		log.Fatalf("load timezone: %v", err)
	}
	if databaseURL, err = ctr.ConnectionString(ctx, "sslmode=disable"); err != nil {
		log.Fatalf("connection string: %v", err)
	}
	if pool, err = postgres.Open(ctx, databaseURL); err != nil {
		log.Fatalf("open: %v", err)
	}
	defer pool.Close()
	if err := postgres.Migrate(ctx, pool); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, password_hash) SELECT $1, 'Intruso', 'intruso@boraquest.dev', password_hash FROM users WHERE id = 'lia'`,
		outsiderID); err != nil {
		log.Fatalf("seed outsider: %v", err)
	}
	return m.Run()
}

// newServer wires the app exactly like cmd/webapp/main.go, against the test database.
// The daily reminder sees 20:30 today (São Paulo) and a fake push sender.
func newServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv, _ := newReminderServer(t, todayAt(20, 30))
	return srv
}

// newReminderServer is newServer with the daily reminder's clock fixed at reminderNow;
// it also returns the fake push sender so tests can inspect what was sent.
func newReminderServer(t *testing.T, reminderNow time.Time) (*httptest.Server, *fakePushSender) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	systemClock := clock.SystemClock{}
	sender := &fakePushSender{errs: map[string]error{}}
	tokens := security.NewJWT(testJWTSecret, time.Hour, systemClock)
	router := routes.NewRouter(routes.Controllers{
		Health:  controllers.NewHealthController(service.NewHealthService(systemClock, "test")),
		Auth:    controllers.NewAuthController(service.NewAuthService(postgres.NewUserRepository(pool), security.BcryptComparer{}, tokens)),
		Profile: controllers.NewProfileController(service.NewProfileService(postgres.NewUserRepository(pool))),
		Rules: controllers.NewRuleController(service.NewRuleService(
			postgres.NewRuleRepository(pool),
			postgres.NewGuildRepository(pool),
			idgen.UUIDGenerator{},
			systemClock,
		)),
		Prizes: controllers.NewPrizeController(service.NewPrizeService(
			postgres.NewPrizeRepository(pool),
			postgres.NewGuildRepository(pool),
			systemClock,
		)),
		Entries: controllers.NewEntryController(service.NewEntryService(
			postgres.NewEntryRepository(pool),
			postgres.NewRuleRepository(pool),
			postgres.NewGuildRepository(pool),
			idgen.UUIDGenerator{},
			systemClock,
			saoPaulo,
			fakePhotoStore{},
		)),
		Guild: controllers.NewGuildController(service.NewGuildService(
			postgres.NewGuildRepository(pool),
			postgres.NewUserRepository(pool),
		)),
		Scoreboard: controllers.NewScoreboardController(service.NewScoreboardService(
			postgres.NewEntryRepository(pool),
			postgres.NewGuildRepository(pool),
			systemClock,
			saoPaulo,
		)),
		Weeks: controllers.NewWeekHistoryController(service.NewWeekHistoryService(
			postgres.NewWeekResultRepository(pool),
			postgres.NewEntryRepository(pool),
			postgres.NewPrizeRepository(pool),
			postgres.NewGuildRepository(pool),
			systemClock,
			saoPaulo,
		)),
		Push: controllers.NewPushController(service.NewPushService(
			postgres.NewPushSubscriptionRepository(pool),
			idgen.UUIDGenerator{},
			systemClock,
			testVAPIDPublicKey,
		)),
		Reminders: controllers.NewReminderController(service.NewReminderService(
			postgres.NewGuildRepository(pool),
			postgres.NewRuleRepository(pool),
			postgres.NewEntryRepository(pool),
			postgres.NewPushSubscriptionRepository(pool),
			postgres.NewDailyReminderRepository(pool),
			sender,
			fixedClock{now: reminderNow},
			saoPaulo,
		)),
		RequireAuth: middleware.RequireAuth(tokens),
		RequireCron: middleware.RequireCronSecret(testCronSecret),
	})
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)
	return srv, sender
}

// todayAt returns h:m today in São Paulo, the day the API's real clock is on.
func todayAt(h, m int) time.Time {
	y, mo, d := time.Now().In(saoPaulo).Date()
	return time.Date(y, mo, d, h, m, 0, 0, saoPaulo)
}

// fixedClock always reads now.
type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

// fakePushSender stands in for the Web Push adapter: it records every send and
// returns the error configured for that endpoint (nil by default).
type fakePushSender struct {
	mu   sync.Mutex
	errs map[string]error
	sent []string // endpoints, in send order
	last reminder.Payload
}

func (f *fakePushSender) Send(_ context.Context, s push.Subscription, p reminder.Payload) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, s.Endpoint)
	f.last = p
	return f.errs[s.Endpoint]
}

func (f *fakePushSender) sentTo() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.sent...)
}

// resetRules empties the rules table so each test starts from a clean guild.
func resetRules(t *testing.T) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `DELETE FROM rules`)
	require.NoError(t, err)
}

// resetPrizes empties the prizes table so each test starts from a clean guild.
func resetPrizes(t *testing.T) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `DELETE FROM prizes`)
	require.NoError(t, err)
}

// resetPush empties the push subscription and daily reminder tables.
func resetPush(t *testing.T) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `DELETE FROM push_subscriptions; DELETE FROM daily_reminders`)
	require.NoError(t, err)
}

// resetEntries empties the entries table so each test starts from a clean guild.
func resetEntries(t *testing.T) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `DELETE FROM entries`)
	require.NoError(t, err)
}

// fakePhotoStore stands in for Cloudinary so the suite never needs network access or secrets.
type fakePhotoStore struct{}

func (fakePhotoStore) Upload(_ context.Context, guildID, entryID string, file io.Reader) (string, string, error) {
	if _, err := io.Copy(io.Discard, file); err != nil {
		return "", "", err
	}
	id := "boraquest/" + guildID + "/entries/" + entryID
	return "https://photos.test/" + id + ".jpg", id, nil
}
