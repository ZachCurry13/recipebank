// Me → Keep it simple: each person can show less, and in their own order,
// without changing anything for the others: which meals their plan shows,
// and which pages their menu has and in what order.
import { put } from "./api.js";
import { $, $$, esc, attempt, toast, featureOn, ALL_MEALS, planMeals, hiddenPages } from "./ui.js";
import { menuOrder, ALWAYS, PAGE_FEATURES } from "./nav.js";
import { rebuildNav } from "./app.js";

const MEAL_LABELS = { breakfast: "Breakfast", lunch: "Lunch", dinner: "Dinner", snack: "Snacks" };

export function renderSimpler(box, state) {
  const u = state.user;
  const meals = planMeals(u), hidden = hiddenPages(u);
  const usable = ([r, , , may]) => may(u) && (PAGE_FEATURES[r] || []).every((f) => featureOn(state, f));
  // rows: the pages in this person's order, each shown or not.
  const rows = menuOrder(u).filter(usable).map(([r, label, ico]) => ({ r, label, ico, on: ALWAYS.includes(r) || !hidden.includes(r) }));
  box.innerHTML = `<form class="card space-y-4">
    <div><h2 class="font-semibold">🌿 Keep it simple</h2>
      <p class="text-sm text-slate-400">Show less of RecipeBank, in your own order, just for you. Nothing changes for anyone else,
        and nothing is deleted.</p></div>
    ${featureOn(state, "plan") ? `<fieldset class="space-y-2"><legend class="label">Meals my plan shows</legend>
      <div class="flex flex-wrap gap-2">${ALL_MEALS.map((m) => `<label class="pick has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-emerald-500 ${meals.includes(m) ? "on" : ""}">
        <input type="checkbox" data-meal="${m}" class="sr-only" ${meals.includes(m) ? "checked" : ""}>${MEAL_LABELS[m]}</label>`).join("")}</div>
      <p class="text-xs text-slate-500">Only dinner? Untick the others. Meals someone else plans stay in the plan.</p></fieldset>` : ""}
    <fieldset class="space-y-2"><legend class="label">My menu, in order</legend>
      <p class="text-xs text-slate-500">The first five are on a phone's bottom bar; on a computer the first four to seven are in the top bar.
        Untick a page to leave it out. Tonight and Recipes always stay.</p>
      <ol data-rows class="space-y-1"></ol></fieldset>
    <button class="btn-primary">Save</button></form>`;
  const form = $("form", box);
  $$("[data-meal]", form).forEach((c) => (c.onchange = () => c.closest("label").classList.toggle("on", c.checked)));
  const list = $("[data-rows]", form);
  const draw = () => {
    list.innerHTML = rows.map((row, i) => `<li class="flex items-center gap-2 rounded-lg border border-slate-800 px-2 py-1">
      <input type="checkbox" data-show="${i}" aria-label="Show ${esc(row.label)}" ${row.on ? "checked" : ""} ${ALWAYS.includes(row.r) ? "disabled" : ""}>
      <span class="min-w-0 flex-1 break-words ${row.on ? "" : "text-slate-500 line-through"}">${row.ico} ${esc(row.label)}</span>
      <button type="button" data-up="${i}" class="btn-ghost min-h-0 px-2 py-1" aria-label="Move ${esc(row.label)} up" ${i ? "" : "disabled"}>↑</button>
      <button type="button" data-down="${i}" class="btn-ghost min-h-0 px-2 py-1" aria-label="Move ${esc(row.label)} down" ${i < rows.length - 1 ? "" : "disabled"}>↓</button></li>`).join("");
    const swap = (a, b) => { [rows[a], rows[b]] = [rows[b], rows[a]]; draw(); list.querySelector(`[data-${a < b ? "down" : "up"}="${b}"]`)?.focus(); };
    $$("[data-up]", list).forEach((b) => (b.onclick = () => swap(Number(b.dataset.up), Number(b.dataset.up) - 1)));
    $$("[data-down]", list).forEach((b) => (b.onclick = () => swap(Number(b.dataset.down), Number(b.dataset.down) + 1)));
    $$("[data-show]", list).forEach((c) => (c.onchange = () => { rows[Number(c.dataset.show)].on = c.checked; draw(); }));
  };
  draw();
  form.onsubmit = (e) => {
    e.preventDefault();
    const nextMeals = featureOn(state, "plan") ? $$("[data-meal]", form).filter((c) => c.checked).map((c) => c.dataset.meal) : meals;
    if (!nextMeals.length) {
      toast("Keep at least one meal on your plan.", true);
      return;
    }
    // Pages not listed here (turned off for the house) keep their place and what the person chose before.
    const listed = rows.map((r) => r.r);
    const order = [...listed, ...menuOrder(u).map((n) => n[0]).filter((r) => !listed.includes(r))];
    const nextHidden = [...hidden.filter((p) => !listed.includes(p)), ...rows.filter((r) => !r.on).map((r) => r.r)];
    attempt(async () => {
      await put("/api/me/simpler", { plan_meals: nextMeals, hidden_pages: nextHidden, menu_order: order });
      u.plan_meals = nextMeals.length === ALL_MEALS.length ? "" : nextMeals.join(",");
      u.hidden_pages = nextHidden.join(",");
      u.menu_order = order.join(",");
      rebuildNav();
    }, "Saved");
  };
}
