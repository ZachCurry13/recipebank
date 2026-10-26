// Cook mode → kitchen screen: type the code a TV or tablet shows at /tv, and
// each step goes to it, big (with its ingredients and timers). The screen
// sees only what's sent here. The code is kept for this browser tab.
import { post, del } from "./api.js";
import { $, esc, attempt, sheet, toast } from "./ui.js";

const KEY = "rb:tv";
const get = () => { try { return sessionStorage.getItem(KEY) || ""; } catch { return ""; } };
const set = (c) => { try { c ? sessionStorage.setItem(KEY, c) : sessionStorage.removeItem(KEY); } catch { /* private mode */ } };
let lastSent = "";

export const tvOn = () => Boolean(get());

// tvButton is cook mode's 📺 button.
export const tvButton = () => `<button id="cook-tv" class="btn-ghost px-2 ${tvOn() ? "text-emerald-300" : ""}" aria-pressed="${tvOn()}">📺 ${tvOn() ? "On the TV" : "TV"}</button>`;

// wireTV connects or disconnects when the button is tapped; again redraws cook mode.
export function wireTV(box, again) {
  const b = $("#cook-tv", box);
  if (!b) return;
  b.onclick = () => {
    if (tvOn()) {
      tvStop();
      return again();
    }
    const d = sheet(`<form class="space-y-3"><h2 class="text-lg font-semibold">📺 Show it on a TV or tablet</h2>
      <p class="text-sm text-slate-400">On the TV's web browser, open <b class="break-all">${esc(location.origin)}/tv</b>. It shows a code: type it here.</p>
      <input name="code" required maxlength="8" autocapitalize="characters" autocomplete="off" spellcheck="false" class="input text-center font-mono text-2xl uppercase tracking-widest" placeholder="CODE">
      <div class="flex gap-2"><button class="btn-primary">Connect</button><button type="button" data-close class="btn-ghost">Cancel</button></div></form>`);
    d.querySelector("form").onsubmit = (e) => {
      e.preventDefault();
      const code = e.target.code.value.trim().toUpperCase();
      attempt(async () => {
        await post("/api/tv/pair", { code });
        set(code);
        lastSent = "";
        d.close();
        toast("Connected: the TV follows your steps");
        again();
      });
    };
  };
}

// tvSend sends what cook mode shows now; same as last time, it sends nothing.
export function tvSend(view) {
  const code = get();
  if (!code) return;
  const sig = JSON.stringify([view.title, view.at, view.text, view.items, view.timers.map((t) => [t.label, t.done])]);
  if (sig === lastSent) return;
  lastSent = sig;
  post("/api/tv/show", { code, view }).catch((e) => {
    if (e.status === 404 || e.status === 409) { set(""); toast("The TV screen ended. Tap 📺 to connect again.", true); }
  });
}

// tvStop clears the screen and forgets it (cook mode closed, or 📺 tapped again).
export function tvStop() {
  const code = get();
  set("");
  lastSent = "";
  if (code) del(`/api/tv/${encodeURIComponent(code)}`).catch(() => {});
}
