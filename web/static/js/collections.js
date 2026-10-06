// Collections (themed shelves, by hand or with the AI's suggestions) and the
// seasonal shelves that come round each year.
import { get, post, put, del } from "./api.js";
import { $, $$, esc, attempt, busy, canManage, sheet, toast } from "./ui.js";
import { cardGrid } from "./library.js";
import { localDate } from "./planpick.js";
import { go } from "./app.js";

export async function renderCollections(view, params, state) {
  const manage = canManage(state.user);
  const data = await get(`/api/collections?date=${localDate()}`);
  const tile = (href, icon, name, sub) => `<a href="${href}" class="card flex min-w-0 items-center gap-3 hover:ring-emerald-700">
    <span class="text-3xl">${esc(icon)}</span><span class="min-w-0"><span class="block break-words font-semibold">${esc(name)}</span>
    <span class="text-xs text-slate-400">${sub}</span></span></a>`;
  view.innerHTML = `
    <div class="mb-4 flex flex-wrap items-end gap-3">
      <div class="min-w-0 basis-full sm:basis-auto sm:flex-1"><h1 class="text-2xl font-bold">📚 Collections</h1>
        <p class="text-sm text-slate-400">Themed shelves: fill them by hand, or describe them and let the AI suggest recipes.</p></div>
      <a href="#/cookbook" class="btn-secondary">📖 Family cookbook</a>
      ${manage ? `<button id="new" class="btn-primary">➕ New collection</button>` : ""}
    </div>
    ${data.seasons.length ? `<h2 class="label">In season now</h2><div class="mb-6 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">${data.seasons.map((s) =>
      tile(`#/season/${s.key}`, s.icon, s.name, `${s.count} recipe${s.count === 1 ? "" : "s"}`)).join("")}</div>` : ""}
    <h2 class="label">Your collections</h2>
    ${data.collections.length ? `<div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">${data.collections.map((c) =>
      tile(`#/collection/${c.id}`, c.icon, c.name, `${c.count} recipe${c.count === 1 ? "" : "s"}${c.area === "home" ? " · Home & Care" : ""}`)).join("")}</div>`
      : `<div class="card text-center text-slate-400">No collections yet.${manage ? " Make one for weeknights, holidays or the kids' favorites." : ""}</div>`}`;
  const btn = $("#new", view);
  if (btn) btn.onclick = () => editCollection(null);
}

function editCollection(c) {
  c = c || { id: 0, name: "", description: "", icon: "📚", area: "kitchen" };
  const d = sheet(`<form id="cf" class="space-y-3">
    <div class="flex items-center"><h2 class="text-lg font-semibold">${c.id ? "Edit collection" : "New collection"}</h2>
      <button type="button" data-close class="btn-ghost ml-auto">✕</button></div>
    <div class="flex gap-2"><label class="block w-20"><span class="label">Icon</span><input name="icon" maxlength="4" class="input text-center" value="${esc(c.icon)}"></label>
      <label class="block flex-1"><span class="label">Name</span><input name="name" required maxlength="80" class="input" value="${esc(c.name)}" placeholder="Quick weeknights"></label></div>
    <label class="block"><span class="label">What belongs (the AI reads this)</span><textarea name="description" rows="3" maxlength="500" class="input"
      placeholder="Dinners under 30 minutes that everyone can eat, no dairy">${esc(c.description)}</textarea></label>
    ${c.id ? "" : `<div class="flex gap-2">${[["kitchen", "🍲 Kitchen"], ["home", "🧴 Home & Care"]].map(([a, l]) =>
      `<label class="pick ${c.area === a ? "on" : ""}"><input type="radio" name="area" value="${a}" class="sr-only" ${c.area === a ? "checked" : ""}>${l}</label>`).join("")}</div>`}
    <button class="btn-primary">Save</button></form>`);
  $$('input[name="area"]', d).forEach((i) => (i.onchange = () => $$('input[name="area"]', d).forEach((j) => j.parentElement.classList.toggle("on", j.checked))));
  $("#cf", d).onsubmit = (e) => {
    e.preventDefault();
    const f = e.target;
    const body = { name: f.name.value, description: f.description.value, icon: f.icon.value, area: f.area?.value || c.area };
    attempt(async () => {
      const { id } = c.id ? await put(`/api/collections/${c.id}`, body) : await post("/api/collections", body);
      d.close();
      go(`#/collection/${c.id || id}`);
    });
  };
}

