// The Library: Kitchen or Home & Care recipes, filtered, with each diner's verdict.
import { get, qs } from "./api.js";
import { $, $$, esc, fmtMin, peppers, stars, canManage, cap, HOME_CATS, LEVELS, levelChip } from "./ui.js";
import { verdictChips } from "./verdicts.js";
import { localDate } from "./planpick.js";
import { askRecipes } from "./ask.js";

// Filters are kept per area while the app is open.
const filters = { kitchen: { who: null, ok: "" }, home: { who: null, ok: "" } };

export async function renderLibrary(view, area, params, state) {
  const f = filters[area];
  const people = await get("/api/people");
  if (f.who === null) f.who = people.filter((p) => !p.is_guest).map((p) => p.id);
  const home = area === "home";
  const title = home ? "Home & Care" : "Kitchen";
  const blurb = home
    ? "Home-made cleaners, toothpaste, mouthwash and more, with safety checks for kids, pets and surfaces."
    : "The family's recipes, checked against everyone's allergies, diets and dislikes.";
  view.innerHTML = `
    <div class="mb-4 flex flex-wrap items-end gap-3">
      <div class="min-w-0 basis-full sm:basis-auto sm:flex-1">
        <h1 class="text-2xl font-bold">${home ? "🧴" : "🍲"} ${title}</h1>
        <p class="text-sm text-slate-400">${blurb}</p>
      </div>
      ${canManage(state.user) ? `<a href="#/add?area=${area}" class="btn-primary">➕ Add a recipe</a>` : ""}
    </div>
    <div id="seasons" class="mb-3 flex flex-wrap gap-2"></div>
    <div class="card mb-4 space-y-3">
      <form id="qf" class="flex flex-wrap gap-2"><input id="q" type="search" class="input min-w-0 flex-1 basis-48" placeholder="Search, or ask: quick dairy-free dinners" value="${esc(f.q || "")}" enterkeyhint="search">
        <button id="ask" class="btn-primary hidden">✨ Ask</button></form>
      ${people.length ? `<div>
        <span class="label">Who's ${home ? "using it" : "eating"}?</span>
        <div id="who" class="flex flex-wrap gap-2">${people.map((p) =>
          `<button class="pick ${f.who.includes(p.id) ? "on" : ""}" data-id="${p.id}">${esc(p.name)}${p.is_guest ? " (guest)" : ""}</button>`).join("")}</div>
      </div>` : `<p class="text-sm text-slate-400">Add the family on the <a class="underline" href="#/family">Family</a> page to see who can eat what.</p>`}
      <details id="more" class="group" ${innerWidth >= 768 || filtered(f) ? "open" : ""}>
      <summary class="cursor-pointer text-sm font-semibold text-slate-300">Filters${filtered(f) ? " (on)" : ""}</summary>
      <div class="mt-3 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <label><span class="label">Show</span><select id="ok" class="input">
          <option value="">Everything</option>
          <option value="1" ${f.ok === "1" ? "selected" : ""}>Only OK for all of them</option>
          <option value="unsure" ${f.ok === "unsure" ? "selected" : ""}>OK or not sure</option></select></label>
        ${home ? `<label><span class="label">Kind</span><select id="course" class="input"><option value="">Any</option>${HOME_CATS.map((c) =>
          `<option ${f.course === c ? "selected" : ""} value="${c}">${cap(c)}</option>`).join("")}</select></label>`
        : `<label><span class="label">Ready in</span><select id="max_min" class="input"><option value="">Any time</option>${[15, 30, 45, 60, 90].map((m) =>
          `<option value="${m}" ${f.max_min == m ? "selected" : ""}>${m} min or less</option>`).join("")}</select></label>
        <label><span class="label">Diet</span><select id="diet" class="input"><option value="">Any</option>${(state.info?.diets || []).map((d) =>
          `<option value="${d.key}" ${f.diet === d.key ? "selected" : ""}>${esc(d.label)}</option>`).join("")}</select></label>
        <label><span class="label">Heat</span><select id="heat" class="input"><option value="">Any</option>${[0, 1, 2, 3, 4].map((h) =>
          `<option value="${h}" ${f.heat === String(h) ? "selected" : ""}>${h ? "Up to " + "🌶️".repeat(h) : "No heat"}</option>`).join("")}</select></label>
        <label><span class="label">How hard</span><select id="level" class="input"><option value="">Any</option>${Object.entries(LEVELS).map(([k, l]) =>
          `<option value="${k}" ${f.level === k ? "selected" : ""}>${l}</option>`).join("")}</select></label>`}
      </div>
      <div id="facets" class="mt-3 grid gap-3 sm:grid-cols-3"></div>
      </details>
    </div>
    <div id="results"></div>`;

  let timer;
  const load = async () => {
    const q = isQuestion(f.q) ? "" : f.q;
    const data = await get("/api/recipes" + qs({ area, who: f.who.join(",") || "0", q, ok: f.ok, max_min: f.max_min,
      diet: f.diet, heat: f.heat, level: f.level, course: f.course, cuisine: f.cuisine, protein: f.protein }));
    if (!home) renderFacets(data.facets, f, load);
    renderCards(data.recipes, area, f);
  };
  // Three words or more is a question: answered by Ask (or Enter), not word search.
  const ask = () => askRecipes($("#results"), area, f.q.trim(), f.who, state.info?.diets, () => { f.q = ""; $("#q").value = ""; $("#ask").classList.add("hidden"); load(); });
  $("#q").oninput = (e) => {
    f.q = e.target.value;
    $("#ask").classList.toggle("hidden", !isQuestion(f.q));
    clearTimeout(timer);
    if (!isQuestion(f.q)) timer = setTimeout(load, 250);
  };
  $("#qf").onsubmit = (e) => {
    e.preventDefault();
    if (isQuestion(f.q)) ask();
    else load();
  };
  $("#ask").classList.toggle("hidden", !isQuestion(f.q));
  $$("#who .pick").forEach((b) => (b.onclick = () => {
    const id = Number(b.dataset.id);
    f.who = f.who.includes(id) ? f.who.filter((x) => x !== id) : [...f.who, id];
    b.classList.toggle("on");
    load();
  }));
  for (const key of ["ok", "course", "max_min", "diet", "heat", "level"]) {
    const el = $("#" + key);
    if (el) el.onchange = () => { f[key] = el.value; load(); };
  }
  await load();
  get(`/api/collections?area=${area}&date=${localDate()}`).then((d) => {
    const box = $("#seasons", view);
    if (box) box.innerHTML = d.seasons.map((s) => `<a href="#/season/${s.key}" class="pick">${esc(s.icon)} ${esc(s.name)} (${s.count})</a>`).join("");
  }).catch(() => {});
}

