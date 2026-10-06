// "What can I make?" results: recipes needing least first, what each one is
// missing, swaps already in the kitchen, and "Substitute" for the rest.
import { post } from "./api.js";
import { $, $$, esc, attempt, busy, toast } from "./ui.js";
import { verdictChips } from "./verdicts.js";
import { askIdeas, ideasBox, wireIdeas } from "./substitute.js";

function card(res, ri, open) {
  const c = res.card;
  const swaps = Object.fromEntries((res.swaps || []).map((s) => [s.missing, s.use]));
  const missing = res.missing.map((m, k) => {
    const key = `${ri}:${k}`;
    const use = swaps[m] || [];
    return `<li class="space-y-1"><div class="flex flex-wrap items-center gap-2">
      <span class="min-w-0 flex-1 basis-32 break-words">No <b>${esc(m)}</b>${use.length ? ` <span class="text-emerald-300">· use ${use.map((u) => esc(u.to)).join(" or ")} (you have it)</span>` : ""}</span>
      <button type="button" data-sub="${key}" class="btn-ghost min-h-0 py-1">↔ Substitute</button></div>
      ${open.has(key) ? ideasBox(key, open.get(key)) : ""}</li>`;
  }).join("");
  return `<div class="card min-w-0 space-y-2">
    <a href="#/recipe/${c.id}" class="block break-words text-lg font-semibold hover:underline">${esc(c.title)}</a>
    <p class="text-sm ${res.missing.length ? "text-slate-300" : "text-emerald-300"}">${res.missing.length
      ? `You have ${res.have} of ${res.total} things` : "✓ You have everything"}</p>
    ${res.uses_soon.length ? `<p class="text-sm text-amber-300">Uses up: ${res.uses_soon.map(esc).join(", ")}</p>` : ""}
    <div class="flex flex-wrap gap-1">${verdictChips(c.verdicts)}</div>
    ${missing ? `<ul class="space-y-1 text-sm">${missing}</ul>
      <button type="button" data-shopmiss="${ri}" class="btn-ghost">🛒 Add what's missing to the list</button>` : ""}</div>`;
}

// showResults draws the answer into box; have() and who() give what's on hand
// and who's eating, for "Substitute".
export function showResults(box, answer, have, who) {
  const open = new Map();
  let only = false;
  const draw = () => {
    const list = answer.results.map((res, ri) => ({ res, ri })).filter(({ res }) => !only || !res.missing.length);
    box.innerHTML = `<div class="mb-3 flex flex-wrap items-center gap-2">
        <h2 class="mr-auto text-lg font-semibold">${answer.results.length ? `${answer.results.length} recipe${answer.results.length === 1 ? "" : "s"} use what you have` : "No recipes use these yet"}</h2>
        ${answer.results.length ? `<label class="toggle"><input type="checkbox" id="only" ${only ? "checked" : ""}> Only what I have</label>` : ""}</div>
      ${list.length ? `<div class="grid gap-3 sm:grid-cols-2">${list.map(({ res, ri }) => card(res, ri, open)).join("")}</div>`
        : answer.results.length ? `<p class="card text-sm text-slate-400">Nothing can be made with only what you have. Turn off "Only what I have" to see what's close.</p>` : ""}`;
    const o = $("#only", box);
    if (o) o.onchange = () => { only = o.checked; draw(); };
    $$("[data-sub]", box).forEach((b) => (b.onclick = () => attempt(async () => {
      const key = b.dataset.sub;
      if (open.has(key)) open.delete(key);
      else open.set(key, await ideasFor(key, false));
      draw();
    })));
    $$("[data-shopmiss]", box).forEach((b) => (b.onclick = () => attempt(() => busy(b, "Adding…", async () => {
      const res = answer.results[Number(b.dataset.shopmiss)];
      for (const m of res.missing) await post("/api/shopping", { text: m, area: "kitchen" });
      toast(`Added ${res.missing.length} to the shopping list`);
    }))));
    wireIdeas(box, "kitchen", async (key) => { open.set(key, { ...(await ideasFor(key, true)), asked: true }); draw(); });
  };
  const ideasFor = (key, ai) => {
    const [ri, k] = key.split(":").map(Number);
    const res = answer.results[ri];
    return askIdeas(res.card.id, res.missing_lines[k], ai, who(), have());
  };
  draw();
}
