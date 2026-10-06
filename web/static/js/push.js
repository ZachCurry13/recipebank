// "📱 Phone notifications" on the Me page: turn Web Push on or off for this
// device (see internal/push) and pick what it's told about.
import { get, post } from "./api.js";
import { esc, attempt, toast } from "./ui.js";

const KINDS = [["timers", "⏱ Cook-mode timers, even with the screen off"], ["useby", "🥫 Food to use soon (each morning)"],
  ["low", "🧽 Something running low"], ["tonight", "🌙 Tonight's dinner (each afternoon)"]];

const isIOS = () => /iPhone|iPad|iPod/.test(navigator.userAgent);
const installed = () => window.matchMedia?.("(display-mode: standalone)").matches || navigator.standalone === true;

// Why this device can't get notifications, or "" if it can.
function blocker() {
  if (!window.isSecureContext) {
    return "Notifications need RecipeBank's secure <b>https://</b> address (an admin sets it up under <b>Admin → Remote access</b>). Open RecipeBank at that address, then come back here.";
  }
  if (isIOS() && !installed()) {
    return "On iPhone and iPad: tap <b>Share → Add to Home Screen</b>, open RecipeBank from the new icon, then turn notifications on here.";
  }
  if (!("serviceWorker" in navigator) || !("PushManager" in window) || !("Notification" in window)) {
    return "This browser can't show notifications from RecipeBank.";
  }
  if (Notification.permission === "denied") {
    return "Notifications are blocked for RecipeBank in this browser's settings. Allow them there, then reload this page.";
  }
  return "";
}

// The server's key as bytes, for PushManager.subscribe.
function keyBytes(b64) {
  const s = (b64 + "===".slice((b64.length + 3) % 4)).replace(/-/g, "+").replace(/_/g, "/");
  return Uint8Array.from(atob(s), (c) => c.charCodeAt(0));
}

function deviceName() {
  const ua = navigator.userAgent;
  const os = /iPhone/.test(ua) ? "iPhone" : /iPad/.test(ua) ? "iPad" : /Android/.test(ua) ? "Android"
    : /Windows/.test(ua) ? "Windows" : /Mac OS X/.test(ua) ? "Mac" : "Computer";
  const browser = /Edg\//.test(ua) ? "Edge" : /Firefox\//.test(ua) ? "Firefox" : /Chrome\//.test(ua) ? "Chrome"
    : /Safari\//.test(ua) ? "Safari" : "browser";
  return `${os} · ${browser}`;
}

// SHA-256 of the endpoint in hex: how GET /api/push names each device.
async function endpointKey(endpoint) {
  const sum = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(endpoint));
  return [...new Uint8Array(sum)].map((b) => b.toString(16).padStart(2, "0")).join("");
}

// serviceWorker.ready never settles if registration failed, so give up after a while.
const workerReady = () => Promise.race([navigator.serviceWorker.ready, new Promise((r) => setTimeout(() => r(null), 4000))]);

export async function renderPushCard(host, user) {
  const intro = "Get RecipeBank's reminders on this phone, even when the app is closed.";
  const why = blocker();
  const reg = why ? null : await workerReady();
  const sub = reg ? await reg.pushManager.getSubscription() : null;
  const info = (await attempt(() => get("/api/push"))) || { devices: [] };
  const key = sub ? await endpointKey(sub.endpoint) : "";
  const here = info.devices.find((d) => d.key === key); // this device, as the server knows it
  const mine = Boolean(sub && here);
  const others = info.devices.filter((d) => d !== here);
  host.innerHTML = `
    <h2 class="text-lg font-semibold">📱 Phone notifications</h2>
    <p class="text-sm text-slate-300">${intro}</p>
    ${why || !reg ? `<p class="rounded-lg bg-slate-800/60 p-3 text-sm text-amber-200">${why || "This browser isn't ready for notifications yet. Reload the page and try again."}</p>` : `
      <p class="text-sm">${mine ? "✅ On for this device." : "Off for this device."}</p>
      <div class="space-y-1">${KINDS.map(([k, label]) => `<label class="toggle"><input type="checkbox" data-want="${k}"
        ${!here || here.wants.split(",").includes(k) ? "checked" : ""}> ${label}</label>`).join("")}</div>
      <div class="flex flex-wrap gap-2">
        ${mine ? `<button data-push="test" class="btn-secondary">Send a test</button>
          <button data-push="off" class="btn-ghost">Turn off on this device</button>`
          : `<button data-push="on" class="btn-primary">Turn on for this device</button>`}
      </div>`}
    ${others.length ? `<p class="text-xs text-slate-500">Also on for ${others.length} other device${others.length === 1 ? "" : "s"}: ${esc(others.map((d) => d.device || "device").join(", "))}.</p>` : ""}`;
  if (why || !reg) return;

  const wants = () => [...host.querySelectorAll("[data-want]:checked")].map((c) => c.dataset.want);
  const subscribe = async () => {
    const opts = { userVisibleOnly: true, applicationServerKey: keyBytes(info.public_key) };
    try {
      return await reg.pushManager.subscribe(opts);
    } catch {
      // Subscribed earlier with a different server key: start over.
      await (await reg.pushManager.getSubscription())?.unsubscribe();
      return reg.pushManager.subscribe(opts);
    }
  };
  host.onchange = async (e) => {
    if (e.target.dataset.want && mine) {
      await attempt(() => post("/api/push/subscribe", { ...sub.toJSON(), wants: wants(), device: deviceName() }), "Saved");
    }
  };
  host.onclick = async (e) => {
    const act = e.target.closest("[data-push]")?.dataset.push;
    if (act === "on") {
      if ((await Notification.requestPermission()) !== "granted") return toast("Notifications weren't allowed on this device", true);
      const s = await attempt(subscribe);
      if (!s) return;
      const ok = await attempt(() => post("/api/push/subscribe", { ...s.toJSON(), wants: wants(), device: deviceName() }), "Notifications are on for this device");
      if (ok) renderPushCard(host, user);
    } else if (act === "off" && sub) {
      await attempt(() => post("/api/push/unsubscribe", { endpoint: sub.endpoint }));
      await sub.unsubscribe().catch(() => {});
      toast("Notifications are off for this device");
      renderPushCard(host, user);
    } else if (act === "test") {
      const r = await attempt(() => post("/api/push/test"));
      if (r) toast(r.sent ? `Test sent to ${r.sent} device${r.sent === 1 ? "" : "s"}. It should appear in a moment.` : `The test didn't go through: ${r.error || "unknown error"}`, !r.sent);
    }
  };
}
