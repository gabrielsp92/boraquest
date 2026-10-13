# Runbook: Daily reminder: a 20h Web Push nudge to mark today's quests

Spec: [todo/daily-reminder-push.md](../todo/daily-reminder-push.md)
PR: (filled in during the PR step)
Last updated: 2026-10-13

## Status
Steps 1–3 done before merge (2026-10-13): the four secrets are imported on Fly
(`boraquest-api`), and GitHub `production` has `REMINDER_CRON_SECRET` and
`BACKEND_PUBLIC_URL=https://boraquest-api.fly.dev`. Step 4 (post-deploy smoke test and
iPhone check) is still pending.

## When to run
Steps 1–3 **before merging** the PR. The new API release refuses to start without
`VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY`, `VAPID_SUBJECT` and `REMINDER_CRON_SECRET`
(`log.Fatal` in `back-end/cmd/webapp/main.go`), so the Fly secrets must exist before
`deploy-backend.yml` ships the new image. Step 4 after both deploys are live.

The migration `0011_push_reminders.sql` is purely additive and runs automatically
(Fly `release_command`); nothing to do for it.

## Steps
1. **Generate the keys once (on your machine, never in CI).**
   - VAPID pair: `npx web-push generate-vapid-keys` (prints a Public Key and a Private Key, base64url).
   - Cron secret: `openssl rand -base64 32`.
   - Subject: `mailto:<a real maintainer email>`, written exactly like that: no space after
     `mailto:`, no angle brackets, not `localhost` or `.local`. Apple rejects anything else
     with 403 `BadJwtToken`.
   - Keep the VAPID pair forever: every browser subscription is tied to the public key.
     Rotating it silently breaks every member's reminder until they turn the switch off and on.
2. **Fly secrets.** Add the four values to `back-end/.env` (names and format in
   `back-end/.env.example`), then from `back-end/`:
   - `make fly-secrets-dry-run`: should list the 9 keys, including `VAPID_PUBLIC_KEY`,
     `VAPID_PRIVATE_KEY`, `VAPID_SUBJECT`, `REMINDER_CRON_SECRET`.
   - `make fly-secrets`: imports all 9 secrets into the `boraquest-api` app (org `boraquest`,
     checked by `scripts/fly-app.sh`). It restarts running machines on the *current*
     release, which ignores the new keys, so this is safe before merge.
3. **GitHub, `production` environment** (repo → Settings → Environments → production):
   - Secret `REMINDER_CRON_SECRET` = the same value as in step 1/2.
   - Variable `BACKEND_PUBLIC_URL` = the Fly app's public https URL with no trailing slash,
     e.g. `https://boraquest-api.fly.dev`.
4. **After merge, once both deploys are green**, smoke test from the Actions tab:
   "Daily reminder" → Run workflow → `force: true`, `userId: <your user id>`.
   (Do it once you have turned the switch on in Guilda on your phone and have an
   unchecked daily quest; see the spec's manual acceptance criteria for the iPhone walk-through.)

## Verify
- Step 2: `flyctl secrets list -a boraquest-api` (via the Makefile's org check if you prefer)
  shows the four new names. After the deploy, `curl -s https://<fly-app>/api/v1/push/public-key`
  returns `{"publicKey":"<the public key from step 1>"}`.
- Step 3/4: the manual workflow run is green and its log ends with JSON like
  `{"day":"2026-10-13","usersEligible":1,"usersNotified":1,...,"devicesSent":1,...}`.
  `devicesSent: 0` with `usersEligible: 0` means you have no subscription or no unchecked
  daily quest. A 401 means the GitHub secret and the Fly secret differ. `curl: (6)`/(3) means
  `BACKEND_PUBLIC_URL` is wrong or empty.
- The phone shows "Bora fechar o dia?" / "Marca as quests que você fez hoje."; tapping it opens Hoje.
- The next evening (around 20:05 São Paulo), the scheduled run appears in the Actions tab
  without anyone triggering it. GitHub can delay it; anything before 23:00 still sends.

## Rollback
- **Bad release:** revert the PR on `main`. The old release ignores the new secrets and
  tables. The migration is additive, so no down-migration is needed; leave the tables.
- **Stop reminders only:** Actions → "Daily reminder" → ⋯ → Disable workflow. Nothing else
  sends pushes.
- **Leaked cron secret:** generate a new one, update it in `back-end/.env` + `make fly-secrets`
  and in the GitHub `production` secret.
- Do **not** unset the Fly VAPID/cron secrets while the new release is deployed: the API
  would refuse to start on its next machine boot.
