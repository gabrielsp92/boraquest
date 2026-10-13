# Disable pinch zoom on mobile

Status: Done
Last updated: 2026-10-13
Depends on: none (front-end only, no back-end or API change)

## Summary
On a phone, pinching the app zooms the whole page, including the sticky bottom nav, so Bora stops looking like a native app. This spec locks the page at 100% scale on mobile: no pinch zoom, no double-tap zoom, and no automatic zoom when a text field gets focus. It's a small front-end change in `app/layout.tsx` and `app/globals.css`, plus one tiny client component for older iOS Safari.

## Problem
`front-end/app/layout.tsx:27-32` sets `width: device-width` and `initialScale: 1`, but nothing limits scale. Pinching scales everything, and `.bq-nav` (`globals.css:179`, `position: sticky; bottom: 0`) grows or shrinks with it. A native app never does that, and the zoomed state is easy to get stuck in.

## Goals
- Pinch zoom does nothing on Android Chrome, iOS Safari (tab) and the installed home-screen app (iOS and Android).
- Double-tap does not zoom.
- Focusing any text input does not zoom the page on iOS.
- Normal vertical/horizontal scrolling, taps, the bottom sheets and the stepper all behave exactly as today.

## Non-goals
- Desktop browser zoom (Ctrl/Cmd +/-, trackpad pinch on macOS). Desktop isn't the target and browser-level zoom should stay available there.
- The OS-level text-size settings (iOS Dynamic Type / Android font size). Untouched; they're the accessibility route that remains.
- Any layout or design change. If some text turns out to be too small to read without zooming, that's a separate design fix, not this spec.
- Offline / service worker work (see `daily-reminder-push.md`, `front-end/README.md`).

## Current state
- `front-end/app/layout.tsx:27-32`:
  ```ts
  export const viewport: Viewport = {
    themeColor: "#fff8ec",
    width: "device-width",
    initialScale: 1,
    viewportFit: "cover",
  };
  ```
  `layout.tsx` is a Server Component (no `"use client"`), which is required for the `viewport` export (`node_modules/next/dist/docs/01-app/03-api-reference/04-functions/generate-viewport.md`).
- `front-end/app/globals.css:93-94`: `html` and `body` base rules. No `touch-action` anywhere in the app.
- Text inputs: every typed input uses `.bq-input` (`globals.css:191`, `font: 700 17px/24px`), so all of them are already at least 16px: `entrar/page.tsx:35,38`, `regras/page.tsx:270,280,306`, `auditoria/page.tsx:69`. `hoje/page.tsx:308` is a hidden file input (no focus zoom). Body text is 17px (`globals.css:94`).
- No front-end test runner (`package.json` has only `lint`), so verification is manual.

## Business rules
1. The page scale is always 1 on mobile. The user can't change it with gestures.
2. Text inputs must use a font size of at least 16px. iOS Safari zooms in on focus below 16px, and with zoom locked the user couldn't zoom back out.

## Architecture & technical decisions

### 1. Viewport meta (Android, and installed web apps)
In `app/layout.tsx`, extend the existing export:
```ts
export const viewport: Viewport = {
  themeColor: "#fff8ec",
  width: "device-width",
  initialScale: 1,
  maximumScale: 1,
  userScalable: false,
  viewportFit: "cover",
};
```
This renders `maximum-scale=1, user-scalable=no` in the viewport meta tag. Android Chrome honors it. Since iOS 10, Safari **ignores** both in a normal tab (an accessibility decision by Apple), so steps 2 and 3 are needed for iPhone.

### 2. CSS `touch-action` (iOS Safari and everything else)
In `app/globals.css`, next to the existing `html` rule (`globals.css:93`):
```css
html { background: var(--ground); touch-action: pan-x pan-y; }
```
`pan-x pan-y` allows one-finger scrolling in both directions but no pinch zoom and no double-tap zoom. WebKit honors `touch-action` even where it ignores the viewport meta.
- Put it on `html` only. `touch-action` is applied as the intersection along the element chain, so children inherit the restriction without each needing it.
- Don't use `touch-action: none`: it would also kill scrolling.
- Don't use `manipulation` on `html` alone: it removes double-tap zoom but still allows pinch.

