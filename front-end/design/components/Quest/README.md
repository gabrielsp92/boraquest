One quest in a list. The same row serves the Today checklist and the Rules list; the round mark on the left says which.

- Default: empty circle with a `line-strong` border, waiting to be checked. The whole row is the tap target.
- `.bq-quest--done`: circle filled `gain` with a check. Keep the name fully legible; no strikethrough.
- `.bq-quest--gain`: Rules list, positive quest. `gain-soft` disc with a plus.
- `.bq-quest--loss`: negative quest or a logged slip. Dashed `loss` border, no card fill, solid `loss` disc with a minus. `--selected` makes the border solid on `loss-soft` when picking one in the slip sheet.

You provide the name (a short imperative, up to about 24 characters), the points, and optionally one meta line: the frequency ("Diária" / "Semanal") in Rules, or who logged it ("anotado por Beto") for a slip. A logged slip always shows who logged it.
