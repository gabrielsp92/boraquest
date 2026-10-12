// Command migrate applies every pending schema migration and exits:
//
//	DATABASE_URL  (required: no default, so it never migrates the wrong database by accident)
//
// It is the only thing that migrates the database: Fly runs it against
// production before each deploy (release_command in fly.toml), and
// `make run` / `make dev` run it against the docker-compose DB before starting
// the API. Run it by hand with `make migrate`.
package main

import (
	"context"
	"log"
	"os"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/infrastructure/postgres"
)

func main() {
	log.SetFlags(0)
	ctx := context.Background()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL must be set")
	}

	pool, err := postgres.Open(ctx, databaseURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()
	if err := postgres.Migrate(ctx, pool); err != nil {
		pool.Close()
		log.Fatalf("migrate database: %v", err)
	}
	log.Print("database is up to date")
}
