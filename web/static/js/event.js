// One event: who's coming (and how many dishes each can eat), the menu with
// every dish checked for everyone, who brings what, and the family's dishes
// to the shopping list for everyone coming.
import { get, post, put, del, qs } from "./api.js";
import { $, $$, esc, attempt, busy, canManage, sheet, toast } from "./ui.js";
import { verdictChips } from "./verdicts.js";
import { dayLabel } from "./planpick.js";
import { editEvent } from "./events.js";
import { go } from "./app.js";

export async function renderEvent(view, params, state) {
  const manage = canManage(state.user);
  const base = `/api/events/${params.id}`;
  let data = await get(base);
  const reload = async () => { data = await get(base); draw(); };
  // Changes show at once and are saved one after another, so quick taps
  // each build on the last.
  let saving = Promise.resolve();
  const saveWho = (patch) => {
    Object.assign(data.event, patch);
    draw();
    saving = saving.then(() => attempt(async () => { await put(base, data.event); await reload(); }));
  };

  const draw = () => {
    const e = data.event;
    const coming = new Set(e.who);
    view.innerHTML = `
      <a href="#/events" class="text-sm text-slate-400 hover:text-slate-200">‹ Events</a>
      <div class="mb-4 mt-2 flex flex-wrap items-end gap-3">
        <div class="min-w-0 basis-full sm:basis-auto sm:flex-1"><h1 class="break-words text-2xl font-bold">🎉 ${esc(e.name)}</h1>
          <p class="text-sm text-slate-400">${e.date ? esc(dayLabel(e.date, true)) : "No date yet"} · ${data.headcount} coming</p>
          ${e.notes ? `<p class="mt-1 whitespace-pre-line text-sm text-slate-300">${esc(e.notes)}</p>` : ""}</div>
        <button id="print" class="btn-secondary">🖨 Print</button>
        ${manage ? `<button id="edit" class="btn-secondary">✎ Edit</button><button id="rm" class="btn-ghost text-rose-300">🗑 Delete</button>` : ""}
      </div>
      <div class="grid gap-4 lg:grid-cols-2">
        <div class="card min-w-0 space-y-3">
          <h2 class="font-semibold">Who's coming</h2>
          <div class="no-print flex flex-wrap gap-2">${data.people.map((p) => `<button type="button" data-who="${p.id}" ${manage ? "" : "disabled"}
            class="pick ${coming.has(p.id) ? "on" : ""}">${esc(p.name)}${p.is_guest ? " (guest)" : ""}</button>`).join("")}</div>
          <label class="no-print flex flex-wrap items-center gap-2 text-sm text-slate-300">More guests without a profile
            <input id="extra" type="number" min="0" max="500" class="input w-24" value="${e.extra || 0}" ${manage ? "" : "disabled"}></label>
          ${data.coming.length ? `<ul class="space-y-1 text-sm">${data.coming.map((p) => `<li class="flex flex-wrap items-center gap-2">
            <span class="min-w-0 break-words font-medium">${esc(p.name)}</span>
            ${p.ok ? `<span class="chip-ok">✓ ${p.ok} dish${p.ok === 1 ? "" : "es"}</span>` : `<span class="chip-no">Nothing they can eat yet</span>`}
            ${p.unsure ? `<span class="chip-unsure">⚠ ${p.unsure} not sure</span>` : ""}</li>`).join("")}</ul>` : ""}
          <p class="text-xs text-slate-500">Add guests with their allergies and diets on the Family page first.</p>
        </div>
        <div class="card min-w-0 space-y-3">
          <h2 class="font-semibold">Menu</h2>
          ${data.dishes.length ? `<ul class="space-y-3">${data.dishes.map((d) => `<li class="space-y-1">
            <div class="flex flex-wrap items-center gap-2">
              <span class="min-w-0 flex-1 basis-40 break-words font-medium">${d.recipe ? `<a href="#/recipe/${d.recipe.id}" class="hover:underline">${esc(d.recipe.title)}</a>` : esc(d.title || "(recipe removed)")}</span>
              <span class="ml-auto flex shrink-0 items-center gap-1"><span class="text-xs text-slate-400">${d.brings ? `Brought by ${esc(d.brings)}` : "We make it"}</span>
              ${manage ? `<button type="button" data-brings="${d.id}" class="no-print btn-ghost min-h-0 px-2 py-1 text-sm" aria-label="Who brings it">✎</button>
                <button type="button" data-rmdish="${d.id}" class="no-print btn-ghost min-h-0 px-2 py-1 text-slate-500" aria-label="Take off the menu">✕</button>` : ""}</span></div>
            ${d.recipe ? `<div class="flex flex-wrap gap-1">${verdictChips(d.verdicts)}</div>`
              : `<p class="text-xs text-slate-500">No recipe: check with ${d.brings ? esc(d.brings) : "the store's label"} about allergies.</p>`}</li>`).join("")}</ul>`
            : `<p class="text-sm text-slate-400">Nothing on the menu yet.</p>`}
          <div class="no-print flex flex-wrap gap-2">
            ${manage ? `<button id="add-dish" class="btn-secondary">➕ Add a dish</button>` : ""}
            ${data.dishes.some((d) => d.recipe && !d.brings) ? `<button id="shop" class="btn-ghost">🛒 Our dishes to the list (for ${data.headcount})</button>` : ""}</div>
        </div>
      </div>`;
    $("#print", view).onclick = () => window.print();
    const shop = $("#shop", view);
    if (shop) shop.onclick = () => attempt(() => busy(shop, "Adding…", async () => {
      const res = await post(base + "/shopping", {});
      toast(`Added ${res.added} to the shopping list${res.have.length ? ` (${res.have.length} already in the house)` : ""}`);
    }));
    if (!manage) return;
    $$("[data-who]", view).forEach((b) => (b.onclick = () => {
      const id = Number(b.dataset.who);
      const who = data.event.who;
      saveWho({ who: who.includes(id) ? who.filter((x) => x !== id) : [...who, id] });
    }));
    $("#extra", view).onchange = (ev) => saveWho({ extra: Math.max(0, Number(ev.target.value) || 0) });
    $("#edit", view).onclick = () => editEvent(e, reload);
    $("#rm", view).onclick = () => {
      if (confirm(`Delete "${e.name}"? The recipes stay in RecipeBank.`)) attempt(async () => { await del(base); go("#/events"); });
    };
    $$("[data-rmdish]", view).forEach((b) => (b.onclick = () => attempt(async () => { data = await del(`${base}/dishes/${b.dataset.rmdish}`); draw(); })));
    $$("[data-brings]", view).forEach((b) => (b.onclick = () => bringsSheet(data.dishes.find((d) => d.id === Number(b.dataset.brings)))));
    $("#add-dish", view).onclick = () => addDish();
  };

  const bringsSheet = (dish) => {
    const d = sheet(`<form id="bf" class="space-y-3">
      <div class="flex items-center"><h2 class="text-lg font-semibold">Who brings it?</h2>
        <button type="button" data-close class="btn-ghost ml-auto" aria-label="Close">✕</button></div>
      <input name="brings" maxlength="60" class="input" value="${esc(dish.brings)}" placeholder="Leave empty if you make it">
      <div class="flex flex-wrap gap-2"><button class="btn-primary">Save</button>
        <button type="button" id="us" class="btn-secondary">We make it</button></div></form>`);
    const save = (brings) => attempt(async () => { data = await put(`${base}/dishes/${dish.id}`, { brings }); d.close(); draw(); });
    $("#bf", d).onsubmit = (ev) => { ev.preventDefault(); save(ev.target.brings.value.trim()); };
    $("#us", d).onclick = () => save("");
  };

  const addDish = () => {
    const d = sheet(`<div class="space-y-3">
      <div class="flex items-center"><h2 class="text-lg font-semibold">Add a dish</h2>
        <button type="button" data-close class="btn-ghost ml-auto" aria-label="Close">✕</button></div>
      <input id="brings" maxlength="60" class="input" placeholder="Who brings it? (empty: you make it)">
      <input id="dq" type="search" class="input" placeholder="Search your recipes">
      <ul id="dpicks" class="max-h-[40dvh] space-y-1 overflow-y-auto"></ul>
      <form id="free" class="flex flex-wrap gap-2 border-t border-slate-800 pt-3">
        <input name="title" maxlength="120" class="input min-w-0 flex-1 basis-40" placeholder="Or a dish without a recipe: store-bought pie">
        <button class="btn-secondary">Add</button></form></div>`);
    const add = (body) => attempt(async () => {
      data = await post(`${base}/dishes`, { ...body, brings: $("#brings", d).value.trim() });
      d.close();
      draw();
    });
    let timer;
    const search = async () => {
      const res = await get("/api/recipes" + qs({ area: "kitchen", who: data.event.who.join(",") || "0", q: $("#dq", d).value }));
      $("#dpicks", d).innerHTML = res.recipes.map((r) => `<li><button type="button" data-dish="${r.id}"
        class="flex w-full min-w-0 flex-wrap items-center gap-2 rounded-lg p-2 text-left hover:bg-slate-800">
        <span class="min-w-0 flex-1 basis-40 break-words font-medium">${esc(r.title)}</span>
        <span class="flex basis-full flex-wrap gap-1">${verdictChips(r.verdicts)}</span></button></li>`).join("")
        || `<li class="p-2 text-sm text-slate-400">No recipes match.</li>`;
      $$("[data-dish]", d).forEach((b) => (b.onclick = () => add({ recipe_id: Number(b.dataset.dish) })));
    };
    $("#dq", d).oninput = () => { clearTimeout(timer); timer = setTimeout(() => attempt(search), 250); };
    $("#free", d).onsubmit = (ev) => { ev.preventDefault(); const t = ev.target.title.value.trim(); if (t) add({ title: t }); };
    attempt(search);
  };
  draw();
}
