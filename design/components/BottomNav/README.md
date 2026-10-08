The four tabs, fixed to the bottom of every main screen: Hoje, Semana, Guilda, Regras, in that order.

Mark the current tab with `aria-current="page"`; it gets the `accent-soft` pill. Always show icon and label together. The bar is `nav-height` tall, which includes 20px for the phone's home indicator; in the PWA use `env(safe-area-inset-bottom)` for that part. Hide the bar on full-screen tasks (the audit flow, the winner screen). Never add a fifth tab or a floating button above it.
