Form parts: a labelled text input, a two-option segmented control, and a stepper for points.

- `.bq-field` wraps a `.bq-label` and one control.
- `.bq-input`: 56px, `line-strong` outline. Prefill or use a real example as placeholder ("Beber 2 L de água").
- `.bq-seg`: two options only; mark the chosen one with `aria-pressed="true"`.
- `.bq-stepper`: two round buttons around the value, in steps of 5.

You provide the label (one or two words) and the value. Never more than three fields in a sheet, and no helper text under a field; if it needs explaining, rename the label.
