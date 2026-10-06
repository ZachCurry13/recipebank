// Smart search: a plain-words question about the recipes, answered by the
// AI (when set up) and RecipeBank's own rules.
import { post, qs } from "./api.js";
import { esc } from "./ui.js";
import { cardGrid } from "./library.js";

// askRecipes shows the answer to q in box; back() returns to the Library.
export async function askRecipes(box, area, q, who, diets, back) {
  box.innerHTML = `<p class="text-slate-400">Thinking about “${esc(q)}”…</p>`;
  let res;
  try {
    res = await post("/api/search" + qs({ who: who.join(",") || "0" }), { q, area });
  } catch (e) {
    box.innerHTML = `<div class="box-danger">${esc(e.message)}</div>`;
    return;
  }
  const label = Object.fromEntries((diets || []).map((d) => [d.key, d.label]));
  const rules = [...res.rules.diets.map((d) => label[d] || d), res.rules.max_min ? `${res.rules.max_min} min or less` : "",
    res.rules.everyone ? "OK for everyone picked" : ""].filter(Boolean);
  box.innerHTML = `
    <div class="mb-3 flex flex-wrap items-center gap-2">
      <h2 class="mr-auto min-w-0 break-words font-semibold">✨ “${esc(q)}”</h2>
      <button id="ask-back" class="btn-ghost">✕ Back to all recipes</button></div>
    ${rules.length ? `<p class="mb-2 flex flex-wrap gap-1 text-xs">${rules.map((r) => `<span class="chip-ok">✓ ${esc(r)}</span>`).join("")}</p>` : ""}
    <p class="mb-3 text-xs text-slate-500">${res.used_ai ? "Picked by the AI, then checked by RecipeBank's own rules." : "Matched by words and rules (set up the AI under Admin for answers by meaning)."}
      ${res.ai_error ? ` The AI didn't answer: ${esc(res.ai_error)}` : ""}</p>
    ${res.picks.length ? cardGrid(res.picks, area) : `<div class="card text-center text-slate-400">No recipe fits that yet.</div>`}
    ${res.left_out.length ? `<details class="mt-3 text-sm text-slate-400"><summary class="cursor-pointer">Left out by the rules (${res.left_out.length})</summary>
      <ul class="mt-1 space-y-0.5">${res.left_out.map((l) => `<li><a href="#/recipe/${l.id}" class="underline">${esc(l.title)}</a>: ${esc(l.reason)}</li>`).join("")}</ul></details>` : ""}`;
  // Each pick's reason goes under its title.
  for (const p of res.picks) {
    if (!p.why) continue;
    const h = box.querySelector(`a[href="#/recipe/${p.id}"] h2`);
    if (h) h.insertAdjacentHTML("afterend", `<p class="text-xs text-emerald-300">${esc(p.why)}</p>`);
  }
  box.querySelector("#ask-back").onclick = back;
}
