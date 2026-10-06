// The meal plan: a week of meals, who's eating each day, and the week's shopping.
import { get, post, del } from "./api.js";
import { $, $$, esc, attempt, busy, canManage, toast, money } from "./ui.js";
import { verdictChips } from "./verdicts.js";
import { pickMeal, pickWho, localDate, dayLabel, MEAL_NAMES, mealName } from "./planpick.js";
import { planWeek } from "./planweek.js";

const MEALS = ["breakfast", "lunch", "dinner"];

export async function renderPlan(view, params, state) {
  const manage = canManage(state.user);
  const today = localDate();
  const start = params.from || today;
  const shift = (n) => { const d = new Date(start + "T00:00:00"); d.setDate(d.getDate() + n); return localDate(d); };
  const data = await get(`/api/plan?from=${start}&days=7`);
  const name = Object.fromEntries(data.people.map((p) => [p.id, p.name]));
  const reload = () => renderPlan(view, params, state);

  const mealRow = (d, meal) => {
    const items = d.meals.filter((m) => m.meal === meal);
    if (!items.length && meal === "snack") return "";
    return `<div class="flex flex-wrap items-start gap-2 py-2">
      <span class="w-20 shrink-0 pt-1 text-xs font-semibold uppercase tracking-wide text-slate-400">${MEAL_NAMES[meal]}</span>
      <div class="min-w-0 flex-1 basis-48 space-y-2">
        ${items.map((m) => `<div class="flex min-w-0 flex-wrap items-center gap-2">
          <div class="min-w-0 flex-1 basis-40">
            ${m.from ? "🍱 Leftovers: " : ""}${m.recipe ? `<a href="#/recipe/${m.recipe.id}" class="break-words font-medium hover:underline">${esc(m.recipe.title)}</a>`
              : `<span class="break-words font-medium">${esc(m.title || "(recipe removed)")}</span>`}
            ${m.servings ? `<span class="text-xs text-slate-400"> · for ${+m.servings}</span>` : ""}
            ${m.from ? `<span class="block text-xs text-slate-400">from ${esc(mealName(m.from))}</span>` : ""}
            ${m.for?.length ? `<span class="block text-xs text-slate-400">makes extra for ${esc(m.for.map(mealName).join(", "))}</span>` : ""}
            ${m.verdicts.length ? `<div class="mt-1 flex flex-wrap gap-1">${verdictChips(m.verdicts)}</div>` : ""}
          </div>
          ${manage ? `<button data-rm="${m.id}" class="btn-ghost min-h-0 px-2 py-1 text-slate-500" aria-label="Remove">✕</button>` : ""}
        </div>`).join("")}
        ${manage ? `<button data-add="${d.date}|${meal}" class="text-sm text-emerald-300 hover:underline">+ ${items.length ? "Add another" : "Add"}</button>` : ""}
        ${!items.length && !manage ? `<span class="text-sm text-slate-500">Nothing planned</span>` : ""}
      </div></div>`;
  };

  view.innerHTML = `
    <div class="mb-4 flex flex-wrap items-end gap-3">
      <div class="min-w-0 basis-full sm:basis-auto sm:flex-1"><h1 class="text-2xl font-bold">📅 Meal plan</h1>
        ${data.cost?.priced ? `<p class="text-sm text-slate-400">These recipes: about ${money(data.cost.total, state.info?.currency)} from the pantry's prices
          (${data.cost.priced} of ${data.cost.lines} ingredients priced).</p>` : ""}
        <p class="text-sm text-slate-400">Plan the week; every meal is checked for who's eating that day.</p></div>
      ${manage ? `<button id="week" class="btn-primary">✨ Plan my week</button>
        <button id="shop" class="btn-secondary">🛒 Add this week to the shopping list</button>` : ""}
    </div>
    <div class="mb-4 flex flex-wrap items-center gap-2">
      <a href="#/plan?from=${shift(-7)}" class="btn-secondary">‹ Earlier</a>
      ${start !== today ? `<a href="#/plan" class="btn-ghost">This week</a>` : ""}
      <a href="#/plan?from=${shift(7)}" class="btn-secondary">Later ›</a>
    </div>
    <div class="grid gap-3 lg:grid-cols-2">${data.days.map((d) => `
      <div class="card min-w-0 ${d.date === today ? "ring-emerald-700" : ""}">
        <div class="flex flex-wrap items-center gap-2">
          <h2 class="mr-auto font-semibold">${esc(dayLabel(d.date))}${d.date === today ? ` <span class="chip-ok">Today</span>` : ""}</h2>
          ${manage ? `<button data-who="${d.date}" class="text-xs text-slate-400 hover:underline">✎ Who's eating</button>` : ""}
        </div>
        <p class="text-xs text-slate-400">Eating: ${d.who.length ? esc(d.who.map((id) => name[id]).filter(Boolean).join(", ")) : "nobody picked"}${d.who_set ? "" : " (everyone)"}</p>
        <div class="divide-y divide-slate-800">${[...MEALS, "snack"].map((m) => mealRow(d, m)).join("")}</div>
      </div>`).join("")}</div>`;

  $$("[data-add]", view).forEach((b) => (b.onclick = () => {
    const [date, meal] = b.dataset.add.split("|");
    pickMeal(date, meal, data.days.find((x) => x.date === date).who, reload);
  }));
  $$("[data-who]", view).forEach((b) => (b.onclick = () => {
    const d = data.days.find((x) => x.date === b.dataset.who);
    pickWho(d.date, data.people, d.who, reload);
  }));
  $$("[data-rm]", view).forEach((b) => (b.onclick = () => attempt(async () => { await del(`/api/plan/${b.dataset.rm}`); reload(); })));
  const week = $("#week", view);
  if (week) week.onclick = () => attempt(() => busy(week, "Thinking…", () => planWeek(start, state, reload)));
  const shop = $("#shop", view);
  if (shop) shop.onclick = () => attempt(async () => {
    const r = await post("/api/plan/shopping", { from: start, days: 7 });
    toast(`Added ${r.added} to the shopping list${r.have.length ? `; ${r.have.length} look like they're in the house already` : ""}`);
  });
}
