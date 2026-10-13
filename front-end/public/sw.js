// Boraquest service worker: shows the daily reminder push and opens Hoje on tap.
// No fetch handler and no caching (offline support is a separate follow-up).
// Served at /sw.js with root scope; headers are set in next.config.ts.

const FALLBACK = {
  title: "Bora fechar o dia?",
  body: "Marca as quests que você fez hoje.",
  url: "/hoje",
  tag: "daily-reminder",
};

// Take control of open pages right away, so a notification tap can navigate them.
self.addEventListener("install", () => self.skipWaiting());
self.addEventListener("activate", (event) => event.waitUntil(self.clients.claim()));

self.addEventListener("push", (event) => {
  let data = {};
  try {
    data = event.data ? event.data.json() : {};
  } catch {
    data = {};
  }
  const n = { ...FALLBACK, ...data };
  // Always show a notification: Safari revokes push for sites that receive silent pushes.
  event.waitUntil(
    self.registration.showNotification(n.title, {
      body: n.body,
      icon: "/icons/icon-192.png",
      tag: n.tag,
      data: { url: n.url },
    }),
  );
});

self.addEventListener("notificationclick", (event) => {
  event.notification.close();
  const url = (event.notification.data && event.notification.data.url) || FALLBACK.url;
  event.waitUntil(
    (async () => {
      const windows = await self.clients.matchAll({ type: "window", includeUncontrolled: true });
      const client = windows[0];
      if (client) {
        await client.focus();
        // navigate() only works on a page this worker controls; otherwise open /hoje.
        const navigated = await client.navigate(url).then(Boolean, () => false);
        if (navigated) return;
      }
      await self.clients.openWindow(url);
    })(),
  );
});
