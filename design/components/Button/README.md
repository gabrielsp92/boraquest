The one thing to do on a screen. Use a single `.bq-btn` (accent) per screen or sheet; everything else is `--plain`, `--small` or `--ghost`.

- `.bq-btn`: primary, `accent` fill with `on-accent` text and the chunky `shadow-button` edge. 56px tall.
- `.bq-btn--plain`: equal-weight alternative next to a primary, as in "Adicionar foto" / "Pular". Same height and width as the primary so neither choice looks like the wrong one.
- `.bq-btn--ink`: the primary on an `accent` ground (winner screen), where an accent button would vanish.
- `.bq-btn--small`: 48px, for an action inside a row ("Pedir auditoria").
- `.bq-btn--ghost`: dismissals ("Agora não").
- `.bq-btn--block`: full width; use it for every button inside a sheet.
- `.bq-iconbtn`: 48px round icon button; always give it an `aria-label`.

You provide the label (two or three words, sentence case, a verb first) and optionally one leading icon. Never two primaries side by side, never an icon-only primary.
