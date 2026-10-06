// A recipe's page: who can eat it and why, swaps, Home & Care warnings,
// servings and units, steps, and the parents' tools.
import { get, post, put, del, qs } from "./api.js";
import { $, $$, esc, attempt, busy, fmtMin, peppers, levelChip, canManage, cap, toast, sheet } from "./ui.js";
import { verdictList } from "./verdicts.js";
import { startCooking } from "./cook.js";
import { ingredientList, stepList } from "./recipeparts.js";
import { cardBox } from "./cardcheck.js";
import { readAgain } from "./readagain.js";
import { cookedSheet, cookHistory, wireHistory } from "./cooked.js";
import { nutritionCard, wireNutrition } from "./nutrition.js";
import { subButton, wireSubstitute } from "./substitute.js";
import { go } from "./app.js";
import { addToCollection } from "./collections.js";

const HAZARD_BOX = { danger: "box-danger", caution: "box-caution", info: "box-info" };
const HAZARD_ICON = { danger: "⛔", caution: "⚠️", info: "ℹ️" };

// Allergen names for this page ("treenut" → "tree nuts").
let NAMES = {};
const nm = (k) => NAMES[k] || k;

export async function renderRecipe(view, params, state) {
  NAMES = Object.fromEntries((state.info?.all_allergens || []).map((a) => [a.key, a.label.toLowerCase()]));
  const people = await get("/api/people");
  const view_ = { who: people.filter((p) => !p.is_guest).map((p) => p.id), factor: 1, servings: 0,
    system: state.user.units || state.info?.default_units || "us", sub: { on: false, open: new Map() } };
  let data;
  const load = async () => {
    data = await get(`/api/recipes/${params.id}` + qs({ who: view_.who.join(",") || "0" }));
    view_.sub.open.clear(); // ideas depend on who's eating
    if (!view_.servings) view_.servings = data.recipe.servings || 0;
    draw();
  };
  const draw = () => {
    const r = data.recipe;
    const home = r.area === "home";
    const manage = canManage(state.user);
    view.innerHTML = `
      <a href="#/${r.area}" class="text-sm text-slate-400 hover:text-slate-200">‹ ${home ? "Home & Care" : "Kitchen"}</a>
      <div class="mt-2 grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.3fr)]">
        <div class="min-w-0 space-y-4">
          ${r.photo ? `<img src="/api/photos/${esc(r.photo)}?w=1200" alt="" class="max-h-80 w-full rounded-xl object-cover">` : ""}
          <div>
            <h1 class="break-words text-2xl font-bold leading-tight">${esc(r.title)}</h1>
            ${r.summary ? `<p class="mt-1 text-slate-300">${esc(r.summary)}</p>` : ""}
            ${data.original ? `<p class="mt-1 text-sm text-slate-400">Our version of <a class="underline" href="#/recipe/${data.original.id}">${esc(data.original.title)}</a></p>` : ""}
            <div class="mt-2 flex flex-wrap items-center gap-1">
              ${r.total_min ? `<span class="chip-info">⏱ ${fmtMin(r.total_min)}${r.prep_min ? ` (prep ${fmtMin(r.prep_min)})` : ""}</span>` : ""}
              ${home ? "" : peppers(data.heat) + levelChip(data.level)}
              ${r.course ? `<span class="chip-cat">${esc(cap(r.course))}</span>` : ""}
              ${r.cuisine ? `<span class="chip-cat">${esc(r.cuisine)}</span>` : ""}
              ${r.protein ? `<span class="chip-cat">${esc(cap(r.protein))}</span>` : ""}
            </div>
            <div class="mt-2 flex items-center gap-1" aria-label="Rating">${[1, 2, 3, 4, 5].map((n) =>
              `<button data-star="${n}" class="px-0.5 text-2xl ${n <= r.rating ? "text-amber-300" : "text-slate-600"}" title="${n} star${n > 1 ? "s" : ""}">★</button>`).join("")}</div>
          </div>
          <div class="card space-y-3">
            <h2 class="font-semibold">${home ? "Who's using it?" : "Who's eating?"}</h2>
            <div class="flex flex-wrap gap-2">${people.map((p) =>
              `<button class="pick ${view_.who.includes(p.id) ? "on" : ""}" data-who="${p.id}">${esc(p.name)}</button>`).join("")}</div>
            ${verdictList(data.verdicts, r.ingredients)}
          </div>
          ${data.hazards.length ? `<div class="space-y-2"><h2 class="font-semibold">Before you make it</h2>${data.hazards.map((h) =>
            `<div class="${HAZARD_BOX[h.level]}">${HAZARD_ICON[h.level]} ${esc(h.text)}</div>`).join("")}</div>` : ""}
          <div class="flex flex-wrap gap-2">
            ${r.steps.length ? `<button id="cook" class="btn-primary">▶ ${home ? "Step by step" : "Start cooking"}</button>` : ""}
            ${home ? "" : `<button id="cooked" class="btn-secondary">🍳 We cooked it</button>`}
            <button id="print" class="btn-secondary">🖨 Print</button>
            ${r.ingredients.length ? `<button id="shop" class="btn-secondary">🛒 Add to shopping list</button>` : ""}
            ${manage ? `<a href="#/edit/${r.id}" class="btn-secondary">✎ Edit</a>
              <button id="version" class="btn-secondary" title="A copy to change, keeping this one">⎘ Make our version</button>
              <button id="tocol" class="btn-secondary">📚 Add to a collection</button>
              <button id="delete" class="btn-ghost text-rose-300">🗑 Delete</button>` : ""}
          </div>
        </div>
        <div class="min-w-0 space-y-4">
          <div class="card">
            <div class="mb-3 flex flex-wrap items-center gap-2">
              <h2 class="mr-auto font-semibold">Ingredients</h2>
              ${scaler(r, view_)}
              <div class="flex rounded-lg ring-1 ring-slate-700" role="group" aria-label="Units">
                ${["us", "metric"].map((s) => `<button data-sys="${s}" class="px-3 py-1.5 text-sm ${view_.system === s ? "rounded-lg bg-slate-700 text-white" : "text-slate-400"}">${s === "us" ? "US" : "Metric"}</button>`).join("")}
              </div>
              ${r.ingredients.length ? subButton(view_.sub.on) : ""}
            </div>
            ${r.needs_review ? `<div class="no-print mb-3">${cardBox(r, data.card_check, manage ? "page" : "")}</div>` : ""}
            <ul class="space-y-0.5">${ingredientList(r, data, manage, view_, nm)}</ul>
          </div>
          <div class="card">
            <h2 class="mb-3 font-semibold">Steps</h2>
            ${view_.factor !== 1 ? `<p class="mb-2 text-xs text-slate-500">Amounts in the steps are for the recipe as written (${r.servings ? `${+r.servings} servings` : "×1"}).</p>` : ""}
            <ol class="space-y-3">${stepList(r, view_)}</ol>
          </div>
          ${cookHistory(data.cooks, people, manage)}
          ${home || !r.ingredients.length ? "" : nutritionCard()}
          ${r.notes ? `<div class="card"><h2 class="mb-1 font-semibold">Our notes</h2><p class="whitespace-pre-line text-sm text-slate-300">${esc(r.notes)}</p></div>` : ""}
          ${r.storage ? `<div class="card"><h2 class="mb-1 font-semibold">Storage</h2><p class="whitespace-pre-line text-sm text-slate-300">${esc(r.storage)}</p></div>` : ""}
          ${source(r, manage)}
        </div>
      </div>`;
    wire(r);
  };

  const wire = (r) => {
    $$("[data-who]", view).forEach((b) => (b.onclick = () => {
      const id = Number(b.dataset.who);
      view_.who = view_.who.includes(id) ? view_.who.filter((x) => x !== id) : [...view_.who, id];
      attempt(load);
    }));
    $$("[data-sys]", view).forEach((b) => (b.onclick = () => { view_.system = b.dataset.sys; draw(); }));
    $("#print", view).onclick = () => window.print();
    const cooked = $("#cooked", view);
    if (cooked) cooked.onclick = () => attempt(() => cookedSheet(r, load));
    wireHistory(view, load);
    wireNutrition(view, r, canManage(state.user));
    const reread = $("[data-reread]", view);
    if (reread) reread.onclick = () => attempt(() => readAgain(r, load));
    const cardOK = $("[data-cardok]", view);
    if (cardOK) cardOK.onclick = () => attempt(() => busy(cardOK, "Saving…", async () => {
      await post(`/api/recipes/${r.id}/card-checked`, {});
      await load();
      toast("Marked as checked");
    }));
    $$("[data-serv]", view).forEach((b) => (b.onclick = () => {
      const base = r.servings || 1;
      if (r.servings) {
        view_.servings = Math.max(1, view_.servings + Number(b.dataset.serv));
        view_.factor = view_.servings / base;
      } else {
        const i = MULTS.indexOf(view_.factor);
        view_.factor = MULTS[Math.max(0, Math.min(MULTS.length - 1, (i < 0 ? 2 : i) + Number(b.dataset.serv)))];
      }
      draw();
    }));
    $$("[data-star]", view).forEach((b) => (b.onclick = () => attempt(async () => {
      const n = Number(b.dataset.star);
      r.rating = r.rating === n ? 0 : n;
      await put(`/api/recipes/${r.id}/rating`, { stars: r.rating });
      draw();
    })));
    $$("[data-label]", view).forEach((b) => (b.onclick = () => attempt(async () => {
      const [i, allergen, on] = b.dataset.label.split(":");
      data = await put(`/api/recipes/${r.id}/label` + qs({ who: view_.who.join(",") || "0" }),
        { ingredient: Number(i), allergen, checked: on === "1" });
      draw();
    })));
    const tocol = $("#tocol", view);
    if (tocol) tocol.onclick = () => attempt(() => addToCollection(r.id, r.area));
    const shop = $("#shop", view);
    if (shop) shop.onclick = () => attempt(() => addToShopping(r, view_.factor));
    $$("[data-ing]", view).forEach((li) => (li.onclick = (e) => { if (!e.target.closest("button")) li.classList.toggle("line-through-soft"); }));
    wireSubstitute(view, r, view_.sub, () => view_.who.join(",") || "0", draw);
    const cook = $("#cook", view);
    if (cook) cook.onclick = () => startCooking(r, view_);
    const version = $("#version", view);
    if (version) version.onclick = () => attempt(async () => {
      const { id } = await post(`/api/recipes/${r.id}/version`);
      go(`#/edit/${id}`);
    });
    const delBtn = $("#delete", view);
    if (delBtn) delBtn.onclick = () => {
      if (!confirm(`Delete "${r.title}"? This can't be undone.`)) return;
      attempt(async () => {
        await del(`/api/recipes/${r.id}`);
        toast("Deleted");
        go(`#/${r.area}`);
      });
    };
  };

  await load();
}

