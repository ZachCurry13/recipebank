// Tonight: the start page. What's planned today and who's eating (with
// everyone's checks), food to use soon, and ideas when nothing's planned.
import { get, post } from "./api.js";
import { $, $$, esc, attempt, canManage, fmtMin, toast } from "./ui.js";
import { verdictList } from "./verdicts.js";
import { startCooking } from "./cook.js";
import { useByChip } from "./stock.js";
import { localDate, dayLabel, MEAL_NAMES, mealName } from "./planpick.js";
import { renderGuide } from "./guide.js";

export async function renderTonight(view, params, state) {
  const date = localDate();
  const data = await get(`/api/tonight?date=${date}`);
  const manage = canManage(state.user);
  const system = state.user.units || state.info?.default_units || "us";
  const name = Object.fromEntries(data.people.map((p) => [p.id, p.name]));
  // Planned leftovers make the meal bigger: cook and buy for them too.
  const factorOf = (m) => {
    const rs = m.recipe?.servings;
    if (!rs) return 1 + (m.for?.length || 0);
    return ((m.servings || rs) + (m.extra || 0)) / rs;
  };
  const photo = (p, cls) => (p ? `<img src="/api/photos/${esc(p)}?w=1200" alt="" class="${cls}">` : "");

  view.innerHTML = `
    <div id="guide"></div>
    ${(data.events || []).map((e) => `<a href="#/event/${e.id}" class="box-info mb-4 block break-words">🎉 Today: <b>${esc(e.name)}</b>. Open the menu ›</a>`).join("")}
    <h1 class="text-2xl font-bold">🌙 Tonight</h1>
    <p class="mb-4 text-sm text-slate-400">${esc(dayLabel(date, true))} · Eating: ${data.who.length ? esc(data.who.map((id) => name[id]).filter(Boolean).join(", ")) : "nobody picked"}
      ${manage ? `· <a href="#/plan" class="underline">change</a>` : ""}</p>
    ${data.meals.map((m, i) => `<div class="card mb-4 space-y-3">
      ${m.recipe ? photo(m.recipe.photo, "max-h-64 w-full rounded-lg object-cover") : ""}
      <div><span class="label">${MEAL_NAMES[m.meal]}${m.from ? " · 🍱 leftovers" : ""}</span>
        <h2 class="break-words text-xl font-semibold">${m.recipe ? `<a href="#/recipe/${m.recipe.id}" class="hover:underline">${esc(m.recipe.title)}</a>` : esc(m.title)}</h2>
        ${m.recipe?.total_min ? `<span class="chip-info">⏱ ${fmtMin(m.recipe.total_min)}</span>` : ""}
        ${m.servings ? `<span class="chip-info">For ${+m.servings}</span>` : ""}
        ${m.from ? `<p class="mt-1 text-sm text-slate-300">From ${esc(mealName(m.from))}: just reheat.</p>` : ""}
        ${m.for?.length ? `<p class="mt-1 text-sm text-slate-300">Make extra: leftovers are planned for ${esc(m.for.map(mealName).join(", "))}.</p>` : ""}</div>
      ${m.full.length ? verdictList(m.full, []) : ""}
      ${m.recipe && !m.from ? `<div class="flex flex-wrap gap-2">
        <button data-cook="${i}" class="btn-primary">▶ Start cooking</button>
        <a href="#/recipe/${m.recipe.id}" class="btn-secondary">Open recipe</a>
        <button data-shop="${i}" class="btn-ghost">🛒 Add to shopping list</button></div>` : ""}
    </div>`).join("")}
    ${data.meals.some((m) => m.meal === "dinner") ? "" : `<div class="card mb-4 space-y-3">
      <h2 class="font-semibold">Nothing planned for dinner${data.meals.length ? "" : " yet"}</h2>
      ${data.ideas.length ? `<p class="text-sm text-slate-400">Ideas everyone eating can have:</p>
        <div class="grid gap-2 sm:grid-cols-2">${data.ideas.map((r) => `<div class="flex min-w-0 items-center gap-3 rounded-lg p-2 ring-1 ring-slate-800">
          ${r.photo ? `<img src="/api/photos/${esc(r.photo)}?w=480" alt="" class="h-14 w-14 shrink-0 rounded-lg object-cover">` : `<span class="text-3xl">🍲</span>`}
          <div class="min-w-0 flex-1"><a href="#/recipe/${r.id}" class="break-words font-medium hover:underline">${esc(r.title)}</a>
            <div class="text-xs text-slate-400">${r.total_min ? fmtMin(r.total_min) : ""}${r.liked ? `${r.total_min ? " · " : ""}👍 liked before` : ""}</div></div>
          ${manage ? `<button data-plan="${r.id}" class="btn-secondary min-h-0 py-1">Plan it</button>` : ""}</div>`).join("")}</div>`
        : `<p class="text-sm text-slate-400">No saved recipe suits everyone eating yet.</p>`}
      <div class="flex flex-wrap gap-2"><a href="#/make" class="btn-secondary">🥕 What can I make?</a>
        ${manage ? `<a href="#/plan" class="btn-ghost">📅 Plan the week</a>` : ""}</div></div>`}
    ${data.use_soon.length ? `<div class="card mb-4"><h2 class="mb-2 font-semibold">Use soon</h2>
      <ul class="space-y-1">${data.use_soon.map((it) => `<li class="flex flex-wrap items-center gap-2"><span class="min-w-0 break-words">${esc(it.name)}</span>${useByChip(it.use_by)}</li>`).join("")}</ul>
      <a href="#/pantry" class="mt-2 inline-block text-sm underline">Open the pantry</a></div>` : ""}
    ${data.next ? `<p class="text-sm text-slate-400">Next up: ${esc(dayLabel(data.next.date))}, ${MEAL_NAMES[data.next.meal].toLowerCase()}:
      ${data.next.recipe ? `<a href="#/recipe/${data.next.recipe.id}" class="underline">${esc(data.next.recipe.title)}</a>` : esc(data.next.title)}</p>` : ""}`;

  if (state.user.role === "admin") renderGuide($("#guide", view)).catch(() => {});
  $$("[data-cook]", view).forEach((b) => (b.onclick = () => attempt(async () => {
    const m = data.meals[Number(b.dataset.cook)];
    const full = await get(`/api/recipes/${m.recipe.id}`);
    startCooking(full.recipe, { factor: factorOf(m), system });
  })));
  $$("[data-shop]", view).forEach((b) => (b.onclick = () => attempt(async () => {
    const m = data.meals[Number(b.dataset.shop)];
    const r = await post("/api/shopping/recipe", { id: m.recipe.id, factor: factorOf(m) });
    toast(`Added ${r.added} to the shopping list${r.have.length ? `; ${r.have.length} look like they're in the house already` : ""}`);
  })));
  $$("[data-plan]", view).forEach((b) => (b.onclick = () => attempt(async () => {
    await post("/api/plan", { date, meal: "dinner", recipe_id: Number(b.dataset.plan) });
    renderTonight(view, params, state);
  })));
}
