// Pantry (food) and Supplies (cleaning and bathroom): what's in the house,
// what to use soon, and what's running low.
import { get, post } from "./api.js";
import { $, $$, esc, attempt, canManage, cap } from "./ui.js";
import { editStock, scanStock } from "./stockedit.js";

const view_ = { kitchen: { q: "", show: "" }, home: { q: "", show: "" } };

// daysLeft: days until a YYYY-MM-DD date (negative once past).
export function daysLeft(date) {
  if (!date) return null;
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  return Math.round((new Date(date + "T00:00:00") - today) / 86400000);
}

export function useByChip(date) {
  const d = daysLeft(date);
  if (d === null) return "";
  const when = new Date(date + "T00:00:00").toLocaleDateString(undefined, { weekday: "short", month: "short", day: "numeric" });
  if (d < 0) return `<span class="chip-no">Past use-by (${esc(when)})</span>`;
  if (d <= 3) return `<span class="chip-unsure">Use by ${d === 0 ? "today" : esc(when)}</span>`;
  return `<span class="chip-info">Use by ${esc(when)}</span>`;
}

const isLow = (it) => it.low_at > 0 && it.qty <= it.low_at;
const soon = (it) => { const d = daysLeft(it.use_by); return d !== null && d <= 3; };

export async function renderStock(view, area, params, state) {
  const home = area === "home";
  const f = view_[area];
  const manage = canManage(state.user);
  const names = Object.fromEntries((state.info?.all_allergens || []).map((a) => [a.key, a.label]));
  let items = await get(`/api/stock?area=${area}`);
  const reload = async () => { items = await get(`/api/stock?area=${area}`); draw(); };

  const row = (it) => `<li class="flex flex-wrap items-center gap-2 py-2">
    <div class="min-w-0 flex-1 basis-40">
      ${manage ? `<button data-edit="${it.id}" class="break-words text-left font-medium hover:underline">${esc(it.name)}</button>` : `<span class="break-words font-medium">${esc(it.name)}</span>`}
      ${it.brand ? `<span class="text-xs text-slate-400">${esc(it.brand)}</span>` : ""}
      <div class="mt-1 flex flex-wrap gap-1">${useByChip(it.use_by)}
        ${isLow(it) ? `<span class="chip-unsure">${it.qty ? "Running low" : "Out"}</span>` : ""}
        ${it.allergens.map((a) => `<span class="chip-no">${esc(names[a] || a)}</span>`).join("")}
        ${it.traces.length ? `<span class="chip-info" title="May contain">May contain ${esc(it.traces.map((a) => names[a] || a).join(", "))}</span>` : ""}</div>
    </div>
    <div class="flex items-center gap-1">
      <button data-adj="${it.id}:-1" class="btn-secondary min-h-0 px-3 py-1" aria-label="One less">−</button>
      <span class="min-w-[3.5rem] text-center text-sm tabular-nums">${+it.qty.toFixed(2)}${it.unit ? " " + esc(it.unit) : ""}</span>
      <button data-adj="${it.id}:1" class="btn-secondary min-h-0 px-3 py-1" aria-label="One more">+</button>
    </div></li>`;

  const draw = () => {
    const q = f.q.toLowerCase();
    let list = items.filter((it) => !q || (it.name + " " + it.brand).toLowerCase().includes(q));
    if (f.show === "soon") list = list.filter(soon);
    if (f.show === "low") list = list.filter(isLow);
    const groups = {};
    for (const it of list) (groups[it.location || "Other"] ??= []).push(it);
    const nSoon = items.filter(soon).length, nLow = items.filter(isLow).length;
    view.innerHTML = `
      <div class="mb-4 flex flex-wrap items-end gap-3">
        <div class="min-w-0 basis-full sm:basis-auto sm:flex-1">
          <h1 class="text-2xl font-bold">${home ? "🧽 Supplies" : "🥫 Pantry"}</h1>
          <p class="text-sm text-slate-400">${home ? "Cleaning, laundry and bathroom supplies: what's in the house and what's running low."
            : "Food in the house: what's here, what to use soon, and what's running low."}</p>
        </div>
        ${manage ? `<button id="scan" class="btn-primary">📷 Scan a barcode</button><button id="add" class="btn-secondary">➕ Add</button>` : ""}
      </div>
      <div class="card mb-4 space-y-3">
        <input id="q" type="search" class="input" placeholder="Search" value="${esc(f.q)}">
        <div class="flex flex-wrap gap-2">
          <button data-show="" class="pick ${f.show === "" ? "on" : ""}">All (${items.length})</button>
          ${home ? "" : `<button data-show="soon" class="pick ${f.show === "soon" ? "on" : ""}">Use soon (${nSoon})</button>`}
          <button data-show="low" class="pick ${f.show === "low" ? "on" : ""}">Running low (${nLow})</button>
        </div>
      </div>
      ${list.length ? Object.keys(groups).sort().map((g) => `<div class="card mb-3">
          <h2 class="label">${esc(cap(g))}</h2><ul class="divide-y divide-slate-800">${groups[g].map(row).join("")}</ul></div>`).join("")
        : `<div class="card text-center text-slate-400">${items.length ? "Nothing matches." : `Nothing here yet.${manage ? " Scan a barcode or add things by hand." : ""}`}</div>`}`;
    const qIn = $("#q", view);
    qIn.oninput = () => {
      f.q = qIn.value;
      const pos = qIn.selectionStart;
      draw();
      const again = $("#q", view);
      again.focus();
      again.setSelectionRange(pos, pos);
    };
    $$("[data-show]", view).forEach((b) => (b.onclick = () => { f.show = b.dataset.show; draw(); }));
    $$("[data-adj]", view).forEach((b) => (b.onclick = () => attempt(async () => {
      const [id, d] = b.dataset.adj.split(":");
      const it = await post(`/api/stock/${id}/adjust`, { delta: Number(d) });
      items = items.map((x) => (x.id === it.id ? it : x));
      draw();
    })));
    $$("[data-edit]", view).forEach((b) => (b.onclick = () =>
      editStock(area, items.find((x) => x.id === Number(b.dataset.edit)), state, reload)));
    const add = $("#add", view);
    if (add) add.onclick = () => editStock(area, null, state, reload);
    const scan = $("#scan", view);
    if (scan) scan.onclick = () => scanStock(area, state, reload);
  };
  draw();
}