### 3. `gesturestart` guard (older iOS fallback)
Some iOS versions still let a pinch through despite `touch-action`. Add a small client component, `components/NoPinchZoom.tsx`:
```tsx
"use client";

import { useEffect } from "react";

// iOS Safari ignores user-scalable=no; cancelling its proprietary gesture events blocks pinch zoom.
export default function NoPinchZoom() {
  useEffect(() => {
    const block = (e: Event) => e.preventDefault();
    document.addEventListener("gesturestart", block);
    document.addEventListener("gesturechange", block);
    return () => {
      document.removeEventListener("gesturestart", block);
      document.removeEventListener("gesturechange", block);
    };
  }, []);
  return null;
}
```
Render it once inside `<body>` in `app/layout.tsx` (before `{children}`). `layout.tsx` itself stays a Server Component. `gesture*` events only exist in WebKit, so this is a no-op elsewhere.

### 4. Input font size (iOS focus zoom)
Already satisfied: `.bq-input` is 17px. The change is to keep it that way:
- Add a one-line comment above `.bq-input` in `globals.css`: inputs must stay ≥16px or iOS zooms on focus.
- Any new `<input>`, `<textarea>` or `<select>` must use `.bq-input` (or another ≥16px style).

## Edge cases
- **Already zoomed when the update ships:** a reload resets the scale to 1.
- **Horizontal scroll areas** (if any): `pan-x` keeps them working.
- **Bottom sheets / stepper / segmented control:** they use taps, not gestures, so they're unaffected. Check them anyway in the acceptance run.
- **Double-tap on buttons:** with double-tap zoom gone, taps also stop waiting ~300 ms for a possible second tap on older browsers, so the app feels slightly faster.
- **Text selection / long-press:** unaffected by `touch-action`.

## Accessibility, privacy & performance
- **Accessibility trade-off (accepted by the user):** people who rely on pinch-to-zoom lose it inside Bora. Mitigations: body text is already 17px, inputs 17px, tap targets 48px+ (`--tap-min`), and OS text-size settings still work. Native apps behave the same way. WCAG 1.4.4 (resize text) is a known trade-off here; revisit if any user asks for zoom back.
- No privacy impact. Performance: two event listeners on `document`, negligible.

## Decision log
- Small change, no full product discovery; spec written directly at the user's request (2026-10-13).
- Combine viewport meta + `touch-action` + `gesturestart` rather than only one: each covers a different browser (Android / modern WebKit / older WebKit).
- Lock zoom on all pages, including `/entrar` and `/auditoria`, for a consistent native feel.

## Implementation plan
1. `app/layout.tsx`: add `maximumScale: 1` and `userScalable: false` to `viewport`; render `<NoPinchZoom />` inside `<body>`.
2. `components/NoPinchZoom.tsx`: new file as above.
3. `app/globals.css`: add `touch-action: pan-x pan-y` to `html`; add the ≥16px comment above `.bq-input`.
4. `npm run lint` (Node 22: `/opt/homebrew/opt/node@22/bin`).
5. Manual run on real devices (acceptance criteria below).

## Acceptance criteria
Run on a real iPhone (Safari tab **and** home-screen app) and a real Android phone (Chrome), on production or the dev server over LAN:
1. View source / devtools: the viewport meta contains `maximum-scale=1` and `user-scalable=no`.
2. Pinching with two fingers on Hoje, Semana, Guilda and Regras doesn't change the page scale; the bottom nav keeps its size.
3. Double-tapping text or empty space doesn't zoom.
4. Tapping the email and password fields on `/entrar` doesn't zoom the page on iPhone; same for the Regras quest-name field and the Auditoria input.
5. Vertical scrolling works on every tab; bottom sheets open, scroll and close normally; the stepper and segmented controls respond to taps.
6. On desktop Chrome/Safari, Ctrl/Cmd +/- browser zoom still works.
7. `npm run lint` passes.

## Test plan
No front-end test runner exists, so testing is the manual device checklist above. In desktop Chrome's device emulation, check the meta tag and that the layout is unchanged; real pinch behavior can only be confirmed on hardware.

## Out of scope / follow-ups
- A Playwright/visual test setup for the front-end (would let item 1 of the acceptance criteria be automated).
- Revisit if users ask for zoom to read something: fix the specific text size instead of re-enabling zoom.

## Open questions
None.

## Notes for the implementing agent
- Read `front-end/AGENTS.md` first. The `viewport` export must stay in a Server Component (`layout.tsx`); don't add `"use client"` to it.
- The `Viewport` type in this Next.js version supports `maximumScale` and `userScalable`; check `generate-viewport.md` in `node_modules/next/dist/docs/` if the types complain.
- Front-end needs Node ≥ 20.9; the global `node` on this machine is 18.