// Recipes without a servings count scale by these steps instead.
const MULTS = [0.25, 0.5, 1, 1.5, 2, 3, 4];

function scaler(r, v) {
  const label = r.servings ? `${v.servings} serving${v.servings === 1 ? "" : "s"}` : `× ${v.factor}`;
  return `<div class="flex items-center gap-1 text-sm">
    <button data-serv="-1" class="btn-secondary min-h-0 px-3 py-1" aria-label="Fewer">−</button>
    <span class="min-w-[5.5rem] text-center">${label}</span>
    <button data-serv="1" class="btn-secondary min-h-0 px-3 py-1" aria-label="More">+</button></div>`;
}

function source(r, manage) {
  const parts = [];
  if (r.source_url) parts.push(`<a class="break-all underline" href="${esc(r.source_url)}" target="_blank" rel="noopener noreferrer">${esc(r.source_url)}</a>`);
  if (r.source_note) parts.push(esc(r.source_note));
  const photos = (r.source_photos || []).map((p) => `<a href="/api/photos/${esc(p)}" target="_blank" rel="noopener"><img src="/api/photos/${esc(p)}?w=480" alt="The original" class="h-28 rounded-lg object-cover"></a>`).join("");
  if (!parts.length && !photos) return "";
  return `<div class="card space-y-2"><h2 class="font-semibold">Where it's from</h2>${parts.map((p) => `<p class="text-sm text-slate-300">${p}</p>`).join("")}
    ${photos ? `<div class="flex flex-wrap gap-2">${photos}</div>` : ""}
    ${r.read_by ? `<p class="text-xs text-slate-500">Read from the photo by ${esc(r.read_by)}.</p>` : ""}
    ${manage && photos ? `<button type="button" data-reread class="btn-secondary">↻ Read the card again</button>` : ""}
    <p class="text-xs text-slate-500">Kept for the family's own use.</p></div>`;
}

