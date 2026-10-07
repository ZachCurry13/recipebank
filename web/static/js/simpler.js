// Me → Keep it simple: each person can show less without changing anything
// for the others: which meals their plan shows, and which pages their menu has.
import { put } from "./api.js";
import { $, $$, attempt, toast, featureOn, ALL_MEALS, planMeals, hiddenPages } from "./ui.js";
import { NAV, PAGE_FEATURES } from "./nav.js";
import { rebuildNav } from "./app.js";

const MEAL_LABELS = { breakfast: "Breakfast", lunch: "Lunch", dinner: "Dinner", snack: "Snacks" };
// Pages a person may leave out of their menu (Tonight, Kitchen, Me and Admin always stay).
const HIDEABLE = ["plan", "shopping", "make", "home", "pantry", "supplies", "collections", "books", "events", "add", "family"];

export function renderSimpler(box, state) {
  const u = state.user;
  const meals = planMeals(u), hidden = hiddenPages(u);
  const pages = NAV.filter(([r, , , may]) => HIDEABLE.includes(r) && may(u) &&
    (PAGE_FEATURES[r] || []).every((f) => featureOn(state, f)));
  box.innerHTML = `<form class="card space-y-4">
    <div><h2 class="font-semibold">🌿 Keep it simple</h2>
      <p class="text-sm text-slate-400">Show less of RecipeBank, just for you. Nothing changes for anyone else, and nothing is deleted.</p></div>
    ${featureOn(state, "plan") ? `<fieldset class="space-y-2"><legend class="label">Meals my plan shows</legend>
      <div class="flex flex-wrap gap-2">${ALL_MEALS.map((m) => `<label class="pick has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-emerald-500 ${meals.includes(m) ? "on" : ""}">
        <input type="checkbox" data-meal="${m}" class="sr-only" ${meals.includes(m) ? "checked" : ""}>${MEAL_LABELS[m]}</label>`).join("")}</div>
      <p class="text-xs text-slate-500">Only dinner? Untick the others. Meals someone else plans stay in the plan.</p></fieldset>` : ""}
    ${pages.length ? `<fieldset class="space-y-2"><legend class="label">Pages in my menu</legend>
      <div class="flex flex-wrap gap-2">${pages.map(([r, label, ico]) => `<label class="pick has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-emerald-500 ${hidden.includes(r) ? "" : "on"}">
        <input type="checkbox" data-page="${r}" class="sr-only" ${hidden.includes(r) ? "" : "checked"}>${ico} ${label}</label>`).join("")}</div>
      <p class="text-xs text-slate-500">Tonight, the Kitchen and Me always stay.</p></fieldset>` : ""}
    <button class="btn-primary">Save</button></form>`;
  const form = $("form", box);
  $$("input[type=checkbox]", form).forEach((c) => (c.onchange = () => c.closest("label").classList.toggle("on", c.checked)));
  form.onsubmit = (e) => {
    e.preventDefault();
    const nextMeals = featureOn(state, "plan") ? $$("[data-meal]", form).filter((c) => c.checked).map((c) => c.dataset.meal) : meals;
    if (!nextMeals.length) {
      toast("Keep at least one meal on your plan.", true);
      return;
    }
    // Pages not listed here (turned off for the house) keep what the person chose before.
    const boxes = $$("[data-page]", form);
    const listed = boxes.map((c) => c.dataset.page);
    const nextHidden = [...hidden.filter((p) => !listed.includes(p)), ...boxes.filter((c) => !c.checked).map((c) => c.dataset.page)];
    attempt(async () => {
      await put("/api/me/simpler", { plan_meals: nextMeals, hidden_pages: nextHidden });
      u.plan_meals = nextMeals.length === ALL_MEALS.length ? "" : nextMeals.join(",");
      u.hidden_pages = nextHidden.join(",");
      rebuildNav();
    }, "Saved");
  };
}
