# Scoreboard: real weekly and monthly standings on Guilda, with tasks completed

Status: Ready
Last updated: 2026-10-09
Depends on: [`quest-entries.md`](./quest-entries.md) — needs the `entries` table, the `entry` domain package (`DateOnly`/`WeekStart`), the fixed `America/Sao_Paulo` timezone wiring in `main.go`, and the `EntryRepository`/`GuildRepository`/`Clock` interfaces it introduces. Do not start this spec until that one is implemented.

## Summary
A new read-only `GET /api/v1/scoreboard?period=week|month` endpoint aggregates every guild member's entries (from `quest-entries.md`) into one row per member: their total points and how many quests they've completed. `app/(tabs)/guilda/page.tsx` is rewired to show that instead of the hardcoded `weekStandings`/`monthStandings` sample data, keeping its existing weekly/monthly toggle, leader crown, "Você" badge (now correctly tied to whoever is actually signed in, not a hardcoded id — see Decision log #8), and the (still non-functional) "Pedir auditoria" flow exactly as they are today.

## Problem
`app/(tabs)/guilda/page.tsx` already renders a ranked list of every guild member via `RankRow` (`members[s.member]`, `s.points`), but the data is `weekStandings`/`monthStandings` from `front-end/lib/data.ts:63-74` — four made-up numbers that never change no matter what anyone does on Hoje. There is also no "tasks completed" figure anywhere in the app today; the ask here ("list every guild member name followed by his scores and tasks completed") adds that alongside making the points real.

## Goals
- Every guild member appears on Guilda, by name, with their real total points for the selected period (week or month), even a member with zero activity.
- Every guild member also shows how many quests they've completed in that period (count of positive, `sum`-type entries — not slips).
- Switching the existing "Semana"/"Mês" toggle shows the correct period's real numbers.
- The ranking order and the leader crown keep working exactly as today, just driven by real data; the "Você" badge keeps its exact look but now correctly follows the real signed-in user (Decision log #8), not a hardcoded id.
- A member whose point total is negative (now possible for the first time, once slips are real — see Business rule 4) is still displayed correctly per `design/BRAND.md`'s "write points with the real minus" rule.

## Non-goals
- **The audit flow.** "Pedir auditoria" and the "Auditoria pendente" badge stay exactly as they are today: local-only, non-persisted, unrelated to any real backend state (confirmed — see Decision log). This spec does not add, remove, or wire anything about audits.
- **Prize cards.** `PrizeCard` stays on the `prizes` sample data from `lib/data.ts`. Unrelated to scores.
- **"Esta semana"/"Anteriores" on the Semana tab**, and `app/fim-de-semana` — untouched, already covered (or deferred) by `quest-entries.md`.
- **A guild-members/names API.** Display names and avatars still come from `lib/data.ts`'s hardcoded `members`/`memberList` (the same four dev users seeded into the one real guild, `"familia"`, by migration `0004_guilds.sql`). The new endpoint returns member *ids* only, same approach as `quest-entries.md` already established.
- **Declaring a winner, or anything about what happens at the end of a week/month.** Out of scope; depends on the deferred audit flow (per BRAND: "cannot be declared winner until approved").
- **Members joining/leaving mid-period, or a guild with more than the four seeded members.** Guilds now live in Postgres (`guilds`/`guild_members`, migration `0004_guilds.sql`; `back-end/internal/src/infrastructure/postgres/guild_repository.go`), and `ScoreboardService` enumerates whichever members `GuildRepository.FindByMember` returns for the caller's guild — not a hardcoded list in Go. In practice there is still only the one seeded guild (`"familia"`, the four dev users), so this is a limitation of today's data, not of this spec's code. Already a known limitation carried over from `quest-entries.md`.

## Current state
- `front-end/app/(tabs)/guilda/page.tsx`: `view` state toggles `"semana"|"mes"`; `standings = view === "semana" ? weekStandings : monthStandings` (both sample arrays, `lib/data.ts:63-74`); `pending` (local `MemberId[]`) seeds from `weekStandings.filter(s => s.auditPending)` and grows when "Pedir auditoria" → "Pedir auditoria" is confirmed in the sheet — never persisted, lost on reload, already a known gap.
- `RankRow` (`front-end/components/ui.tsx:185-200`): `{ position, member, points, leader, crown = leader, children }`. Renders `points` as a **plain number** (`<p className="bq-rank__score">{points}<small>pontos</small></p>`, line 194-197) — no sign formatting at all, unlike `Points`/`formatPoints` (`ui.tsx:51-54`) used elsewhere for signed deltas. Every existing sample value is positive, so this has never been exercised with a negative total; this spec's real aggregation makes a negative total possible for the first time (e.g. a member who only logs slips against themselves and never completes a quest).
- `Bar` (`ui.tsx:201-207`): renders a percent-width bar for the monthly view (`percent = Math.round((s.points / top) * 100)`); only used today with positive sample numbers.
- `Badge`/`Button` (`ui.tsx:92-99`, `16-27`): render the audit-pending badge, the "Você" badge, and the "Pedir auditoria" button inside `RankRow`'s `children` slot, one of the three depending on state — this spec adds a second child (the new tasks-completed caption) alongside whichever of those three is already there; `RankRow`'s `children` prop already accepts any `ReactNode`, including a fragment with two elements, so no component signature change is needed for that part.
- `quest-entries.md`'s backend (once implemented) provides: the `entries` table (`guild_id, member_id, score_type, points, occurred_on, …`); `back-end/internal/src/domain/entry/entry.go`'s `DateOnly`/`WeekStart`; `EntryRepository`, `GuildRepository`, `Clock` interfaces in `back-end/internal/src/app/service/`; and the `*time.Location` for `America/Sao_Paulo` constructed once in `main.go`. This spec reuses every one of those rather than redefining anything.
- `guild.Guild` (`back-end/internal/src/domain/guild/guild.go`) has `UserIDs []string` — the full, ordered list of every member of the guild. This spec zero-fills any member with no entries in the period from this list, so nobody is ever missing from the scoreboard.

## User stories
- As any guild member, I want to see everyone's real score for this week or this month, so that I know where I stand.
- As any guild member, I want to see how many quests each person actually completed, not just their points, so that a big score from slips-against-others doesn't look the same as a big score from doing the work.
- As a guild member who hasn't done anything yet this period, I want to still see myself listed (at zero), so that the list doesn't look broken or exclude me.

## UX flow

```mermaid
flowchart TD
  Load[Open Guilda] --> Fetch["GET /scoreboard?period=week"]
  Fetch -->|error| LoadError[EmptyState: Não deu para carregar]
  LoadError -->|Tentar de novo| Fetch
  Fetch -->|ok| List["PrizeCard (sample, unchanged) + ranked RankRow list"]
  List -->|toggle Segmented to Mês| FetchMonth["GET /scoreboard?period=month"]
  FetchMonth -->|error| LoadError
  FetchMonth -->|ok| ListMonth[Re-rendered ranked list for the month]
  List -->|"Pedir auditoria" (unchanged, local-only)| AuditSheet[Existing sheet, unchanged behavior]
```

### Screen: Guilda (`app/(tabs)/guilda/page.tsx`) — closest reference: `design/components/TelaGuilda`
- **Loading**: on first mount and on every toggle of the Segmented control, `<p className="bq-caption" role="status">Carregando guilda…</p>` replaces the list while that period's fetch is in flight (the already-loaded period's data, if any, is not shown stale — see Business rule 5).
- **Load error**: `EmptyState icon="users" title="Não deu para carregar" text="Confira sua conexão e tente de novo."` with a `Button onClick={retry}>Tentar de novo</Button>` — same pattern as `quest-entries.md`'s Hoje/Semana error states.
- **Success**: unchanged layout — `TopBar` with the Segmented "Semana"/"Mês" control, `PrizeCard` (unchanged, sample data), then one `RankRow` per guild member, `position` = rank (1-based, after sorting per Business rule 3), `leader = i === 0`, `crown` defaults to `leader`. Inside each row's `children`:
  - The existing conditional content, unchanged in structure: `Badge kind="audit"` if the member is in the local `pending` set, else `Badge>Você</Badge>` if it's the caller, else a `Button variant="small" icon="shield">Pedir auditoria</Button>` for everyone else. **"The caller" is now the real signed-in user** (`useSession().user?.id`), not the hardcoded `me` from `lib/data.ts` — see Decision log #8 and Open questions.
  - **New**, always, below that: `<p className="bq-caption">{completedLabel(s.completed)}</p>` (see Copy).
  - Monthly view keeps its `Bar percent={...}` instead of the audit/Você/Pedir-auditoria content, with the same new tasks-completed caption added below it.
- **Primary action**: Guilda still has no primary `.bq-btn` on its main view (unchanged — it's a read screen; "Pedir auditoria" is a `variant="small"` button, not the primary action, per BRAND's one-primary-action rule).

## Copy (pt-BR)
| Key / location | Text |
| --- | --- |
| Guilda, loading | Carregando guilda… |
| Guilda, load error title | Não deu para carregar |
| Guilda, load error text | Confira sua conexão e tente de novo. |
| Guilda, retry button | Tentar de novo |
| Tasks-completed caption, 0 | 0 tarefas concluídas |
| Tasks-completed caption, 1 | 1 tarefa concluída |
| Tasks-completed caption, 2+ | `{n}` tarefas concluídas |
| Everything else on Guilda (Segmented labels, "Pedir auditoria", "Auditoria pendente", "Você", prize labels, audit sheet copy) | Unchanged from the current `front-end/app/(tabs)/guilda/page.tsx`. |

## Business rules
1. **A member's points for a period are the sum of every one of their entries' `points` with `occurredOn` in that period** — identical definition to `quest-entries.md`'s Business rules 7-8 (every entry counts, `sum` and `decrease` alike, regardless of whether the entry's rule still exists).
2. **"Tasks completed" is the count of that member's `sum`-type entries with `occurredOn` in the period.** `decrease`-type entries (slips) never count toward this number, even though they do count toward points (rule 1).
3. **Standings are sorted by points descending; ties broken by tasks completed descending; remaining ties broken by member id ascending**, for a fully deterministic order on every load (no random/unstable ordering between reloads when several members are exactly tied).
4. **Every member of the guild (`guild.UserIDs`) appears exactly once**, even with zero entries in the period — zero points, zero tasks completed, not omitted.
5. **"Week" and "month" are computed in the same fixed `America/Sao_Paulo` timezone `quest-entries.md` established**, from `entry.DateOnly(clock.Now().In(loc))`. Week = Monday through Sunday (`entry.WeekStart`, reused as-is). Month = the 1st through the last day of the current calendar month in that timezone.
6. **Switching the Segmented control always re-fetches**; the previous period's data is not shown while the new one loads (no stale cross-period flash — avoids ever showing "Semana" numbers mislabeled as "Mês" for a frame).
7. **A member's total may be negative** (e.g. only self-logged slips, no completions). `RankRow`'s score display must render a negative total with the real minus character (`−`, not a hyphen), per `design/BRAND.md`'s point-formatting rule — see Architecture for the one-line fix.