export async function renderCollection(view, params, state) {
  const manage = canManage(state.user);
  const data = await get(`/api/collections/${params.id}`);
  const c = data.collection;
  const reload = () => renderCollection(view, params, state);
  view.innerHTML = `
    <a href="#/collections" class="text-sm text-slate-400 hover:text-slate-200">‹ Collections</a>
    <div class="mb-4 mt-2 flex flex-wrap items-end gap-3">
      <div class="min-w-0 basis-full sm:basis-auto sm:flex-1"><h1 class="break-words text-2xl font-bold">${esc(c.icon)} ${esc(c.name)}</h1>
        ${c.description ? `<p class="text-sm text-slate-400">${esc(c.description)}</p>` : ""}</div>
      ${data.recipes.length && c.area === "kitchen" ? `<a href="#/cookbook?collection=${c.id}" class="btn-secondary">📖 Print as a cookbook</a>` : ""}
      ${manage ? `<button id="suggest" class="btn-primary">✨ Suggest recipes</button>
        <button id="edit" class="btn-secondary">✎ Edit</button><button id="rm" class="btn-ghost text-rose-300">🗑 Delete</button>` : ""}
    </div>
    ${data.recipes.length ? cardGrid(data.recipes, c.area, manage) : `<div class="card text-center text-slate-400">Nothing here yet.
      ${manage ? "Use ✨ Suggest recipes, or add recipes from their page." : ""}</div>`}`;
  $$("[data-remove]", view).forEach((b) => (b.onclick = () => attempt(async () => {
    await del(`/api/collections/${c.id}/recipes/${b.dataset.remove}`);
    reload();
  })));
  if (!manage) return;
  $("#edit", view).onclick = () => editCollection(c);
  $("#rm", view).onclick = () => {
    if (confirm(`Delete the collection "${c.name}"? Its recipes stay in RecipeBank.`)) attempt(async () => { await del(`/api/collections/${c.id}`); go("#/collections"); });
  };
  $("#suggest", view).onclick = (e) => attempt(() => busy(e.currentTarget, "Thinking…", async () => {
    const sug = await post(`/api/collections/${c.id}/suggest`);
    showSuggestions(c, sug, reload);
  }));
}

function showSuggestions(c, sug, done) {
  const d = sheet(`<div class="space-y-3">
    <div class="flex items-center"><h2 class="text-lg font-semibold">Suggestions for ${esc(c.name)}</h2>
      <button type="button" data-close class="btn-ghost ml-auto">✕</button></div>
    <p class="text-xs text-slate-400">${sug.used_ai ? "Picked by the AI, then checked by RecipeBank's own rules." : "Recipes that share words with the description (set up the AI under Admin for smarter picks)."}
      ${sug.ai_error ? ` The AI didn't answer: ${esc(sug.ai_error)}` : ""}</p>
    ${sug.picks.length ? `<ul class="space-y-1">${sug.picks.map((p) => `<li><label class="flex min-w-0 items-start gap-2 rounded-lg p-2 hover:bg-slate-800">
      <input type="checkbox" checked value="${p.id}" class="mt-1 h-5 w-5 accent-emerald-600"><span class="min-w-0"><span class="block break-words font-medium">${esc(p.title)}</span>
      ${p.why ? `<span class="text-xs text-slate-400">${esc(p.why)}</span>` : ""}</span></label></li>`).join("")}</ul>
      <button id="add" class="btn-primary">Add the ticked ones</button>` : `<p class="text-sm text-slate-300">Nothing more fits yet.</p>`}
    ${sug.left_out.length ? `<details class="text-sm text-slate-400"><summary class="cursor-pointer">Left out by the rules (${sug.left_out.length})</summary>
      <ul class="mt-1 space-y-0.5">${sug.left_out.map((l) => `<li>${esc(l.title)}: ${esc(l.reason)}</li>`).join("")}</ul></details>` : ""}</div>`);
  const add = $("#add", d);
  if (add) add.onclick = () => attempt(async () => {
    const ids = $$("input[type=checkbox]:checked", d).map((i) => Number(i.value));
    await post(`/api/collections/${c.id}/recipes`, { ids });
    d.close();
    toast(`Added ${ids.length}`);
    done();
  });
}

export async function renderSeason(view, params) {
  const data = await get(`/api/seasons/${params.id}`);
  const s = data.season;
  view.innerHTML = `
    <a href="#/collections" class="text-sm text-slate-400 hover:text-slate-200">‹ Collections</a>
    <h1 class="mb-1 mt-2 break-words text-2xl font-bold">${esc(s.icon)} ${esc(s.name)}</h1>
    <p class="mb-4 text-sm text-slate-400">${esc(data.theme)}</p>
    ${data.recipes.length ? cardGrid(data.recipes, s.area) : `<div class="card text-center text-slate-400">No recipes for this season yet.</div>`}`;
}

// addToCollection lets a parent put a recipe on collections from its page.
export async function addToCollection(recipeId, area) {
  const data = await get("/api/collections");
  const list = data.collections.filter((c) => c.area === area);
  const d = sheet(`<div class="space-y-3">
    <div class="flex items-center"><h2 class="text-lg font-semibold">Add to a collection</h2><button type="button" data-close class="btn-ghost ml-auto">✕</button></div>
    ${list.length ? `<div class="grid gap-2">${list.map((c) => `<button data-col="${c.id}" class="card flex items-center gap-2 text-left">
      <span class="text-2xl">${esc(c.icon)}</span><span class="min-w-0 break-words">${esc(c.name)}</span></button>`).join("")}</div>`
      : `<p class="text-sm text-slate-400">No collections yet. Make one on the Collections page.</p>`}</div>`);
  $$("[data-col]", d).forEach((b) => (b.onclick = () => attempt(async () => {
    await post(`/api/collections/${b.dataset.col}/recipes`, { ids: [recipeId] });
    d.close();
    toast("Added");
  })));
}
