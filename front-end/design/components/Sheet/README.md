A panel that slides up from the bottom for one short decision: the optional photo, logging a slip, adding a quest, asking for an audit.

Structure: `.bq-scrim` over the screen, then `.bq-sheet` with the grip, a title, at most one sentence, and the buttons stacked full width. One question per sheet. Tapping the scrim or dragging down closes it without saving.

You provide the title, the content and the buttons. When both answers are equally fine (photo or skip), pair a primary with a `--plain` button of the same size; when one is a dismissal, use `--ghost`.
