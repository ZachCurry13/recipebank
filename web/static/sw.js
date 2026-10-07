// RecipeBank service worker: lets the installed app open offline and start
// fast. Versioned files (/v/<build>/…) never change, so they're served from
// the cache; pages are fetched fresh when online. The API is never cached.
const CACHE = "recipebank-shell-v7";

self.addEventListener("install", () => self.skipWaiting());

self.addEventListener("activate", (e) => {
  e.waitUntil(caches.keys().then((keys) => Promise.all(keys.filter((k) => k !== CACHE).map((k) => caches.delete(k)))));
  self.clients.claim();
});

self.addEventListener("fetch", (e) => {
  const url = new URL(e.request.url);
  if (e.request.method !== "GET" || url.origin !== location.origin || url.pathname.startsWith("/api/")) return;
  if (url.pathname.startsWith("/v/") || url.pathname.startsWith("/icons/")) {
    e.respondWith(caches.open(CACHE).then(async (c) => {
      const hit = await c.match(e.request);
      if (hit) return hit;
      const res = await fetch(e.request);
      if (res.ok) c.put(e.request, res.clone());
      return res;
    }));
    return;
  }
  if (e.request.mode === "navigate") {
    e.respondWith(fetch(e.request).then((res) => {
      const copy = res.clone();
      caches.open(CACHE).then((c) => c.put("/", copy));
      return res;
    }).catch(() => caches.match("/")));
  }
});

// Phone notifications (see internal/push): show what the server sent.
self.addEventListener("push", (event) => {
  let msg = {};
  try {
    msg = event.data ? event.data.json() : {};
  } catch {
    msg = { body: event.data ? event.data.text() : "" };
  }
  event.waitUntil(self.registration.showNotification(msg.title || "RecipeBank", {
    body: msg.body || "",
    icon: "/icons/icon-192.png",
    badge: "/icons/icon-192.png",
    tag: msg.tag || undefined,
    renotify: msg.tag === "timer",
    data: { url: msg.url || "/" },
  }));
});

// Tapping a notification opens (or focuses) RecipeBank on the right page.
self.addEventListener("notificationclick", (event) => {
  event.notification.close();
  const url = new URL(event.notification.data?.url || "/", self.location.origin).href;
  event.waitUntil(self.clients.matchAll({ type: "window", includeUncontrolled: true }).then((wins) => {
    const win = wins.find((w) => w.url.startsWith(self.location.origin));
    if (!win) return self.clients.openWindow(url);
    return win.focus().then((w) => (w && w.navigate ? w.navigate(url) : w)).catch(() => self.clients.openWindow(url));
  }));
});
