// The shopping list: shared by the family, grouped by store section, and
// usable in the store without signal (ticks are sent once back online).
import { get, post, put, del } from "./api.js";
import { $, $$, esc, attempt, toast, money, canEmail } from "./ui.js";

// budgetCard: about what the list costs, from the pantry's prices, against
// the weekly budget if one is set.
function budgetCard(b, currency) {
  const c = b?.cost;
  if (!c || (!c.priced && !b.weekly)) return "";
  if (!c.priced) return `<p class="mb-4 text-sm text-slate-400">No prices yet: give pantry items a price to see what the list costs
    against the ${money(b.weekly, currency)} weekly budget.</p>`;
  const over = b.weekly > 0 && c.total > b.weekly;
  return `<div class="card mb-4 space-y-2 text-sm">
    <p>About <b>${money(c.total, currency)}</b> for ${c.priced} of ${c.lines} thing${c.lines === 1 ? "" : "s"}${c.lines > c.priced ? ` (${c.lines - c.priced} without a price in the pantry)` : ""}.</p>
    ${b.weekly > 0 ? `<progress class="h-2 w-full" max="${b.weekly}" value="${Math.min(c.total, b.weekly)}"></progress>
      <p class="${over ? "text-rose-300" : "text-slate-400"}">${over ? `${money(c.total - b.weekly, currency)} over` : `${money(b.weekly - c.total, currency)} left of`} the ${money(b.weekly, currency)} weekly budget.</p>` : ""}</div>`;
}
import { amountText } from "./units.js";
import { emailSheet } from "./email.js";

const SECTIONS = ["Produce", "Bakery", "Meat & fish", "Dairy & eggs", "Frozen", "Pantry", "Spices & baking", "Drinks",
  "Household", "Personal care", "Other"];
const CACHE = "rb:shopping";
const QUEUE = "rb:shopqueue";

const load = (k, d) => { try { return JSON.parse(localStorage.getItem(k)) ?? d; } catch { return d; } };
const save = (k, v) => { try { localStorage.setItem(k, JSON.stringify(v)); } catch { /* private mode */ } };

// flush sends ticks made while offline (also when the app starts).
export async function flush() {
  const queue = load(QUEUE, []);
  while (queue.length) {
    const t = queue[0];
    try {
      await put(`/api/shopping/${t.id}`, { checked: t.checked });
    } catch (e) {
      if (e.status === 0) return false; // still offline
    }
    queue.shift();
    save(QUEUE, queue);
  }
  return true;
}