// addToShopping puts the recipe (at this scale) on the list, and says what
// the pantry probably has already.
async function addToShopping(r, factor) {
  const res = await post("/api/shopping/recipe", { id: r.id, factor });
  if (!res.have.length) return toast(`Added ${res.added} thing${res.added === 1 ? "" : "s"} to the shopping list`);
  const d = sheet(`<div class="space-y-3">
    <div class="flex items-center"><h2 class="text-lg font-semibold">Added ${res.added} to the shopping list</h2>
      <button type="button" data-close class="btn-ghost ml-auto">✕</button></div>
    <p class="text-sm text-slate-300">These look like they're in the house already, so they weren't added:</p>
    <ul class="space-y-1 text-sm">${res.have.map((h, i) => `<li class="flex flex-wrap items-center gap-2"><span class="min-w-0 flex-1 basis-40 break-words">${esc(h.line)}
      <span class="block text-xs text-slate-500">${h.area === "home" ? "Supplies" : "Pantry"}: ${esc(h.stock)}</span></span>
      <button data-have="${i}" class="btn-ghost min-h-0 py-1">+ Add anyway</button></li>`).join("")}</ul>
    <a href="#/shopping" class="btn-primary" data-close>Open the list</a></div>`);
  $$("[data-have]", d).forEach((b) => (b.onclick = () => attempt(async () => {
    await post("/api/shopping", { text: res.have[Number(b.dataset.have)].line });
    b.replaceWith(Object.assign(document.createElement("span"), { className: "text-xs text-emerald-300", textContent: "✓ Added" }));
  })));
}
