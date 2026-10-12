The app icon: a smiling face in `ground` wearing the weekly winner's crown in `ink`, on `accent`.

- `icon.svg`: the source, with rounded corners (radius 112 on 512). Used for `public/icons/icon-192.png`, `icon-512.png` and `app/icon.png` (browser tab).
- `icon-fullbleed.svg`: the same art on a full square, for platforms that apply their own mask. Used for `public/icons/icon-maskable-512.png` (Android) and `app/apple-icon.png` (iPhone home screen, 180px). All of the art sits inside the central 80% circle, so Android's circle crop never cuts the crown.

To change the icon, edit the SVGs and re-export the PNGs at the same sizes. Do not add text or a second color to it.
