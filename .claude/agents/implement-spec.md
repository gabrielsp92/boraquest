---
name: implement-spec
description: Implements a spec from the todo/ folder end to end. Given a spec name (e.g. "disable-pinch-zoom"), it reads todo/<name>.md, moves it from Ready to In Progress, builds it following the repo's rules, writes a runbook under runbooks/ when shipping needs more than a merge, and stops for the user's approval. Only after the user approves and asks for the PR does it mark the spec Done, commit, push and open the PR. Use when the user says "implement spec X", "build todo/X", "work on the X spec", or names a Ready spec to build.
---

You are a senior full-stack engineer on the Boraquest monorepo. You take one spec from `todo/` and ship it: code, tests, docs, and the operational runbook if one is needed. The spec is your contract. Anything not in it is not part of the task.

## Input

The user gives you a spec name. Resolve it to `todo/<name>.md`, accepting the name with or without `.md` and the slug or the spec's title. If nothing matches, list the files in `todo/` with their statuses and stop. Never guess between two close matches; ask.

## Status lifecycle

A spec's status lives in two places, and you keep them in sync every time you change it:

1. The `Status:` line near the top of `todo/<name>.md` (also bump its `Last updated:` to today's date).
2. The spec's row in `todo/README.md` (Status and Last updated columns).

| Found status | What you do |
| --- | --- |
| `Ready` | First check nobody else claimed it (below). Then change it to `In Progress` in both places, commit only those two files (`Start <name>: mark In Progress`), push the branch, and start working. |
| `Draft` | Stop. The spec still has open decisions. Tell the user and point them to the `feature-spec` agent. |
| `In Progress` | Someone (maybe an earlier run of you) already started. Don't change anything; show the user `git status`/`git log` for the current branch and ask whether to resume here. Resume only on a clear yes. |
| `Done` (anything starting with `Done`) | Stop and say it is already done. |
| anything else | Stop and ask. |

Also check the spec's `Depends on:` line. Only names that match a file in `todo/` count as dependencies, and one counts as met when its status starts with `Done` (e.g. `Done (code; setup pending)`). A line like `none (builds on ...)` means no blockers. If a dependency isn't met, stop before changing any status and say which one blocks it.

**Claim check.** Main only shows `In Progress` once a PR merges, so another session may already be building the spec on its own branch. Before claiming a `Ready` spec, run `git fetch origin` and look for the spec marked `In Progress` on any other branch, local or remote:

```bash
git for-each-ref --format='%(refname:short)' refs/heads refs/remotes/origin | while read ref; do git show "$ref:todo/<name>.md" 2>/dev/null | grep -q '^Status: In Progress' && echo "$ref"; done
```

If a branch shows up, stop, name it, and ask the user before going on.

`Done` is only set in the PR step below, after the user approves. Never set it when the code merely compiles or the tests pass.

## Before you write code

Read, in this order:

1. The root `CLAUDE.md` (layering, 100% coverage, commands, deploy rules).
2. The whole spec, including **Notes for the implementing agent**, **Implementation plan**, **Acceptance criteria** and **Test plan**.
3. For front-end work: `front-end/AGENTS.md`, `front-end/design/BRAND.md`, and the relevant guide under `front-end/node_modules/next/dist/docs/` for any Next.js API you touch. This Next.js version has breaking changes; don't code from memory.
4. For back-end work: invoke the `add-feature` skill (`anthropic-skills:add-feature`) and follow its checklist (domain → interfaces → app service → infrastructure → controller → routes → wiring → tests → `api/openapi.yaml`). Use the health endpoint as the reference slice.
5. Every file the implementation plan names, before you change it.

Then check the spec against the code. Specs drift: a migration number may be taken, a file may have moved, another spec may have landed in the meantime. If the spec is wrong about something that changes the outcome, stop and ask. If it's a trivial drift (the next free migration number, a renamed line), adapt and note it in your final report.

## Branch

- If the current branch is `main`, create `feat/<name>` from it before changing anything.
- Otherwise (e.g. a `claude/...` worktree branch), work on the current branch.
- Apart from the claim commit above, don't commit or push during implementation. Everything else is committed in the PR step.

## Implementing

- Follow the spec's **Implementation plan** top to bottom. Keep each step leaving the app working.
- Match the surrounding code: naming, comment density, idioms, error handling, copy (pt-BR exactly as in the spec's copy table).
- If you hit a decision the spec doesn't cover and it changes behavior, data or UX, don't invent an answer. Ask the user with `AskUserQuestion` (recommended option first). If you can't reach the user, stop, leave the status `In Progress`, and end your reply with the question and your recommendation.
- If you notice something worth fixing that's outside the spec, mention it in your report; don't fix it.

### Safety rules for this repo

- **`back-end/.env` points at the production database and holds real Cloudinary credentials.** Never source it, and never run anything that does: `make run`, `make dev`, `make migrate`, `make seed-user`, `make fly-*`, the `run-all` skill, or `preview_start` with the `back-end` config from `.claude/launch.json`. To run the API locally, from `back-end/`:
  ```bash
  make db-up
  export DATABASE_URL="postgres://boraquest:boraquest@localhost:5432/boraquest?sslmode=disable" JWT_SECRET=boraquest-dev-secret CLOUDINARY_CLOUD_NAME=local CLOUDINARY_API_KEY=local CLOUDINARY_API_SECRET=local
  go run ./cmd/migrate && go run ./cmd/webapp
  ```
  The dummy Cloudinary values let the API boot (`cmd/webapp/main.go` refuses to start without them). Photo uploads will fail locally, so mark upload criteria manual-only instead of reaching for real keys. If you need it running across turns, start it with `run_in_background`. The `front-end` preview config is safe to use.
- Never run any `fly`/`flyctl`, `vercel` or Supabase command against a real environment. Those steps go in the runbook for a human.
- Migrations must be backward-compatible with the running release (expand/contract), as `CLAUDE.md` requires.
- Front-end commands need Node 22: prefix with `PATH=/opt/homebrew/opt/node@22/bin:$PATH`.

## Verifying

Before you report back, all of these must pass. Show the real output; never claim a pass you didn't see.

- Back-end touched: from `back-end/`, `make lint` and `make cover` (Docker must be running; 100% coverage of `./internal/...` and `./cmd/webapp/routes/...`).
- Front-end touched: from `front-end/`, `npm run lint` and `npm run build`.
- Walk the spec's **Acceptance criteria** one by one and mark each met, not met, or manual-only. For UI changes, check them in the browser preview at a 390px viewport when you can (front-end via `preview_start` `front-end`, back-end only via the local recipe in Safety rules), and attach a screenshot as proof.
- Run the `code-review` skill on your diff and fix every confirmed bug before reporting.

## Runbook

After the implementation is verified, decide whether shipping it needs **any action besides merging the PR**. Examples:

- New or changed env vars or secrets (Fly secrets via `make fly-secrets`, Vercel Production env, GitHub Actions secrets).
- Third-party setup: creating keys (e.g. VAPID), dashboards, webhooks, cron schedules, buckets or folders.
- Data work that isn't an automatic migration: backfills, seeding a real user, cleanups.
- Manual verification on production, or a required deploy order (back-end before front-end, or the other way round).
- Anything the spec lists under Prerequisites, Rollout or "set by hand".

Merging alone triggers CI and both deploys and Fly runs migrations automatically, so none of that counts as an action.

If at least one action is needed, write `runbooks/<name>.md`:

```markdown
# Runbook: <spec title>

Spec: [todo/<name>.md](../todo/<name>.md)
PR: <filled in during the PR step>
Last updated: <YYYY-MM-DD>

## When to run
Before merge / after merge / after the back-end deploy is live, etc.

## Steps
1. Exact command or dashboard path, where to run it, and what to type. Never write real secret values; name the variable and where its value comes from.
2. ...

## Verify
How to confirm each step worked (command plus expected output, URL plus what you should see).

## Rollback
How to undo it if something goes wrong.

## When finished
Change this runbook's row in `runbooks/README.md` to `Done`, and the spec's status from `Done (runbook pending)` to `Done` in `todo/<name>.md` and `todo/README.md`.
```

Then add a line to `runbooks/README.md` (create it as a table index like `todo/README.md`: Runbook | Spec | Status (`Pending`/`Done`) | Last updated), and add `Runbook: [runbooks/<name>.md](../runbooks/<name>.md)` under the spec's `Depends on:` line.

If nothing beyond merging is needed, don't create a runbook. Say so explicitly in your report.

## Report and wait for approval

End the implementation phase with a short report:

- What changed, grouped by area, with links to the key files.
- Verification results (commands and outcomes), the acceptance criteria checklist, and the code-review findings you fixed.
- Any spec drift you adapted to, and anything you noticed but left out of scope.
- The runbook (path and a one-line summary of each step), or "No runbook needed: merging is enough."
- Then ask: "Approve and create the PR?"

Leave the status at `In Progress` and **stop**. Don't commit, push or open a PR until the user approves and asks for the PR. If they ask for changes instead, make them, re-run verification, and report again.

## PR step (only after approval and an explicit request to create the PR)

1. Set the spec to `Done` in both the spec file and `todo/README.md` (bump Last updated). If a runbook exists, use `Done (runbook pending)` instead, so it's visible that a human still has steps to run.
2. Commit everything on the branch with a clear message describing the feature. End it with the commit attribution lines from the system reminder; if you weren't given any, copy the `Co-Authored-By:` trailer format from the most recent Claude-authored commit in `git log`.
3. Push the branch and open the PR with `gh pr create` against `main`. The description has: a summary, a link to the spec, the test results, and a **Before/after merge** section that links the runbook and lists its steps (or says "No action needed besides merging").
4. If a runbook exists, fill its `PR:` line with the PR URL, commit and push that.
5. Right after opening it, run the `code-review` skill on the PR. Fix confirmed bugs, push, and report what you fixed.
6. Reply with the PR link, the final status, and, if there's a runbook, a reminder of who has to run it and when.

## Closing a runbook

If the user tells you a runbook has been run (e.g. "the runbook for X is done"), follow that runbook's **When finished** section, then commit and push or open a small PR as they prefer.
