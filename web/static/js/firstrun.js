// Shown once per person: a short welcome the first time they open
// RecipeBank, and "What's new" after an update. Both reopen from Me.
import { get, put } from "./api.js";
import { $$, esc, canManage, sheet } from "./ui.js";
import { renderMarkdown, changelogSections } from "./markdown.js";

const PAGES = [
  (state) => `<h2 class="text-xl font-bold">👋 Welcome to RecipeBank</h2>
    <p>The family's recipes in one place, each one checked against everyone's allergies, diets and dislikes.</p>
    <div class="flex flex-wrap gap-1"><span class="chip-ok">✓ OK for everyone</span>
      <span class="chip-no">✕ Not for one person: contains milk</span><span class="chip-unsure">⚠ Not sure: check the label</span></div>
    <p class="text-sm text-slate-400"><b>Not sure</b> means RecipeBank can't tell from the words, so check the package.
      It never guesses that something is safe.</p>`,
  (state) => `<h2 class="text-xl font-bold">🧭 Finding your way</h2>
    <ul class="space-y-2">
      <li>🌙 <b>Tonight:</b> what's for dinner and who's eating.</li>
      <li>🍲 <b>Kitchen:</b> every recipe. Search in plain words ("something quick with chicken").</li>
      <li>🥕 <b>What can I make?</b> Recipes from what's in the house.</li>
      <li>🛒 <b>Shopping:</b> one list for everyone, even in a store without signal.</li>
      <li>▶ <b>Start cooking:</b> one step at a time, big text, timers.</li>
      ${canManage(state.user) ? `<li>➕ <b>Add a recipe</b> from a link, a photo of a card, or pasted text.</li>` : ""}
    </ul>
    <p class="text-sm text-slate-400">On a phone, the rest is under <b>☰ More</b>.</p>`,
  () => `<h2 class="text-xl font-bold">📱 Put it on your phone</h2>
    <ul class="space-y-2">
      <li><b>iPhone:</b> in Safari, tap <b>Share</b>, then <b>Add to Home Screen</b>.</li>
      <li><b>Android:</b> in Chrome, tap <b>⋮</b>, then <b>Install app</b> (or <b>Add to Home screen</b>).</li>
    </ul>
    <p class="text-sm text-slate-400">Phones only install apps from an address starting with <b>https://</b>. If you don't
      see the option, ask whoever set up RecipeBank for that address.</p>
    <p class="text-sm text-slate-400">You can see this again any time under <b>Me</b>.</p>`,
];

// welcome shows the three pages.
export function welcome(state) {
  let page = 0;
  const draw = () => {
    const d = sheet(`<div class="space-y-4">
      <div class="flex items-center"><span class="text-xs text-slate-500">${page + 1} of ${PAGES.length}</span>
        <button type="button" data-close class="btn-ghost ml-auto" aria-label="Close">✕</button></div>
      <div class="space-y-3">${PAGES[page](state)}</div>
      <div class="flex flex-wrap gap-2">
        ${page ? `<button type="button" data-step="-1" class="btn-secondary">‹ Back</button>` : ""}
        ${page < PAGES.length - 1 ? `<button type="button" data-step="1" class="btn-primary">Next ›</button>`
          : `<button type="button" data-close class="btn-primary">Let's go</button>`}</div></div>`);
    $$("[data-step]", d).forEach((b) => (b.onclick = () => { page += Number(b.dataset.step); draw(); }));
  };
  draw();
}

// cores turns "0.3.0" (or "v0.3.0-2-gabc") into [0, 3, 0], or null.
const cores = (v) => {
  const m = String(v || "").match(/^v?(\d+)\.(\d+)\.(\d+)/);
  return m ? m.slice(1).map(Number) : null;
};
const newer = (a, b) => {
  const x = cores(a), y = cores(b);
  if (!x || !y) return false;
  for (let i = 0; i < 3; i++) if (x[i] !== y[i]) return x[i] > y[i];
  return false;
};

async function whatsNew(since) {
  const data = await get("/api/updates");
  const notes = changelogSections(data.changelog).filter((s) => newer(s.version, since));
  if (!notes.length) return;
  sheet(`<div class="space-y-3">
    <div class="flex items-center"><h2 class="text-xl font-bold">🆕 What's new in RecipeBank</h2>
      <button type="button" data-close class="btn-ghost ml-auto" aria-label="Close">✕</button></div>
    ${notes.map((s) => `<section><h3 class="font-semibold">${esc(s.version)}</h3>
      <div class="break-words text-sm leading-relaxed text-slate-300">${renderMarkdown(s.body)}</div></section>`).join("")}
    <div class="flex flex-wrap gap-2"><button type="button" data-close class="btn-primary">Got it</button>
      <a href="#/whatsnew" data-close class="btn-ghost">All versions</a></div></div>`);
}

// firstRun opens the welcome or "What's new" if this person hasn't seen it,
// and counts it as seen once it's shown (both reopen from Me).
export async function firstRun(state) {
  const u = state.user;
  const markSeen = () => put("/api/me/seen", {}).then((r) => (u.seen_version = r.seen_version)).catch(() => {});
  if (!u.seen_version) {
    welcome(state);
    return markSeen();
  }
  if (newer(u.version, u.seen_version)) {
    await whatsNew(u.seen_version).catch(() => {});
    return markSeen();
  }
}