## Data model

### Backend (new, added to `back-end/internal/src/app/service/` — no new domain package)
A scoreboard row is a pure projection over existing entries, not a persisted entity with its own identity or invariants — it lives in the application layer as a DTO, with no corresponding `domain/scoreboard` package.
```go
// back-end/internal/src/app/service/scoreboard_service.go
package service

import (
	"context"
	"sort"
	"time"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/entry"
)

type ScoreboardPeriod string

const (
	ScoreboardWeek  ScoreboardPeriod = "week"
	ScoreboardMonth ScoreboardPeriod = "month"
)

// MemberTotal is one guild member's raw aggregate over a period, as read from storage.
// Members with zero entries in the period are absent from the repository's result;
// ScoreboardService fills them in at zero (Business rule 4).
type MemberTotal struct {
	MemberID  string
	Points    int
	Completed int
}

// Standing is one ranked row of the scoreboard.
type Standing struct {
	MemberID  string
	Points    int
	Completed int
}

type ScoreboardService struct {
	entries EntryRepository // extended below; declared in entry_service.go per quest-entries.md
	guilds  GuildRepository // reused as-is from rule_service.go / entry_service.go
	clock   Clock
	loc     *time.Location
}

func NewScoreboardService(entries EntryRepository, guilds GuildRepository, clock Clock, loc *time.Location) *ScoreboardService {
	return &ScoreboardService{entries: entries, guilds: guilds, clock: clock, loc: loc}
}

func (s *ScoreboardService) List(ctx context.Context, callerID string, period ScoreboardPeriod) (standings []Standing, from, to, today time.Time, err error) {
	g, err := s.guilds.FindByMember(ctx, callerID)
	if err != nil {
		return nil, time.Time{}, time.Time{}, time.Time{}, err
	}
	today = entry.DateOnly(s.clock.Now().In(s.loc))
	if period == ScoreboardMonth {
		from = time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
		to = from.AddDate(0, 1, 0).AddDate(0, 0, -1) // last day of the month
	} else {
		from = entry.WeekStart(today)
		to = from.AddDate(0, 0, 6)
	}
	totals, err := s.entries.SumByGuild(ctx, g.ID, from, to)
	if err != nil {
		return nil, time.Time{}, time.Time{}, time.Time{}, err
	}
	byMember := make(map[string]MemberTotal, len(totals))
	for _, t := range totals {
		byMember[t.MemberID] = t
	}
	standings = make([]Standing, 0, len(g.UserIDs))
	for _, id := range g.UserIDs {
		t := byMember[id] // zero value when the member has no entries this period
		standings = append(standings, Standing{MemberID: id, Points: t.Points, Completed: t.Completed})
	}
	sort.Slice(standings, func(i, j int) bool {
		if standings[i].Points != standings[j].Points {
			return standings[i].Points > standings[j].Points
		}
		if standings[i].Completed != standings[j].Completed {
			return standings[i].Completed > standings[j].Completed
		}
		return standings[i].MemberID < standings[j].MemberID
	})
	return standings, from, to, today, nil
}
```

