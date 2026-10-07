// 🔍 Search everything, from the top bar (or "/" on a computer): recipes,
// what's in the house, the list, the cookbooks, events, collections, people,
// and the settings pages, as you type.
import { get, qs } from "./api.js";
import { $, $$, esc, sheet } from "./ui.js";
import { state } from "./app.js";

// Settings and pages found by what people call them: [label, link, words, who].
const PAGES = [
  ["👪 Family: allergies, diets, dislikes", "#/family", "family people allergies allergy diet dislikes pets guests", "all"],
  ["🌿 Keep it simple, menu order", "#/profile", "simple menu order meals hide pages tabs units theme dark light password", "all"],
  ["🔔 Phone notifications", "#/profile", "notifications reminders push", "all"],
  ["🆕 What's new", "#/whatsnew", "new changelog version update", "all"],
  ["⚙️ AI settings and Ollama", "#/admin", "ai model ollama gemini openai claude gpu photos speed", "admin"],
  ["⚙️ Features on and off", "#/admin", "features turn off hide", "admin"],
  ["⚙️ Email, budget, nutrition, backup", "#/admin", "email smtp budget prices currency nutrition usda backup restore accounts users", "admin"],
];

export function openFind() {
  const d = sheet(`<div class="space-y-3">
    <div class="flex items-center gap-2"><input id="find-q" type="search" class="input min-w-0 flex-1" placeholder="Search everything"
        autocomplete="off" enterkeyhint="go" aria-label="Search everything">
      <button type="button" data-close class="btn-ghost" aria-label="Close">✕</button></div>
    <div id="find-out" class="space-y-4" aria-live="polite"></div></div>`);
  const q = $("#find-q", d), out = $("#find-out", d);
  let timer, asked = "";
  // A page matches when each word typed starts one of its words ("allerg" finds Family).
  const pages = (words) => PAGES.filter(([, , w, who]) => (who === "all" || state.user?.role === "admin") &&
    words.every((x) => w.split(" ").some((k) => k.startsWith(x))));
  const draw = async () => {
    const text = q.value.trim();
    asked = text;
    if (!text) return (out.innerHTML = `<p class="text-sm text-slate-500">Recipes, the pantry, the shopping list, cookbooks, events, people and settings.</p>`);
    const res = await get("/api/find" + qs({ q: text })).catch(() => ({ groups: [] }));
    if (asked !== text) return; // typed more since
    const words = text.toLowerCase().split(/\s+/);
    const found = pages(words);
    const groups = [...res.groups, ...(found.length ? [{ label: "Pages", hits: found.map(([title, link]) => ({ title, link })) }] : [])];
    out.innerHTML = groups.length ? groups.map((g) => `<section><h2 class="label">${esc(g.label)}</h2>
      <ul class="space-y-1">${g.hits.map((h) => `<li><a href="${esc(h.link)}" data-hit class="block rounded-lg px-2 py-1.5 hover:bg-slate-800">
        <span class="block break-words">${esc(h.title)}</span>${h.sub ? `<span class="block text-xs text-slate-400">${esc(h.sub)}</span>` : ""}</a></li>`).join("")}</ul></section>`).join("")
      : `<p class="text-sm text-slate-400">Nothing found for "${esc(text)}".</p>`;
    $$("[data-hit]", out).forEach((a) => (a.onclick = () => d.close()));
  };
  q.oninput = () => { clearTimeout(timer); timer = setTimeout(draw, 250); };
  q.onkeydown = (e) => {
    if (e.key !== "Enter") return;
    e.preventDefault();
    const first = $("[data-hit]", out);
    if (first) { location.hash = first.getAttribute("href"); d.close(); }
  };
  draw();
  setTimeout(() => q.focus(), 50);
}

let wired = false;

// wireFind adds the top bar's 🔍 and the "/" key.
export function wireFind() {
  const btn = $("#find-btn");
  if (btn) btn.onclick = openFind;
  if (wired) return;
  wired = true;
  document.addEventListener("keydown", (e) => {
    const typing = /^(INPUT|TEXTAREA|SELECT)$/.test(e.target.tagName) || e.target.isContentEditable;
    if (e.key === "/" && !typing && !e.ctrlKey && !e.metaKey && !$("#sheet").open) {
      e.preventDefault();
      openFind();
    }
  });
}
