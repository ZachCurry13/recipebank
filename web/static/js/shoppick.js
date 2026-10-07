// Recipe → "🛒 Add to list": tick which ingredients to buy. What the pantry
// or supplies seem to have is unticked (with what it matched), so nothing is
// bought twice by accident, and anything can be ticked anyway.
import { get, post } from "./api.js";
import { $, $$, esc, attempt, busy, sheet, toast } from "./ui.js";

export async function pickForList(r, factor) {
  const { have } = await get(`/api/shopping/recipe/${r.id}`);
  const d = sheet(`<div class="space-y-3">
    <div class="flex items-center"><h2 class="text-lg font-semibold">🛒 Add to the shopping list</h2>
      <button type="button" data-close class="btn-ghost ml-auto" aria-label="Close">✕</button></div>
    <p class="text-sm text-slate-400">Tick what to buy${factor !== 1 ? ` (amounts ×${+factor.toFixed(2)}, as scaled)` : ""}.
      What the house seems to have is unticked.</p>
    <div class="flex gap-3 text-sm"><button type="button" data-all class="underline">Tick all</button>
      <button type="button" data-none class="underline">Untick all</button></div>
    <ul class="space-y-1">${r.ingredients.map((ing, i) => `<li><label class="flex items-start gap-2 rounded-lg px-1 py-1">
      <input type="checkbox" data-line="${i}" class="mt-1" ${have[i] ? "" : "checked"}>
      <span class="min-w-0 break-words">${esc(ing.line)}${have[i] ? `<span class="block text-xs text-slate-500">${have[i].area === "home" ? "Supplies" : "Pantry"}: ${esc(have[i].stock)}</span>` : ""}</span>
    </label></li>`).join("")}</ul>
    <button type="button" data-add class="btn-primary w-full"></button></div>`);
  const boxes = $$("[data-line]", d);
  const add = $("[data-add]", d);
  const count = () => {
    const n = boxes.filter((b) => b.checked).length;
    add.textContent = n ? `Add ${n} to the list` : "Nothing ticked";
    add.disabled = !n;
  };
  boxes.forEach((b) => (b.onchange = count));
  $("[data-all]", d).onclick = () => { boxes.forEach((b) => (b.checked = true)); count(); };
  $("[data-none]", d).onclick = () => { boxes.forEach((b) => (b.checked = false)); count(); };
  count();
  add.onclick = () => attempt(() => busy(add, "Adding…", async () => {
    const only = boxes.filter((b) => b.checked).map((b) => Number(b.dataset.line));
    const res = await post("/api/shopping/recipe", { id: r.id, factor, only });
    d.close();
    toast(`Added ${res.added} to the shopping list`);
  }));
}
