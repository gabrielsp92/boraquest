---
name: stop-all
description: Stop the Boraquest back-end and front-end dev servers started by /run-all (kills whatever is listening on :8080 and :3000). Use when the user runs /stop-all or asks to stop/shut down/kill the dev servers.
---

# stop-all

Stops the dev servers by killing whatever is listening on their ports — this works regardless of how they were started (Air, `next dev`, or any child process tree), so it stays correct even if `/run-all`'s exact launch mechanism changes.

## Steps

1. For each port, find and kill the listening process(es):
   ```
   lsof -ti:8080 | xargs -r kill
   lsof -ti:3000 | xargs -r kill
   ```

2. Give them a moment, then confirm they're down:
   ```
   lsof -ti:8080
   lsof -ti:3000
   ```
   Both should return nothing. If a port still shows a PID after ~3 seconds, kill it with `-9`:
   ```
   lsof -ti:8080 | xargs -r kill -9
   lsof -ti:3000 | xargs -r kill -9
   ```

3. Report which ports were freed, or if both were already stopped.

## Notes

- This does **not** stop the local Postgres container (`make db-down` in `back-end/` does that separately) — Postgres is left running intentionally since it's cheap to keep up and other tools may depend on it.
- Use the companion `/run-all` skill to start both back up.
