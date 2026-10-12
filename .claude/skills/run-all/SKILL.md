---
name: run-all
description: Start the Boraquest back-end and front-end dev servers with hot reload (back-end on :8080 via Air, front-end on :3000 via Next.js). Use when the user runs /run-all or asks to start/boot up both dev servers.
---

# run-all

Starts both dev servers from `.claude/launch.json`, in the background, with hot reload, so edits are picked up live. Logs go to `.claude/run/backend.log` and `.claude/run/frontend.log`.

## Steps

1. Ensure the log directory exists: `mkdir -p .claude/run` (run from the repo root).

2. Check if each port is already in use before starting anything:
   ```
   lsof -ti:8080
   lsof -ti:3000
   ```
   If a port already has a listener, treat that server as already running — skip starting it and say so.

3. Start the back-end (only if :8080 is free). From the repo root:
   ```
   cd back-end && docker compose up -d --wait postgres
   cd back-end && set -a && { [ ! -f .env ] || . ./.env; } && set +a && JWT_SECRET=${JWT_SECRET:-boraquest-dev-secret} nohup $HOME/go/bin/air > ../.claude/run/backend.log 2>&1 &
   disown
   ```
   `back-end/.env` must define `CLOUDINARY_CLOUD_NAME`, `CLOUDINARY_API_KEY` and `CLOUDINARY_API_SECRET`, or the API refuses to start.
   Use the Bash tool for this (not `run_in_background`, since the process must outlive this turn/session — `nohup` + `disown` detaches it from the shell).

4. Start the front-end (only if :3000 is free). From the repo root:
   ```
   cd front-end && PATH=/opt/homebrew/opt/node@22/bin:$PATH nohup npm run dev > ../.claude/run/frontend.log 2>&1 &
   disown
   ```

5. Wait ~3-5 seconds, then verify both came up:
   ```
   curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/v1/health
   curl -s -o /dev/null -w "%{http_code}" http://localhost:3000
   ```
   If either fails, show the last ~20 lines of the relevant log file (`.claude/run/backend.log` or `.claude/run/frontend.log`) to help diagnose.

6. Report back concisely: which servers were (re)started vs. already running, and the two URLs (http://localhost:8080, http://localhost:3000). Mention that logs are being written to `.claude/run/*.log` if the user wants to tail them.

## Notes

- Air and `next dev` both hot-reload on file changes — no need to restart after this for normal edits.
- Do not run this from inside `front-end/` or `back-end/` — always `cd` from the repo root as shown, since the log paths are relative to it.
- Use the companion `/stop-all` skill to shut both down.
