Boraquest is a family habit game. Family members (the guild) complete Quests to earn points, share a scoreboard, and each week the top scorer wins a prize. It is a mobile-first, installable PWA with four tabs: Hoje, Semana, Guilda, Regras. Build it as a game a child or a grandparent can play without instructions, never as a productivity tool.

## Principles

- One primary action per screen. Use one `.bq-btn` (accent); every other action is plain, small or ghost.
- A few things per screen. One score, one list, one button. If a screen needs a second list, it is a second screen or a sheet.
- Big and round. Nothing pressable is smaller than `tap-min` (48px); primary buttons and inputs are `tap-primary` (56px). Corners come from the radius scale, never square.
- Friendly competition. Celebrate gains loudly (confetti, crown); state losses plainly and without blame.

## Voice

All interface copy is Brazilian Portuguese, informal, addressed as "você". Sentence case everywhere: "Nova quest", not "Nova Quest". Keep the game words: Quest, guilda, pontos, prêmio, auditoria. A negative behavior is a "deslize" (a slip), never "erro", "falha" or "punição".

- Greet by first name: "Oi, Lia!"
- Buttons start with a verb and stay under four words: "Adicionar foto", "Pular", "Anotar deslize", "Pedir auditoria", "Enviar provas", "Aprovar".
- Celebrate in a few words with one exclamation mark at most: "Quest concluída!", "Vó Nena venceu!"
- Explain a rule in one sentence, when it matters: "Até você aprovar, ele não pode vencer."
- A logged slip always names who logged it: "anotado por Beto".
- Write points with their sign and the real minus: `+10`, `−5`.
- No emoji in the interface; the avatars, the crown and the confetti carry the fun.

## Color

One theme, light. The ground is `ground` (warm cream) with white `surface` cards on it. All text is `ink`, or `ink-muted` for secondary lines.

- `accent` is the one bright color. Spend it on the primary button, the score card, the leader and the winner screen. Text on it is always `on-accent` (dark), never white.
- `accent-soft` marks the selected tab, the leader's row and banners.
- Positive quests use `gain`: text on a `gain-soft` pill. Negative quests use `loss`: white text on a solid `loss` pill and a dashed `loss` border. The pair differs by sign, by fill and by lightness, so never tell them apart with color alone.
- The prize card and the "Auditoria pendente" badge are solid `ink`. Keep `ink` fills to those two.
- `avatar-*` colors belong to avatars and confetti. Do not use them for buttons, text or status.
- `line` is decoration; any border that marks a control uses `line-strong`.

## Type

Two rounded families from Google Fonts, loaded with weights 700 and 800 only. Baloo 2 (`--font-display`) for numbers, titles and headings: `display`, `title`, `heading`, `points`. Nunito (`--font-body`) for everything read as a sentence: `body`, `label`, `caption`. Nothing is lighter than 700 and nothing is smaller than `caption` (13px). One `display` number per screen.

## Layout

Design at `screen-width` (390px), one column, `space-4` gutters, stretching from 320px to 480px; center the column on anything wider. A main screen is a title, a score or prize card, one list, and the bottom navigation (`nav-height`, including the home-indicator safe area). Rows in a list sit `space-2` apart; sections `space-4` apart. Short decisions happen in a bottom sheet; full tasks (audit, winner) take the whole screen and hide the navigation.

## Shape and depth

Rows, cards and inputs use `radius-md`; the score card, prize card and sheets `radius-lg`; buttons, pills and badges `radius-pill`. Depth is a flat edge, not a blur: `shadow-card` under cards, `shadow-button` under the primary button, which sinks 4px when pressed. Only a sheet floats, with `shadow-sheet` over `scrim`. The focus ring is 3px solid `ink`, offset 2px.

## Motion

Two celebrations only: confetti when a quest is checked and when a winner is crowned. Pieces drop in once, in under a second, and never loop. Sheets slide up in about 200ms. Honor reduced motion by showing the end state.

## Icons and avatars

Icons are inline SVG line icons on a 24px grid: 2.5px stroke, round caps and joins, `currentColor`. The set in use: check, plus, minus, close, camera, image, trophy, gift, shield, crown, pencil, and the four tab icons (sun, calendar, users, scroll). Draw any addition the same way. Avatars are the six files in `assets/Avatars/`. There is no logo yet: set the name "Boraquest" in Baloo 2 at weight 800.

## Screens

The cards in the Telas group are the reference for every screen and state; build a screen by copying the closest one.

- `TelaHoje`: the checklist, the optional photo sheet after a check, and the slip sheet.
- `TelaSemana`: the week day by day, and past weeks with their winners.
- `TelaGuilda`: weekly scoreboard, monthly standings, and asking for an audit.
- `FluxoAuditoria`: the audited member sending proof, and the requester approving or rejecting.
- `TelaRegras`: quests and prizes, and the three-field new-quest sheet.
- `EstadosVazios`: Hoje, Semana and Regras for a brand-new guild.
- `FimDeSemana`: the winner celebration, and the hold state when the leader has an audit pending.

## Rules of the game the interface must keep

- A photo after checking a quest is optional. "Adicionar foto" and "Pular" are the same size.
- Anyone can log a slip for anyone, and the entry shows who logged it.
- A member with "Auditoria pendente" keeps their points and position but cannot be declared winner until the requester approves.
- A new quest has three fields only: name, frequency (Diária or Semanal), and points with a sign.

## Installing as an app

Use `ground` as the manifest `background_color` and `accent` as `theme_color`, `display: standalone`, portrait. Pad the bottom navigation and sheets with `env(safe-area-inset-bottom)`.
