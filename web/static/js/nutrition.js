// "🥗 Nutrition": an estimate per serving from USDA FoodData Central, with
// what it couldn't count and how each line was worked out.
import { get } from "./api.js";
import { $, esc, attempt, busy } from "./ui.js";

export const nutritionCard = () => `<div id="nutri" class="no-print card space-y-3">
  <div class="flex flex-wrap items-center gap-2"><h2 class="mr-auto font-semibold">🥗 Nutrition</h2>
    <button type="button" id="nutri-go" class="btn-secondary min-h-0 py-1 text-sm">Estimate</button></div>
  <p class="text-xs text-slate-500">An estimate from USDA FoodData Central. Only the food names are sent.</p></div>`;

const ROWS = [["kcal", "Calories", ""], ["protein_g", "Protein", "g"], ["carbs_g", "Carbohydrates", "g"], ["sugar_g", "  of which sugars", "g"],
  ["fiber_g", "Fiber", "g"], ["fat_g", "Fat", "g"], ["sodium_mg", "Sodium", "mg"]];

function show(box, a, manage) {
  const e = a.estimate;
  box.classList.toggle("no-print", !e.counted); // print it once there's something to print
  const n = e.per_serving;
  const round = (v, unit) => (unit === "g" && v < 10 ? v.toFixed(1) : Math.round(v)).toString();
  box.innerHTML = `<div class="flex flex-wrap items-center gap-2"><h2 class="mr-auto font-semibold">🥗 Nutrition</h2>
      <span class="text-xs text-slate-400">${e.servings > 0 ? `per serving (of ${+e.servings})` : "for the whole recipe"}</span></div>
    ${a.key_set ? "" : `<p class="box-info text-sm">Estimates need a free USDA FoodData Central key.
      ${manage ? "An admin can add one under <b>Admin → House settings</b>." : "Ask whoever set up RecipeBank."}</p>`}
    ${a.error ? `<p class="text-sm text-amber-300">${esc(a.error)}</p>` : ""}
    ${e.counted ? `<table class="w-full text-sm"><tbody>${ROWS.map(([k, label, unit]) => `<tr class="border-b border-slate-800">
      <td class="py-1 ${label.startsWith(" ") ? "pl-4 text-slate-400" : ""}">${label.trim()}</td>
      <td class="py-1 text-right font-semibold">${round(n[k], unit)}${unit ? " " + unit : ""}</td></tr>`).join("")}</tbody></table>` : ""}
    ${e.not_counted.length ? `<p class="text-sm ${e.counted < e.not_counted.length ? "text-amber-300" : "text-slate-400"}">Counted
      ${e.counted} of ${e.counted + e.not_counted.length} lines${e.counted < e.not_counted.length ? ", so the real numbers are higher" : ""}.
      Not counted: ${esc(e.not_counted.join("; "))}</p>` : ""}
    ${e.lines.length ? `<details class="text-sm"><summary class="cursor-pointer text-slate-400">How it was worked out</summary>
      <ul class="mt-2 space-y-1">${e.lines.map((l) => `<li class="break-words">${esc(l.text)} <span class="text-slate-500">→ ${Math.round(l.grams)} g of
        “${esc(l.food)}”</span></li>`).join("")}</ul></details>` : ""}
    <p class="text-xs text-slate-500">An estimate from USDA FoodData Central (public domain): brands and cooking change the real numbers.</p>`;
}

// wireNutrition sets up the Estimate button for recipe r.
export function wireNutrition(view, r, manage) {
  const go = $("#nutri-go", view);
  if (go) go.onclick = () => attempt(() => busy(go, "Working it out…", async () => {
    show($("#nutri", view), await get(`/api/recipes/${r.id}/nutrition`), manage);
  }));
}