`EntryRepository` (declared in `back-end/internal/src/app/service/entry_service.go` by `quest-entries.md`) gains one method:
```go
// SumByGuild returns, for every member of guildID with at least one entry whose
// occurredOn falls in [from, to], their total points and count of sum-type entries.
// A member with zero entries in range is simply absent from the result.
SumByGuild(ctx context.Context, guildID string, from, to time.Time) ([]MemberTotal, error)
```

### Frontend (new, `front-end/lib/api.ts` additions)
```ts
export type ScoreboardPeriod = "week" | "month";
export type Standing = { memberId: string; points: number; completed: number };
export type Scoreboard = { standings: Standing[]; from: string; to: string; today: string };

export const getScoreboard = (period: ScoreboardPeriod) => apiFetch<Scoreboard>(`/scoreboard?period=${period}`);
```

### Diff against `front-end/lib/data.ts`
No changes to the file itself. `weekStandings`, `monthStandings`, and the `Standing` type exported there stay defined (unused by Guilda after this spec, but harmless — nothing else imports them, so either leave them or remove them in the same change; removing is preferred for cleanliness, but not required for correctness). `members`, `memberList`, `prizes` stay exactly as used today. `me` also stays defined and exported (still a sample constant, `"lia"`) — this spec just stops `guilda/page.tsx` from importing and reading it (see Architecture, Decision log #8).

## Architecture & technical decisions

### Backend
| Layer | File | Change |
| --- | --- | --- |
| App | `back-end/internal/src/app/service/entry_service.go` | Add `SumByGuild` to the `EntryRepository` interface (extends the one `quest-entries.md` defines; do not redeclare it elsewhere). |
| App | `back-end/internal/src/app/service/scoreboard_service.go` | New: `ScoreboardPeriod`, `MemberTotal`, `Standing`, `ScoreboardService` as in Data model. |
| Infrastructure | `back-end/internal/src/infrastructure/postgres/entry_repository.go` | Add `SumByGuild`, one grouped SQL query (below). |
| Interface | `back-end/internal/src/interface/http/controllers/scoreboard_controller.go` | New: `ScoreboardController.List`. |
| Routes | `back-end/cmd/webapp/routes/routes.go` | Add `Scoreboard *controllers.ScoreboardController`; new `/scoreboard` group (`RequireAuth`). |
| Wiring | `back-end/cmd/webapp/main.go` | Construct `service.NewScoreboardService(entryRepo, postgres.NewGuildRepository(pool), systemClock, loc)`, reusing the exact same `EntryRepository` variable (`entryRepo`) and `*time.Location` (`loc`) instances `quest-entries.md`'s wiring step already constructed, and the same `postgres.NewGuildRepository(pool)` call already used for `ruleService` (no new `GuildRepository` adapter — the old `back-end/internal/src/infrastructure/guild/static_repository.go` no longer exists; guilds are Postgres-backed per `0004_guilds.sql`). |

```go
func (r *EntryRepository) SumByGuild(ctx context.Context, guildID string, from, to time.Time) ([]service.MemberTotal, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT member_id,
		       COALESCE(SUM(points), 0)::int AS points,
		       COUNT(*) FILTER (WHERE score_type = 'sum')::int AS completed
		FROM entries
		WHERE guild_id = $1 AND occurred_on BETWEEN $2 AND $3
		GROUP BY member_id`,
		guildID, from, to)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (service.MemberTotal, error) {
		var t service.MemberTotal
		err := row.Scan(&t.MemberID, &t.Points, &t.Completed)
		return t, err
	})
}
```
(Depends on the `entries` table and its `guild_id, member_id, occurred_on, points, score_type` columns, all introduced by `quest-entries.md`'s `0005_entries.sql` — no new migration needed for this spec. Note `0004` is already taken by `0004_guilds.sql`; `quest-entries.md`'s migration is `0005_entries.sql`.)

`ScoreboardController` mirrors `entry_controller.go`'s period-validation style: parse and validate `period` (400 if missing or not `week`/`month`) before calling the service.

| Domain error | HTTP status |
| --- | --- |
| `guild.ErrNotMember` | 403 |
| missing/invalid `period` query param | 400 |

### Frontend
- `app/(tabs)/guilda/page.tsx` fetches `getScoreboard(view === "semana" ? "week" : "month")` in a `useEffect` keyed on `view` (re-runs on every toggle), following the same cleanup-flag pattern as `regras/page.tsx:45-53`. State: `standings: Standing[] | null`, `loadError: boolean`, reset to `null`/`false` at the start of each fetch so a period switch shows the loading state rather than the previous period's numbers (Business rule 6).
- `pending` (the local audit-pending set) changes its initializer from `weekStandings.filter(s => s.auditPending).map(s => s.member)` to `useState<MemberId[]>([])` — there is no real "already pending" signal anymore (there never was, functionally; sample data just faked one at load). Everything else about `pending`/`asking`/the audit sheet is untouched.
- **`me` is replaced by the real session** (Decision log #8): the page stops importing `me` from `lib/data.ts` and instead reads `const callerId = useSession().user?.id`, matching the pattern `quest-entries.md` already establishes for `hoje/page.tsx`/`semana/page.tsx`. The `s.member === me` check that drives the "Você" badge becomes `s.memberId === callerId`. While `callerId === undefined` (session not yet resolved client-side), render the existing loading caption — same convention as `AuthGate`/Hoje already use. `lib/data.ts`'s `me` constant itself is untouched (still exported, still used elsewhere as noted in `quest-entries.md`'s own diff) — only this page stops reading it.
- **Negative-total fix (Business rule 7)**: `RankRow` in `components/ui.tsx` changes its score line from
  ```tsx
  <p className="bq-rank__score">{points}<small>pontos</small></p>
  ```
  to
  ```tsx
  <p className="bq-rank__score">{points < 0 ? `−${Math.abs(points)}` : points}<small>pontos</small></p>
  ```
  Deliberately *not* reusing `formatPoints`/`Points` (`ui.tsx:51-54`) here — those always prefix a positive value with `+`, which reads naturally for a per-entry delta ("+10") but not for a total ("+85 pontos" would be unusual for a running total); this one-line change only swaps the minus character for negative totals, leaving positive and zero totals exactly as rendered today.
- `completedLabel(n: number)` — small helper (inline in `guilda/page.tsx` or added to `lib/ui` text helpers if one exists): `` n === 1 ? "1 tarefa concluída" : `${n} tarefas concluídas` ``.

## API / contracts

### `GET /api/v1/scoreboard?period={week|month}`
- **Auth**: required.
- **200**: `{ standings: [{ memberId, points, completed }], from: "YYYY-MM-DD", to: "YYYY-MM-DD", today: "YYYY-MM-DD" }`, `standings` sorted per Business rule 3, one entry per member of the caller's guild (Business rule 4).
- **400**: `period` missing or not `week`/`month`.
- **403**: caller belongs to no guild.

Add a `Standing`/`Scoreboard` schema and the `/scoreboard` path to `back-end/api/openapi.yaml`, mirroring the `/entries` section `quest-entries.md` adds (same `security: [bearerAuth: []]`, same `components.responses.Error`).

## Edge cases
| Case | Expected behavior |
| --- | --- |
| A member has zero entries in the selected period | Still listed, `points: 0`, `completed: 0`. |
| Every member is tied at zero (brand-new guild) | Order falls through to the member-id tie-break (Business rule 3) — deterministic, but the resulting "leader" crown goes to whoever sorts first alphabetically by id, not to anyone who's actually ahead. Accepted (no real leader exists yet); unchanged from how `RankRow`'s `leader={i===0}` already behaves with the old sample data whenever scores were tied. |
| A member's total is negative | Rendered with the real minus character, e.g. `−15 pontos` (Business rule 7). |
| Member switches "Semana" ↔ "Mês" rapidly | Each toggle starts a new fetch; the loading state is shown immediately on toggle (state reset to `null`), so a slow earlier response arriving late for a period the user already switched away from only matters if it lands after the newer request — use the same cleanup-flag-on-unmount/re-run pattern as `regras/page.tsx` so a stale response for an abandoned fetch is simply ignored. |
| Today falls in a month with 28, 29, 30, or 31 days | `MonthEnd` computed via `firstOfMonth.AddDate(0,1,0).AddDate(0,0,-1)` always lands on that month's real last day (Go's date arithmetic normalizes correctly across all month lengths and leap years). |
| A rule behind a `sum` entry was deleted (per `quest-entries.md`'s Business rule 6) | The entry still counts toward both `points` and `completed` — the scoreboard aggregates raw entries, never joins back to `rules`. |

## Accessibility, privacy & performance
- Accessibility: no new interactive controls beyond the existing Segmented/Button/Badge (all already meet the 48px tap target). The new tasks-completed caption is plain text, not color-coded, so it adds no color-only signal.
- Privacy: no new personal data; the response carries member ids only, same as `quest-entries.md`.
- Performance: one aggregation query over at most a few dozen rows (a household's weekly/monthly entry count) per request; negligible. No pagination needed.

## Decision log
| # | Decision | Rationale | Alternatives rejected |
| --- | --- | --- | --- |
| 1 | Wire the existing Guilda tab rather than build a new screen | BRAND.md already names Guilda "the weekly scoreboard, monthly standings"; a second screen would duplicate and conflict with it (confirmed by user) | A standalone scoreboard route — two places showing (different) standings, confusing |
| 2 | "Tasks completed" counts only `sum`-type entries | A slip isn't a completed task; it already affects points as a deduction (confirmed by user) | Count every entry — a member who got caught twice would show as having "completed" those slips, which reads backwards |
| 3 | Leave "Pedir auditoria"/"Auditoria pendente" completely untouched | No real audit backend exists yet (deferred in `quest-entries.md`); keeps this spec's scope to "make the numbers real" only (confirmed by user) | Hide the audit UI until a real backend exists — a legitimate alternative, but a UI-removal decision that's cheap to make later and not needed to ship real scores now |
| 4 | Tie-break: points desc → tasks completed desc → member id asc | Fully deterministic ordering with no extra lookup; for the current hardcoded four dev users (`lia, beto, nena, caio`), id order happens to already match name order, so this reads correctly today (confirmed by user) | Tie-break by real display name — would require `ScoreboardService` to depend on a user/name lookup it otherwise doesn't need, for a case (exact point *and* task-count tie) that's rare and, if it ever stops matching name order, cheap to revisit |
| 5 | "Month" = calendar month in `America/Sao_Paulo`, 1st to last day | Matches the discrete week/month mental model already used by `pastWeeks` sample data and by `quest-entries.md`'s week boundary; resets predictably on the 1st (confirmed by user) | Rolling 30 days — smoother but introduces a second, different notion of "period" alongside the fixed-Monday week |
| 6 | `Standing`/`MemberTotal` live as plain DTOs in `app/service`, no `domain/scoreboard` package | A scoreboard row has no identity or invariants of its own — it's purely a computed projection over `entry.Entry` rows, which already is a real domain entity | Add a `domain/scoreboard` package for symmetry with `rule`/`entry`/`guild`/`user` — unnecessary ceremony for a value with nothing to validate |
| 7 | Fix `RankRow`'s score line to use a real minus character for negative totals, without reusing the always-signed `Points`/`formatPoints` helper | `design/BRAND.md` requires the real minus for any negative number shown in the UI; a running total reading "+85 pontos" would look wrong for the common positive case, so this is a narrower, one-line fix rather than reusing the per-delta formatter | Reuse `Points`/`formatPoints` for the total too — would prefix every positive total with a `+`, a visible regression from today's unsigned-positive rendering |
| 8 | Guilda's "Você" badge compares against the real signed-in user (`useSession().user?.id`), not the hardcoded `me` from `lib/data.ts` | `quest-entries.md` (now Ready) already switched Hoje/Semana to the real session for the exact same reason: once that spec ships, `me` ("lia") no longer reflects who's actually signed in, so leaving Guilda on it would make "Você" always highlight Lia's row regardless of who's logged in on that device — a visible bug for every other dev user, not just a cosmetic inconsistency (confirmed by user) | Leave Guilda on `me` from `lib/data.ts` — smaller diff, but ships a known-wrong badge the moment `quest-entries.md` lands; there's no reason to special-case Guilda when every other page this session touches already made the switch |

## Implementation plan
1. **Backend app service — extend the interface**: add `SumByGuild` to `EntryRepository` in `back-end/internal/src/app/service/entry_service.go`. Done when the interface compiles with the method signature from Data model (the Postgres implementation comes in step 2). *(Depends on `quest-entries.md` being implemented.)*
2. **Backend infrastructure**: implement `SumByGuild` in `back-end/internal/src/infrastructure/postgres/entry_repository.go` per Architecture. Done when a manual query against a locally seeded `entries` table returns the expected grouped totals.
3. **Backend app service — scoreboard**: new `back-end/internal/src/app/service/scoreboard_service.go` exactly as in Data model. Unit tests with a hand-written `EntryRepository` fake covering: zero-fill for members with no entries, the three-level sort/tie-break, week vs. month date ranges (including a January-to-December boundary and a 28/29/30/31-day month boundary), and `guild.ErrNotMember` propagation. *(Depends on step 1.)*
4. **Backend interface + routes**: `back-end/internal/src/interface/http/controllers/scoreboard_controller.go` (period validation, `writeScoreboardError`), register `/scoreboard` (`RequireAuth`) in `back-end/cmd/webapp/routes/routes.go`. *(Depends on step 3.)*
5. **Backend wiring**: construct `ScoreboardService` in `back-end/cmd/webapp/main.go`, reusing the existing `EntryRepository`/`GuildRepository`/`loc` instances. Done when `make run` serves `GET /api/v1/scoreboard?period=week` for a logged-in dev user, returning all four seeded members at zero on a fresh database. *(Depends on step 4.)*
6. **Backend OpenAPI**: add `Standing`/`Scoreboard` schemas and the `/scoreboard` path to `back-end/api/openapi.yaml`.
7. **Backend tests to 100%**: `back-end/test/unit/scoreboard_service_test.go`, `scoreboard_controller_test.go`; extend `back-end/test/integration/` with a `scoreboard_test.go` (mirroring `rules_test.go`/`entries_test.go`'s style: seed entries across two members and two periods via the real `/entries` endpoints, then assert the scoreboard's totals, tie order, and zero-fill for an untouched member) and add `Scoreboard` to `newServer()`'s wired controllers in `main_test.go`. Done when `make lint && make cover` pass at 100%. *(Depends on steps 1-6.)*
8. **Frontend API client**: add `ScoreboardPeriod`/`Standing`/`Scoreboard`/`getScoreboard` to `front-end/lib/api.ts`. *(Depends on step 6 for the final shape; can be stubbed earlier.)*
9. **Frontend component fix**: apply the one-line negative-total fix to `RankRow` in `front-end/components/ui.tsx` (Architecture).
10. **Frontend Guilda rewrite**: `front-end/app/(tabs)/guilda/page.tsx` — fetch on mount and on every `view` toggle, loading/error states, zero-init `pending`, switch the caller id from the imported `me` to `useSession().user?.id` (Decision log #8), new tasks-completed caption in every row. *(Depends on steps 8-9.)*
11. **Manual verification**: per Test plan below.

## Acceptance criteria
- [ ] Given a signed-in member opens Guilda, when the scoreboard is loading, then "Carregando guilda…" is shown and no list renders yet.
- [ ] Given the fetch fails, when Guilda renders, then the "Não deu para carregar" EmptyState with a working "Tentar de novo" button is shown.
- [ ] Given a fresh guild with zero entries ever, when Guilda renders "Semana", then all four members are listed, each with 0 pontos and "0 tarefas concluídas".
- [ ] Given one member completed 3 daily quests this week (30 points) and another logged 2 slips against themselves (−10 points) and completed nothing, when Guilda renders "Semana", then the first shows "30 pontos" / "3 tarefas concluídas" and the second shows "−15 pontos" / "0 tarefas concluídas" — with a real minus character on the second.
- [ ] Given two members have exactly equal points and equal tasks-completed this period, when Guilda renders, then they appear in ascending member-id order, consistently across reloads.
- [ ] Given the member toggles from "Semana" to "Mês", when the toggle happens, then a new fetch starts, the loading state is shown (not stale "Semana" data), and the resulting list reflects the calendar month's totals.
- [ ] Given today is any day of a 28-, 29-, 30-, or 31-day month, when "Mês" is selected, then the queried range is the 1st through that month's real last day.
- [ ] Given the caller is viewing their own row, when Guilda renders "Semana", then that row still shows the "Você" badge (unchanged) alongside the new tasks-completed caption.
- [ ] Given a member other than Lia is signed in (e.g. `beto@boraquest.dev`), when Guilda renders, then the "Você" badge appears on Beto's row, not Lia's — confirming the caller is resolved from the real session, not the hardcoded `me` sample constant.
- [ ] Given a non-caller member's row on "Semana", when Guilda renders, then it still shows "Pedir auditoria" (unchanged) alongside the new tasks-completed caption, and tapping it still behaves exactly as today (local-only, no persistence).
- [ ] Given `GET /scoreboard` is called with no `period` or an invalid one, then the API returns 400.
- [ ] Given the caller belongs to no guild, when `GET /scoreboard` is called, then the API returns 403.
- [ ] Given a rule behind a logged `sum` entry has since been deleted, when the scoreboard is computed, then that entry's points and completed-count still contribute (no join to `rules`, matching `quest-entries.md`'s Business rule 6).

## Test plan
No test-runner changes beyond what already exists. Back-end: extend `make test`/`make cover` with the new unit and integration tests in the Implementation plan, keeping the 100% coverage gate on `./internal/...` and `./cmd/webapp/routes/...`. Front-end: manual verification only, on a 390px viewport:
1. Sign in as `lia@boraquest.dev`. On a freshly migrated/seeded database, open Guilda; confirm all four dev users are listed at 0 pontos / 0 tarefas concluídas.
2. On Hoje, check two daily quests and log one slip against yourself; return to Guilda; confirm your row updates to the right points (completions minus the slip) and "2 tarefas concluídas" (the slip doesn't count).
3. In a second browser profile signed in as `beto@boraquest.dev`, log a slip against Lia; reload Lia's Guilda; confirm Lia's points dropped further and her tasks-completed count is unchanged.
4. Toggle "Semana" → "Mês"; confirm the loading state appears briefly and the monthly numbers (which include the same entries, since it's still within the current month) match.
5. In Regras, delete a rule Lia already completed today; reload Guilda; confirm her points and tasks-completed count are unaffected.
6. With the back-end stopped, reload Guilda; confirm the retry EmptyState appears, and that restarting the back-end and tapping "Tentar de novo" recovers.
7. In the second browser profile signed in as `beto@boraquest.dev`, open Guilda; confirm the "Você" badge is on Beto's row, not Lia's (Decision log #8).

## Out of scope / follow-ups
- Wiring the audit flow for real (persisted "Auditoria pendente", approval blocking "winner" status) — separate, larger feature; this spec explicitly leaves the current local-only UI untouched (Non-goals).
- A guild-members/names API, so the front-end stops hardcoding `members`/`memberList` — same deferred item as in `quest-entries.md`.
- Removing the now-unused `weekStandings`/`monthStandings` sample exports from `lib/data.ts` — harmless to leave; cleanup, not correctness.

## Open questions
None. (Decision log #8 — switching Guilda's "Você" badge to the real signed-in user — was confirmed by the user.)

## Notes for the implementing agent
- Implement `quest-entries.md` first and completely — this spec adds a method to an interface and a new service that both assume that spec's files already exist exactly as specified there.
- Read `front-end/AGENTS.md` and `front-end/design/BRAND.md` before touching front-end files; read the root `CLAUDE.md` before touching back-end files (DDD layering, 100% coverage gate).
- Do not add a `domain/scoreboard` package (Decision log #6) and do not add a user/name lookup to `ScoreboardService` (Decision log #4) — both are deliberate simplifications, not oversights.
- Do not touch anything about the audit sheet/button/badge beyond the one required change (zero-initializing `pending` instead of seeding it from sample data) — see Non-goals and Decision log #3.
- The one-line `RankRow` fix (Architecture, Decision log #7) is easy to skip since it only matters once a negative total is actually possible — do not skip it; write the acceptance criterion's exact scenario (a member with more slip deductions than completions) as a test.
- This spec was updated after `quest-entries.md` reached `Status: Ready`, to fix every reference that assumed the old `guild.Default`/`back-end/internal/src/infrastructure/guild/static_repository.go` static guild model (now deleted — guilds are Postgres-backed, see `back-end/internal/src/infrastructure/postgres/guild_repository.go`) and the old `0004_entries.sql` migration number (entries are `0005_entries.sql`; `0004` is `0004_guilds.sql`). If you find another leftover reference to either, treat it as a defect in this spec, not as a signal to follow it.
