// Shared DOM helpers. All dynamic text goes through esc() before innerHTML.

export function esc(v) {
  return String(v ?? "").replace(/[&<>"']/g, (c) => ({
    "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;",
  })[c]);
}

export const $ = (sel, root = document) => root.querySelector(sel);
export const $$ = (sel, root = document) => [...root.querySelectorAll(sel)];

let toastTimer;
// toast shows a short message. Errors stay until closed so they can't vanish
// before they're read.
export function toast(msg, isError = false) {
  const t = $("#toast");
  t.replaceChildren();
  const text = document.createElement("span");
  text.textContent = msg;
  t.append(text);
  if (isError) {
    const close = document.createElement("button");
    close.textContent = "✕";
    close.title = "Close";
    close.className = "ml-3 px-1 text-slate-400 hover:text-white";
    close.onclick = () => t.classList.add("hidden");
    t.append(close);
  }
  t.classList.toggle("toast-error", isError);
  t.classList.remove("hidden");
  clearTimeout(toastTimer);
  if (!isError) toastTimer = setTimeout(() => t.classList.add("hidden"), 3500);
}

// attempt runs an async action, showing a failure as a toast.
export async function attempt(fn, okMsg) {
  try {
    const r = await fn();
    if (okMsg) toast(okMsg);
    return r;
  } catch (e) {
    toast(e.message || String(e), true);
    return undefined;
  }
}

// busy disables a button and shows a label while fn runs.
export async function busy(btn, label, fn) {
  const old = btn.innerHTML;
  btn.disabled = true;
  btn.textContent = label;
  try {
    return await fn();
  } finally {
    btn.disabled = false;
    btn.innerHTML = old;
  }
}

export const canManage = (user) => user?.role === "admin" || user?.role === "editor";

export function fmtMin(m) {
  if (!m) return "";
  if (m < 60) return `${m} min`;
  const h = Math.floor(m / 60), r = m % 60;
  return r ? `${h} h ${r} min` : `${h} h`;
}

export const HEAT = ["No heat", "Mild", "Warm", "Hot", "Very hot", "Extreme"];

// money formats a price in the house currency.
export function money(v, currency) {
  try {
    return new Intl.NumberFormat(undefined, { style: "currency", currency: currency || "USD" }).format(v);
  } catch {
    return (Math.round(v * 100) / 100).toFixed(2);
  }
}

// levelChip shows a recipe's difficulty.
export const LEVELS = { easy: "🟢 Easy", medium: "🟡 Medium", hard: "🔴 Hard" };
export const levelChip = (level) => (LEVELS[level] ? `<span class="chip-info" title="How hard it is to make">${LEVELS[level]}</span>` : "");

export function peppers(level) {
  if (level === undefined || level === null || level < 0) return "";
  if (level === 0) return `<span class="chip-info" title="No heat">No heat</span>`;
  return `<span class="chip-info" title="${esc(HEAT[level])} (${level} of 5)">${"🌶️".repeat(level)}</span>`;
}

export function stars(n) {
  return n > 0 ? `<span class="text-amber-300" title="${n} of 5 stars">${"★".repeat(n)}</span>` : "";
}

// sheet opens the shared dialog (a bottom sheet on phones) with html inside.
// Buttons marked data-close close it.
export function sheet(html) {
  const d = $("#sheet");
  d.innerHTML = `<div class="max-h-[88dvh] overflow-y-auto p-5">${html}</div>`;
  $$("[data-close]", d).forEach((b) => (b.onclick = () => d.close()));
  d.onclick = (e) => { if (e.target === d) d.close(); };
  if (!d.open) d.showModal();
  return d;
}

// Home & Care categories and kitchen courses, as the AI and editor name them.
export const COURSES = ["breakfast", "main", "side", "soup", "salad", "appetizer", "snack", "dessert", "baking", "bread", "drink", "sauce"];
export const HOME_CATS = ["cleaning", "laundry", "oral care", "personal care", "other"];
export const cap = (s) => (s ? s[0].toUpperCase() + s.slice(1) : "");
