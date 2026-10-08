# Boraquest

A family habit game: the guild completes Quests to earn points, shares a scoreboard, and the top scorer wins the weekly prize. Mobile-first PWA built with Next.js (App Router), React and Tailwind CSS v4.

This is a front-end starter. All data is sample data in `lib/data.ts` and all state is local to each page; there is no backend, auth or persistence yet.

## Run it

```bash
npm install
npm run dev
```

Open http://localhost:3000 in a phone-sized window (390px wide).

## What is where

| Path | What |
| --- | --- |
| `app/(tabs)/hoje` `semana` `guilda` `regras` | The four tabs, sharing the bottom navigation in `app/(tabs)/layout.tsx` |
| `app/auditoria` | The audited member sends proof |
| `app/auditoria/revisar` | The requester approves or rejects |
| `app/fim-de-semana` | Winner celebration, or the hold screen while the leader has an audit pending |
| `app/globals.css` | Design tokens as a Tailwind theme, plus the `bq-*` component classes |
| `app/manifest.ts`, `public/icons` | PWA manifest and placeholder icons |
| `components/ui.tsx` | The React components (Button, Avatar, QuestRow, RankRow, Sheet, ...) |
| `components/Icon.tsx` | The line-icon set |
| `lib/data.ts` | Types and sample data: the first thing to replace |
| `design/` | The design system this was built from: `BRAND.md` (rules and voice), `tokens.json`, a README and an HTML reference per component, and every screen under `design/components/Tela*` |

## Styling

Tokens live once in `app/globals.css`, in two forms:

- as Tailwind theme values, so utilities work: `bg-accent`, `text-ink`, `text-ink-muted`, `bg-gain-soft`, `rounded-md` (20px), `rounded-lg` (28px), `rounded-pill`, `shadow-card`, `font-display`, `font-body`;
- under their design-system names (`var(--accent)`, `var(--space-4)`), which the `bq-*` component classes use.

The components are styled by those `bq-*` classes in Tailwind's `components` layer, ported unchanged from the design system so the app matches the design exactly. Tailwind utilities override them, so add utilities freely for layout and one-offs. Note that the theme replaces Tailwind's default `rounded-sm/md/lg` sizes with the Boraquest radii.

Read `design/BRAND.md` before adding a screen: one primary button per screen, nothing pressable under 48px, UI copy in Brazilian Portuguese.

## Not done yet

- Backend, accounts and a real guild. `me` in `lib/data.ts` is hardcoded.
- Photos are kept in memory only; proof thumbnails are placeholders.
- No service worker, so the app installs but does not work offline.
- The app icon is a placeholder letter; there is no logo.
- Nothing triggers the end-of-week or audit screens automatically; open their routes directly.