export async function renderShopping(view, params, state) {
  const system = state.user.units || state.info?.default_units || "us";
  const currency = state.info?.currency;
  let data = { items: [], low: [] };
  let offline = false;
  const refresh = async () => {
    try {
      await flush();
      data = await get("/api/shopping");
      save(CACHE, { ...data, at: Date.now() });
      offline = false;
    } catch (e) {
      if (e.status !== 0) throw e;
      const cached = load(CACHE, null);
      if (!cached) throw new Error("You're offline, and this phone hasn't saved the list yet. Open the list once while online.");
      data = cached;
      offline = true;
    }
    draw();
  };
  const tick = (it) => {
    it.checked = !it.checked;
    save(CACHE, { ...data, at: Date.now() });
    draw();
    put(`/api/shopping/${it.id}`, { checked: it.checked }).catch((e) => {
      if (e.status !== 0) return toast(e.message, true);
      const queue = load(QUEUE, []).filter((q) => q.id !== it.id);
      queue.push({ id: it.id, checked: it.checked });
      save(QUEUE, queue);
      offline = true;
      draw();
    });
  };
  const row = (it) => `<li class="flex items-center gap-3 py-1">
    <label class="flex min-h-[2.75rem] min-w-0 flex-1 cursor-pointer items-center gap-3">
      <input type="checkbox" data-tick="${it.id}" class="h-6 w-6 shrink-0 accent-emerald-600" ${it.checked ? "checked" : ""}>
      <span class="min-w-0 break-words ${it.checked ? "line-through-soft" : ""}">
        ${it.qty ? `<b>${esc(amountText(it.qty, it.unit, system))}</b> ` : ""}${esc(it.name)}
        ${!it.checked && data.costs?.[it.id] ? `<span class="text-xs text-slate-400"> · ≈ ${money(data.costs[it.id], currency)}</span>` : ""}
        ${it.note ? `<span class="block text-xs text-slate-500">${esc(it.note)}</span>` : ""}</span></label>
    ${offline ? "" : `<button data-rm="${it.id}" class="btn-ghost min-h-0 px-2 py-1 text-slate-500" aria-label="Remove">✕</button>`}</li>`;
  const draw = () => {
    const open = data.items.filter((i) => !i.checked);
    const done = data.items.filter((i) => i.checked);
    const groups = SECTIONS.map((s) => [s, open.filter((i) => (i.section || "Other") === s)]).filter(([, l]) => l.length);
    view.innerHTML = `
      <h1 class="text-2xl font-bold">🛒 Shopping list</h1>
      <p class="mb-4 text-sm text-slate-400">One list for the whole family. Add recipes from their page, or type things here.</p>
      ${!offline && open.length && canEmail(state) ? `<button id="mail-list" class="btn-ghost mb-3 min-h-0 py-1">✉️ Email the list</button>` : ""}
      ${!offline ? budgetCard(data.budget, currency) : ""}
      ${offline ? `<p class="box-caution mb-4">Offline: showing the list saved on this phone. Ticks are kept and sent when you're back online.</p>` : ""}
      ${offline ? "" : `<form id="add" class="mb-4 flex flex-wrap gap-2"><input name="text" class="input min-w-0 flex-1 basis-48" placeholder="Add something: 2 lb apples, milk" autocomplete="off">
        <button class="btn-primary">Add</button></form>`}
      ${!offline && data.low.length ? `<div class="card mb-4"><div class="mb-2 flex flex-wrap items-center gap-2"><h2 class="mr-auto font-semibold">Running low</h2>
        <button id="add-low" class="btn-secondary min-h-0 py-1">Add all</button></div>
        <ul class="space-y-1">${data.low.map((s) => `<li class="flex items-center gap-2"><span class="min-w-0 flex-1 break-words">${esc(s.name)}
          <span class="text-xs text-slate-500">${s.area === "home" ? "Supplies" : "Pantry"}: ${+s.qty.toFixed(2)} left</span></span>
          <button data-low="${s.id}" class="btn-ghost min-h-0 py-1">+ Add</button></li>`).join("")}</ul></div>` : ""}
      ${groups.length ? groups.map(([s, l]) => `<div class="card mb-3"><h2 class="label">${esc(s)}</h2><ul class="divide-y divide-slate-800">${l.map(row).join("")}</ul></div>`).join("")
        : `<div class="card mb-3 text-center text-slate-400">${done.length ? "Everything's in the cart." : "The list is empty."}</div>`}
      ${done.length ? `<div class="card"><div class="mb-1 flex flex-wrap items-center gap-2"><h2 class="mr-auto label">In the cart (${done.length})</h2>
        ${offline ? "" : `<button id="clear" class="btn-secondary min-h-0 py-1">Clear ticked items</button>`}</div>
        <ul class="divide-y divide-slate-800">${done.map(row).join("")}</ul></div>` : ""}`;
    $$("[data-tick]", view).forEach((c) => (c.onchange = () => tick(data.items.find((i) => i.id === Number(c.dataset.tick)))));
    const mailList = $("#mail-list", view);
    if (mailList) mailList.onclick = () => emailSheet(state, "the shopping list", "/api/shopping/email", false);
    $$("[data-rm]", view).forEach((b) => (b.onclick = () => attempt(async () => { await del(`/api/shopping/${b.dataset.rm}`); await refresh(); })));
    $$("[data-low]", view).forEach((b) => (b.onclick = () => attempt(async () => { data = await post("/api/shopping", { stock_id: Number(b.dataset.low) }); draw(); })));
    const addLow = $("#add-low", view);
    if (addLow) addLow.onclick = () => attempt(async () => {
      for (const s of data.low) data = await post("/api/shopping", { stock_id: s.id });
      draw();
    });
    const form = $("#add", view);
    if (form) form.onsubmit = (e) => {
      e.preventDefault();
      const text = form.text.value.trim();
      if (!text) return;
      attempt(async () => { data = await post("/api/shopping", { text }); form.text.value = ""; draw(); $("#add input", view)?.focus(); });
    };
    const clear = $("#clear", view);
    if (clear) clear.onclick = () => attempt(async () => {
      const r = await post("/api/shopping/clear");
      toast(`Cleared ${r.cleared}${r.restocked.length ? `; back in stock: ${r.restocked.join(", ")}` : ""}`);
      await refresh();
    });
  };
  await refresh();
  // Others' changes show up while the list is open; ticks made offline go out once online.
  const timer = setInterval(() => { if (document.visibilityState === "visible") refresh().catch(() => {}); }, 15000);
  const online = () => refresh().catch(() => {});
  window.addEventListener("online", online);
  return () => { clearInterval(timer); window.removeEventListener("online", online); };
}
