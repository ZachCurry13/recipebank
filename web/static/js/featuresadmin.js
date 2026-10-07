// Admin → Features: turn off what the family doesn't use, for everyone. Off
// hides it from the menu and the pages; nothing is deleted, and turning it
// back on brings it all back. Each person can also simplify their own view
// on Me → Keep it simple.
import { put } from "./api.js";
import { $, $$, attempt } from "./ui.js";
import { refreshInfo } from "./app.js";

const FEATURES = [
  ["plan", "📅 Meal plan", "The week's meals, Plan my week, and Tonight's planned meals."],
  ["shopping", "🛒 Shopping list", "One shared list; \"add to the list\" buttons go with it."],
  ["make", "🥕 What can I make?", "Recipes from what's in the kitchen or a photo of the fridge."],
  ["home", "🧴 Home & Care", "Cleaner, toothpaste and other home recipes, and the supply closet."],
  ["pantry", "🥫 Pantry and supplies", "What's in the house: barcodes, receipts, use-by dates, running low."],
  ["collections", "📚 Collections and seasons", "Themed shelves and seasonal recipes."],
  ["books", "📖 Bookshelf", "Cookbooks by page, and the cards still to photograph."],
  ["events", "🎉 Events", "Holiday dinners and potlucks."],
  ["budget", "💲 Prices and budget", "What recipes and the list cost, from pantry prices."],
  ["nutrition", "🥗 Nutrition estimates", "Calories and more per serving (USDA)."],
  ["email", "✉️ Email", "Emailing recipes and the list."],
  ["share", "🔗 Share links", "Read-only links for people without an account. Off also stops links already shared."],
  ["kids", "🧒 Kids can help", "Which steps a kid can do, in cook mode and on Recipes."],
];

export function renderFeatures(box, s) {
  box.innerHTML = `<form class="card space-y-3">
    <h2 class="font-semibold">Features</h2>
    <p class="text-sm text-slate-400">Turn off what your family doesn't use, so RecipeBank shows less. It's hidden for everyone;
      nothing is deleted, and turning it back on brings it all back. Each person can also simplify just their own view
      (for example, a plan with only dinner) on <a href="#/profile" class="underline">Me</a>.</p>
    <div class="grid gap-2 sm:grid-cols-2">${FEATURES.map(([k, label, help]) => `
      <label class="flex min-w-0 items-start gap-3 rounded-lg border border-slate-800 p-3">
        <input type="checkbox" data-feature="${k}" class="mt-1" ${s["feature_" + k] === "false" ? "" : "checked"}>
        <span class="min-w-0"><span class="block font-medium">${label}</span><span class="block text-xs text-slate-400">${help}</span></span>
      </label>`).join("")}</div>
    <button class="btn-primary">Save</button></form>`;
  const form = $("form", box);
  form.onsubmit = (e) => {
    e.preventDefault();
    const body = Object.fromEntries($$("[data-feature]", form).map((c) => ["feature_" + c.dataset.feature, c.checked ? "true" : "false"]));
    attempt(async () => { await put("/api/admin/settings", body); await refreshInfo(); }, "Saved");
  };
}
