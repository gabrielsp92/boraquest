One member on the scoreboard: position, avatar, name, points, and under the name either the "Pedir auditoria" button or a badge.

- `.bq-rank--leader`: first place. `accent-soft` fill, `accent` border, crown on the avatar.
- The signed-in member's own row shows the "Você" badge in place of the audit button.
- A member under audit shows `.bq-badge--audit` in place of the button. If that member is first, keep the highlight but never show the winner celebration for them until the audit is approved.
- Monthly standings: swap the second line for a `.bq-bar` whose inner width is the share of the top score.

You provide position, avatar, name and points. Sort by points, highest first.
