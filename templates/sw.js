// Service worker: aplicația instalabilă (PWA) și citirea offline.
//
// Generat de site.go; VERSION se schimbă la fiecare modificare a site-ului,
// iar cache-urile vechi sunt șterse la activare.
//
// - paginile HTML: întâi rețeaua (conținut proaspăt), apoi cache-ul;
// - restul (CSS, JS, iconițe, indexul de căutare): cache, actualizat în fundal;
// - mesajul {type: "download", lang} salvează toate lecțiile unei limbi.
"use strict";

const VERSION = "{{.Version}}";
const CACHE = `go-features-${VERSION}`;
const BASE = "{{.Base}}";
const PRECACHE = {{.Precache}};
const PAGES = {{.Pages}};

self.addEventListener("install", (ev) => {
  ev.waitUntil(
    caches
      .open(CACHE)
      .then((c) => c.addAll(PRECACHE))
      .then(() => self.skipWaiting()),
  );
});

self.addEventListener("activate", (ev) => {
  ev.waitUntil(
    caches
      .keys()
      .then((keys) => Promise.all(keys.filter((k) => k.startsWith("go-features-") && k !== CACHE).map((k) => caches.delete(k))))
      .then(() => self.clients.claim()),
  );
});

function isPage(req) {
  return req.mode === "navigate" || (req.headers.get("accept") || "").includes("text/html");
}

async function networkFirst(req) {
  const cache = await caches.open(CACHE);
  try {
    const resp = await fetch(req);
    if (resp.ok) cache.put(req, resp.clone());
    return resp;
  } catch (err) {
    const hit = await cache.match(req, { ignoreSearch: true });
    if (hit) return hit;
    // pagină nesalvată: pagina de start a limbii, dacă există
    const home = new URL(req.url).pathname.startsWith(BASE + "en/") ? BASE + "en/" : BASE;
    return (await cache.match(home)) || Response.error();
  }
}

async function staleWhileRevalidate(req) {
  const cache = await caches.open(CACHE);
  const hit = await cache.match(req);
  const fresh = fetch(req)
    .then((resp) => {
      if (resp.ok) cache.put(req, resp.clone());
      return resp;
    })
    .catch(() => hit);
  return hit || fresh;
}

self.addEventListener("fetch", (ev) => {
  const req = ev.request;
  const url = new URL(req.url);
  if (req.method !== "GET" || url.origin !== location.origin || !url.pathname.startsWith(BASE)) return;
  ev.respondWith(isPage(req) ? networkFirst(req) : staleWhileRevalidate(req));
});

// Descărcarea tuturor lecțiilor unei limbi, cu progres trimis paginii.
self.addEventListener("message", (ev) => {
  const msg = ev.data || {};
  if (msg.type !== "download" || !PAGES[msg.lang]) return;
  const client = ev.source;
  ev.waitUntil(
    (async () => {
      const cache = await caches.open(CACHE);
      const urls = PAGES[msg.lang];
      let done = 0;
      let failed = 0;
      for (let i = 0; i < urls.length; i += 6) {
        await Promise.all(
          urls.slice(i, i + 6).map(async (u) => {
            try {
              if (!(await cache.match(u))) {
                const resp = await fetch(u);
                if (!resp.ok) throw new Error(resp.status);
                await cache.put(u, resp);
              }
            } catch {
              failed++;
            }
            done++;
          }),
        );
        client.postMessage({ type: "progress", done, total: urls.length });
      }
      client.postMessage({ type: "downloaded", total: urls.length, failed });
    })(),
  );
});
