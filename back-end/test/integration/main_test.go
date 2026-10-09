package integration_test

import (
	"context"
	"log"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/gabrielsp92/boraquest/back-end/cmd/webapp/routes"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/app/service"
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
)

var (
	// databaseURL points at the PostgreSQL container shared by every test.
	databaseURL string
	// pool is migrated and seeded before any test runs.
	pool *pgxpool.Pool
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
func newServer(t *testing.T) *httptest.Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	systemClock := clock.SystemClock{}
	tokens := security.NewJWT(testJWTSecret, time.Hour, systemClock)
	router := routes.NewRouter(routes.Controllers{
		Health: controllers.NewHealthController(service.NewHealthService(systemClock, "test")),
		Auth:   controllers.NewAuthController(service.NewAuthService(postgres.NewUserRepository(pool), security.BcryptComparer{}, tokens)),
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
		RequireAuth: middleware.RequireAuth(tokens),
	})
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)
	return srv
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
