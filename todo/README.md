# TODO specs index

| Spec | Status | Last updated | Depends on |
| --- | --- | --- | --- |
| [CI/CD: deploy front-end to Vercel, back-end to Fly.io, migrate Supabase](./ci-cd-deploy.md) | Done (code; setup pending) | 2026-10-12 | none |
| [Quest entries: mark quests done, log slips, and real scores on Hoje and Semana](./quest-entries.md) | Ready | 2026-10-09 | none |
| [Scoreboard: real weekly and monthly standings on Guilda, with tasks completed](./scoreboard.md) | Ready | 2026-10-09 | quest-entries |
| [Profile personalization: pick your avatar](./profile-avatar.md) | Done | 2026-10-09 | none |
| [Prize definitions: real weekly and monthly prizes on Regras and Guilda](./prizes.md) | Ready | 2026-10-09 | none |
| [Quest photo uploads: optional proof photo on Hoje, guild-visible thumbnail on Semana](./quest-photo-uploads.md) | Done | 2026-10-10 | quest-entries |
| [Image upload flow: gallery picks for quest photos, avatar photos, and split Cloudinary folders](./image-upload-flow.md) | Draft | 2026-10-12 | quest-photo-uploads, profile-avatar |
| [Daily reminder: a 20h Web Push nudge to mark today's quests](./daily-reminder-push.md) | Done (runbook pending) | 2026-10-13 | none (builds on shipped quest-entries; deploys via ci-cd-deploy) |
| [Disable pinch zoom on mobile: lock page scale so the app feels native](./disable-pinch-zoom.md) | Ready | 2026-10-13 | none |