function renderFacets(facets, f, load) {
  const box = $("#facets");
  if (!box) return;
  box.innerHTML = ["course", "cuisine", "protein"].map((k) => {
    const opts = Object.keys(facets[k] || {}).sort();
    if (!opts.length && !f[k]) return "";
    return `<label><span class="label">${cap(k)}</span><select data-facet="${k}" class="input"><option value="">Any</option>${opts.map((o) =>
      `<option value="${esc(o)}" ${f[k] === o ? "selected" : ""}>${esc(cap(o))} (${facets[k][o]})</option>`).join("")}</select></label>`;
  }).join("");
  $$("[data-facet]", box).forEach((s) => (s.onchange = () => { f[s.dataset.facet] = s.value; load(); }));
}

function renderCards(list, area, f) {
  const box = $("#results");
  if (!list.length) {
    box.innerHTML = `<div class="card text-center text-slate-400">${f.q || filtered(f) ? "No recipes match these filters." :
      `No ${area === "home" ? "Home & Care" : ""} recipes yet. Add one from a link, a photo of a card, or by typing it in.`}</div>`;
    return;
  }
  box.innerHTML = `<p class="mb-2 text-sm text-slate-400">${list.length} recipe${list.length === 1 ? "" : "s"}</p>${cardGrid(list, area)}`;
}

// cardGrid is the recipe cards (for the Library, collections and seasons).
// remove, when given, adds a ✕ to each card with data-remove="<id>".
export function cardGrid(list, area, remove = false) {
  return `<div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">${list.map((r) => `<div class="relative min-w-0">
      <a href="#/recipe/${r.id}" class="card block min-w-0 p-0 hover:ring-emerald-700">
        ${r.photo ? `<img src="/api/photos/${esc(r.photo)}?w=480" alt="" loading="lazy" class="h-40 w-full rounded-t-xl object-cover">`
          : `<div class="flex h-24 items-center justify-center rounded-t-xl bg-slate-800 text-4xl">${area === "home" ? "🧴" : "🍲"}</div>`}
        <div class="space-y-2 p-3">
          <h2 class="break-words font-semibold leading-snug">${esc(r.title)}</h2>
          <div class="flex flex-wrap items-center gap-1 text-xs text-slate-400">
            ${r.total_min ? `<span class="chip-info">⏱ ${fmtMin(r.total_min)}</span>` : ""}
            ${area === "kitchen" ? peppers(r.heat) + levelChip(r.level) : ""}
            ${r.course ? `<span class="chip-cat">${esc(cap(r.course))}</span>` : ""}
            ${r.needs_review ? `<span class="chip-unsure" title="The AI wasn't sure about some lines">? Check lines</span>` : ""}
            ${stars(r.rating)}
          </div>
          <div class="flex flex-wrap gap-1">${verdictChips(r.verdicts)}</div>
        </div>
      </a>${remove ? `<button data-remove="${r.id}" class="absolute right-2 top-2 rounded-full bg-slate-900/90 px-3 py-1 text-sm ring-1 ring-slate-700" aria-label="Take off this collection">✕</button>` : ""}</div>`).join("")}</div>`;
}

// filtered: any filter beyond search and who's eating is on.
function filtered(f) {
  return Boolean(f.ok || f.max_min || f.diet || f.heat || f.level || f.course || f.cuisine || f.protein);
}

// isQuestion: three words or more is a question for Ask, not a word search.
function isQuestion(q) {
  return (q || "").trim().split(/\s+/).length >= 3;
}
