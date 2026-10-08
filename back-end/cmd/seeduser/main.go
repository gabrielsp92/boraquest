// Command seeduser adds one user to the database from environment variables:
//
//	SEED_USER_NAME, SEED_USER_EMAIL, SEED_USER_PASSWORD  (required)
//	SEED_USER_GUILD                                      (required: id of an existing guild to join, e.g. familia)
//	DATABASE_URL                                         (defaults to the docker-compose DB)
//
// With -dry-run it validates the input and checks the database, but writes
// nothing (not even migrations). A user whose email already exists is left
// untouched, guild included. Run it through `make seed-user` /
// `make seed-user-dry-run`, which load back-end/.env first.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/app/service"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/infrastructure/idgen"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/infrastructure/postgres"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/infrastructure/security"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "validate and report what would happen without writing to the database")
	flag.Parse()
	log.SetFlags(0)
	ctx := context.Background()

	in := service.SeedUserInput{
		Name:     requireEnv("SEED_USER_NAME"),
		Email:    requireEnv("SEED_USER_EMAIL"),
		Password: requireEnv("SEED_USER_PASSWORD"),
		GuildID:  requireEnv("SEED_USER_GUILD"),
		DryRun:   *dryRun,
	}
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = postgres.DefaultURL
	}

	// Infrastructure
	pool, err := postgres.Open(ctx, databaseURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()
	if !in.DryRun {
		if err := postgres.Migrate(ctx, pool); err != nil {
			log.Fatalf("migrate database: %v", err)
		}
	}

	// Application
	seeder := service.NewUserSeedService(
		postgres.NewUserRepository(pool),
		postgres.NewGuildRepository(pool),
		security.BcryptHasher{},
		idgen.UUIDGenerator{},
	)
	res, err := seeder.Seed(ctx, in)
	if err != nil {
		log.Fatalf("seed user: %v", err)
	}

	prefix := ""
	if in.DryRun {
		prefix = "[dry run] "
	}
	u, g := res.User, res.Guild
	switch res.Outcome {
	case service.SeedCreated:
		fmt.Printf("%screated user %s <%s> with id %s in guild %s (%s)\n", prefix, u.Name, u.Email, u.ID, g.Name, g.ID)
	case service.SeedWouldCreate:
		fmt.Printf("%swould create user %s <%s> in guild %s (%s); nothing was written\n", prefix, u.Name, u.Email, g.Name, g.ID)
	case service.SeedAlreadyExists:
		fmt.Printf("%suser <%s> already exists with id %s; left untouched\n", prefix, u.Email, u.ID)
	}
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if strings.TrimSpace(v) == "" {
		log.Fatalf("%s must be set (in the environment or back-end/.env)", key)
	}
	return v
}
