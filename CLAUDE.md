# Boraquest monorepo

| Directory    | Stack                                  |
|--------------|----------------------------------------|
| `front-end/` | Next.js 16 (App Router) + Tailwind v4  |
| `back-end/`  | Go + Gin, DDD + clean architecture     |

Run each project's commands from inside its own directory. Dev servers are defined in `.claude/launch.json` (`front-end` on :3000, `back-end` on :8080).

## Front-end

- Read `front-end/AGENTS.md` before touching Next.js code — this Next.js version has breaking changes vs. training data.
- Requires Node >= 20.9 (`front-end/.nvmrc`). On this machine Node 22 lives at `/opt/homebrew/opt/node@22/bin`; the global `node` is 18 and will fail.

## Back-end — non-negotiable requirements

These apply to every back-end change. Do not skip them.

### Architecture (DDD + clean architecture)

Dependencies point inward only: `interface` → `app` → `domain`. `infrastructure` implements interfaces declared by inner layers.

| Layer          | Path                                          | Rules |
|----------------|-----------------------------------------------|-------|
| Domain         | `internal/src/domain/<aggregate>/`            | Entities, value objects, domain errors, validation. **No imports** of Gin, DB drivers, or any other layer. |
| Application    | `internal/src/app/service/`                   | Use-cases. Declares the interfaces it needs (repositories, clocks, gateways). Depends on interfaces only — never concrete adapters. No HTTP/SQL. |
| Infrastructure | `internal/src/infrastructure/<adapter>/`      | Implements app/domain interfaces (DB, external APIs, clock). One sub-package per adapter. |
| Interface      | `internal/src/interface/http/controllers/`    | Gin handlers: parse request → call use-case → write response via `interface/http/response`. No business logic. Declare the use-case interface the controller consumes. |
| Routes         | `cmd/webapp/routes/routes.go`                 | All routes under `/api/v1`, registered in `NewRouter`. |
| Wiring         | `cmd/webapp/main.go`                          | Explicit constructor injection only; no reflection/DI frameworks. Keep `main` thin. |

Interfaces belong in the layer that **uses** them, not the one that implements them.

### Adding a feature

Follow the `add-feature` skill checklist: domain → interfaces → app service → infrastructure → controller → routes → wiring → tests → OpenAPI (`api/openapi.yaml`). The health endpoint is the reference slice for all of these.

### Tests — full coverage required

- **Unit tests** in `test/unit/`: each layer in isolation, with hand-written fakes for the interfaces. Cover success and every error path.
- **Integration tests** in `test/integration/`: the full HTTP stack via `routes.NewRouter` + `httptest`, wired like `main.go` with real or in-memory adapters.
- Test files mirror the file under test (`user_service.go` → `test/unit/user_service_test.go`).
- **Coverage must stay at 100%** of `./internal/...` and `./cmd/webapp/routes/...` (`main.go` is exempt). Run `make cover`; it fails below the threshold.
- Before finishing any back-end change, `make lint` and `make cover` must both pass.

### Commands (from `back-end/`)

```
make db-up             # start local PostgreSQL (docker-compose.yml) — run/dev need it
make db-down           # stop it (data volume is kept)
make seed-user         # add one user from SEED_USER_* vars in back-end/.env (see .env.example); existing email = no-op
make seed-user-dry-run # same, but only validates and checks the DB — writes nothing
make migrate           # apply pending migrations to DATABASE_URL (required, no default)
make run               # migrate local DB, then start API on :8080 (PORT env overrides)
make dev               # start API with hot reload (Air, config in .air.toml)
make test              # unit + integration
make cover             # all tests + enforce 100% coverage (integration tests start their own Postgres via testcontainers — Docker must be running)
make lint              # go vet + gofmt check
```

### Runtime config, auth and data

- Env: `DATABASE_URL` (defaults to the docker-compose DB), `JWT_SECRET` (required; the Makefile sets a dev-only default), `PORT`.
- Migrations live in `internal/src/infrastructure/postgres/migrations/` (goose, embedded). The API never migrates itself: `cmd/migrate` (`make migrate`, needs `DATABASE_URL`) does. `make run`/`make dev` migrate the local DB first; production is migrated by Fly's `release_command` (`fly.toml`) before each deploy. Migrations must stay backward-compatible with the running release (expand/contract).
- Dev users `lia`, `beto`, `nena`, `caio` (`<id>@boraquest.dev`) are seeded by `0003_seed_dev_users.sql`; the dev password is in that file's header comment.
- Auth: `POST /api/v1/auth/login` returns an HS256 JWT (`sub` = user id). Protected routes use `middleware.RequireAuth`; read the caller with `middleware.UserID(c)`.
- Guilds live in Postgres (`guilds`, `guild_members`; migration `0004_guilds.sql` creates `familia` with the four dev users). For v1 a user belongs to at most one guild. Everything guild-scoped resolves the caller's guild via `GuildRepository.FindByMember`. New users join a guild via `make seed-user` (`SEED_USER_GUILD`).

## Deploy (CI/CD)

- `.github/workflows/ci.yml`: PR lint + test, per side touched; deploys nothing.
- `deploy-backend.yml` (push to `main`, `back-end/**`): `make lint`/`make cover` → `flyctl deploy` (`back-end/fly.toml`, `back-end/Dockerfile`). Fly runs `migrate` from the new image first; a failed migration aborts the deploy. Fly app secrets (`DATABASE_URL`, `JWT_SECRET`, `CLOUDINARY_*`, `VAPID_*`, `REMINDER_CRON_SECRET`) are set by hand with `make fly-secrets` (reads `back-end/.env`), never by CI.
- **Fly organization is always `boraquest`** (the account belongs to others too). Every Fly command goes through `back-end/scripts/fly-app.sh`, which reads the app from `fly.toml` and refuses unless it belongs to that org; pass `-a "$app"` explicitly and never run bare `fly` commands against an unchecked app. One-time targets: `make fly-app-create` (`--org boraquest`), `make fly-deploy-token`.
- `daily-reminder.yml` (schedule 23:05 UTC = 20:05 São Paulo, plus manual `workflow_dispatch` with `force`/`userId`): calls `POST /api/v1/internal/reminders/daily` on Fly with `REMINDER_CRON_SECRET`; the API sends the Web Push reminder (`todo/daily-reminder-push.md`). Needs the `production` environment secret `REMINDER_CRON_SECRET` and variable `BACKEND_PUBLIC_URL`. Scheduled runs stop after 60 days without repo activity.
- `deploy-frontend.yml` (push to `main`, `front-end/**`): lint → `vercel pull`/`build`/`deploy --prebuilt --prod`. `BACKEND_URL` must be set in the Vercel project's Production env, or the site silently proxies to localhost.
- One-time setup steps are in `todo/ci-cd-deploy.md` → Prerequisites.

## Front-end ↔ back-end

- `next.config.ts` rewrites `/api/v1/*` to `BACKEND_URL` (default `http://localhost:8080`), so the browser stays same-origin.
- `lib/api.ts` is the API client; `lib/auth.ts` holds the session (localStorage). `components/AuthGate.tsx` guards the `(tabs)` routes and sends signed-out users to `/entrar`.
