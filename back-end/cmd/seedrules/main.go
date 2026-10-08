// Command seedrules adds the rules listed in a JSON seed file (default
// seeds/rules.json) to a guild, using environment variables:
//
//	SEED_RULES_GUILD          (required: id of an existing guild, e.g. familia)
//	SEED_RULES_CREATOR_EMAIL  (required: a member of that guild, recorded as the creator)
//	DATABASE_URL              (defaults to the docker-compose DB)
//
// Rules whose name the guild already has are skipped, so it is safe to re-run.
// With -dry-run it validates the file and checks the database, but writes
// nothing (not even migrations). Run it through `make seed-rules` /
// `make seed-rules-dry-run`, which load back-end/.env first.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/app/service"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/infrastructure/clock"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/infrastructure/idgen"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/infrastructure/postgres"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/infrastructure/seedfile"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "validate and report what would happen without writing to the database")
	file := flag.String("file", "seeds/rules.json", "rules seed file")
	flag.Parse()
	log.SetFlags(0)
	ctx := context.Background()

	guildID := requireEnv("SEED_RULES_GUILD")
	creatorEmail := requireEnv("SEED_RULES_CREATOR_EMAIL")
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = postgres.DefaultURL
	}

	// Infrastructure
	f, err := os.Open(*file)
	if err != nil {
		log.Fatalf("open seed file: %v", err)
	}
	rules, err := seedfile.ReadRules(f)
	f.Close()
	if err != nil {
		log.Fatalf("read %s: %v", *file, err)
	}
	pool, err := postgres.Open(ctx, databaseURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()
	if !*dryRun {
		if err := postgres.Migrate(ctx, pool); err != nil {
			log.Fatalf("migrate database: %v", err)
		}
	}

	// Application
	seeder := service.NewRuleSeedService(
		postgres.NewRuleRepository(pool),
		postgres.NewGuildRepository(pool),
		postgres.NewUserRepository(pool),
		idgen.UUIDGenerator{},
		clock.SystemClock{},
	)
	res, err := seeder.Seed(ctx, service.RuleSeedInput{GuildID: guildID, CreatorEmail: creatorEmail, Rules: rules, DryRun: *dryRun})
	if err != nil {
		log.Fatalf("seed rules: %v", err)
	}

	prefix, verb := "", "added"
	if *dryRun {
		prefix, verb = "[dry run] ", "would add"
	}
	for _, r := range res.Added {
		fmt.Printf("%s%s: %s (%s, %s %d)\n", prefix, verb, r.Name, r.Frequency, r.ScoreType, r.Score)
	}
	for _, name := range res.Skipped {
		fmt.Printf("%sskipped (already exists): %s\n", prefix, name)
	}
	fmt.Printf("%s%s %d, skipped %d rules in guild %s (%s), created by <%s>\n",
		prefix, verb, len(res.Added), len(res.Skipped), res.Guild.Name, res.Guild.ID, res.Creator.Email)
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if strings.TrimSpace(v) == "" {
		log.Fatalf("%s must be set (in the environment or back-end/.env)", key)
	}
	return v
}
