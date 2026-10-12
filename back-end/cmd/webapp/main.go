// Command webapp starts the Boraquest HTTP API.
package main

import (
	"context"
	"log"
	"os"
	"time"
	_ "time/tzdata"

	cld "github.com/cloudinary/cloudinary-go/v2"

	"github.com/gabrielsp92/boraquest/back-end/cmd/webapp/routes"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/app/service"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/infrastructure/clock"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/infrastructure/cloudinary"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/infrastructure/idgen"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/infrastructure/postgres"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/infrastructure/security"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/controllers"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/middleware"
)

// version is overridden at build time via -ldflags "-X main.version=...".
var version = "dev"

const tokenTTL = 24 * time.Hour

func main() {
	ctx := context.Background()

	databaseURL := getenv("DATABASE_URL", postgres.DefaultURL)
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET must be set")
	}
	cloudName, cloudKey, cloudSecret := os.Getenv("CLOUDINARY_CLOUD_NAME"), os.Getenv("CLOUDINARY_API_KEY"), os.Getenv("CLOUDINARY_API_SECRET")
	if cloudName == "" || cloudKey == "" || cloudSecret == "" {
		log.Fatal("CLOUDINARY_CLOUD_NAME, CLOUDINARY_API_KEY and CLOUDINARY_API_SECRET must be set")
	}

	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		log.Fatalf("load timezone: %v", err)
	}

	// Infrastructure
	systemClock := clock.SystemClock{}
	pool, err := postgres.Open(ctx, databaseURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()
	if err := postgres.Migrate(ctx, pool); err != nil {
		log.Fatalf("migrate database: %v", err)
	}
	tokens := security.NewJWT(jwtSecret, tokenTTL, systemClock)
	cloudinaryClient, err := cld.NewFromParams(cloudName, cloudKey, cloudSecret)
	if err != nil {
		log.Fatalf("configure cloudinary: %v", err)
	}
	photoUploader := cloudinary.NewUploader(cloudinaryClient)

	// Application
	healthService := service.NewHealthService(systemClock, version)
	userRepository := postgres.NewUserRepository(pool)
	authService := service.NewAuthService(userRepository, security.BcryptComparer{}, tokens)
	profileService := service.NewProfileService(userRepository)
	ruleRepository := postgres.NewRuleRepository(pool)
	guildRepository := postgres.NewGuildRepository(pool)
	ruleService := service.NewRuleService(
		ruleRepository,
		guildRepository,
		idgen.UUIDGenerator{},
		systemClock,
	)
	prizeService := service.NewPrizeService(
		postgres.NewPrizeRepository(pool),
		guildRepository,
		systemClock,
	)
	entryRepository := postgres.NewEntryRepository(pool)
	entryService := service.NewEntryService(
		entryRepository,
		ruleRepository,
		guildRepository,
		idgen.UUIDGenerator{},
		systemClock,
		loc,
		photoUploader,
	)
	guildService := service.NewGuildService(guildRepository, userRepository)
	scoreboardService := service.NewScoreboardService(entryRepository, guildRepository, systemClock, loc)

	// Interface
	router := routes.NewRouter(routes.Controllers{
		Health:      controllers.NewHealthController(healthService),
		Auth:        controllers.NewAuthController(authService),
		Profile:     controllers.NewProfileController(profileService),
		Rules:       controllers.NewRuleController(ruleService),
		Prizes:      controllers.NewPrizeController(prizeService),
		Entries:     controllers.NewEntryController(entryService),
		Guild:       controllers.NewGuildController(guildService),
		Scoreboard:  controllers.NewScoreboardController(scoreboardService),
		RequireAuth: middleware.RequireAuth(tokens),
	})

	port := getenv("PORT", "8080")
	log.Printf("boraquest api listening on :%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
