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
make run               # start API on :8080 (PORT env overrides)
make dev               # start API with hot reload (Air, config in .air.toml)
make test              # unit + integration
make cover             # all tests + enforce 100% coverage
make lint              # go vet + gofmt check
```
