// Picking what to plan for a meal: a recipe (checked for who's eating) or a note.
import { get, post, put, qs } from "./api.js";
import { $, $$, esc, attempt, sheet, fmtMin } from "./ui.js";
import { verdictChips } from "./verdicts.js";

export const MEAL_NAMES = { breakfast: "Breakfast", lunch: "Lunch", dinner: "Dinner", snack: "Snack" };
const ORDER = { breakfast: 0, lunch: 1, dinner: 2, snack: 3 };

// mealName: "Mon, Oct 5 dinner".
export const mealName = (ref) => `${dayLabel(ref.date)} ${MEAL_NAMES[ref.meal].toLowerCase()}`;

// localDate is a date as YYYY-MM-DD in the phone's own time zone.
export function localDate(d = new Date()) {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}

export function dayLabel(date, long = false) {
  const d = new Date(date + "T00:00:00");
  return d.toLocaleDateString(undefined, long ? { weekday: "long", month: "long", day: "numeric" } : { weekday: "short", month: "short", day: "numeric" });
}

// pickMeal opens the picker for one meal; who are the people eating.
export function pickMeal(date, meal, who, done) {
  const d = sheet(`<div class="space-y-3">
    <div class="flex items-center"><h2 class="text-lg font-semibold">${MEAL_NAMES[meal]}, ${esc(dayLabel(date))}</h2>
      <button type="button" data-close class="btn-ghost ml-auto">✕</button></div>
    <div class="grid grid-cols-2 gap-2">
      <input id="pq" type="search" class="input col-span-2" placeholder="Search recipes">
      <label class="col-span-2 flex items-center gap-2 text-sm text-slate-300">For how many?
        <input id="serv" type="number" min="0" step="any" inputmode="decimal" class="input w-24" placeholder="as written"></label>
    </div>
    <div id="lo"></div>
    <ul id="picks" class="max-h-[45dvh] space-y-1 overflow-y-auto"></ul>
    <form id="note" class="flex flex-wrap gap-2 border-t border-slate-800 pt-3">
      <input name="title" class="input min-w-0 flex-1 basis-40" placeholder="Or just write it: Leftovers, Pizza night out">
      <button class="btn-secondary">Add</button></form></div>`);
  const add = (body) => attempt(async () => {
    await post("/api/plan", { date, meal, servings: Number($("#serv", d).value) || 0, ...body });
    d.close();
    done();
  });
  let timer;
  const search = async () => {
    const data = await get("/api/recipes" + qs({ area: "kitchen", who: who.join(",") || "0", q: $("#pq", d).value }));
    $("#picks", d).innerHTML = data.recipes.length ? data.recipes.map((r) => `<li><button data-pick="${r.id}"
      class="flex w-full min-w-0 flex-wrap items-center gap-2 rounded-lg p-2 text-left hover:bg-slate-800">
      <span class="min-w-0 flex-1 basis-40 break-words font-medium">${esc(r.title)}</span>
      ${r.total_min ? `<span class="text-xs text-slate-400">${fmtMin(r.total_min)}</span>` : ""}
      <span class="flex basis-full flex-wrap gap-1">${verdictChips(r.verdicts)}</span></button></li>`).join("")
      : `<li class="p-2 text-sm text-slate-400">No recipes match.</li>`;
    $$("[data-pick]", d).forEach((b) => (b.onclick = () => add({ recipe_id: Number(b.dataset.pick) })));
  };
  $("#pq", d).oninput = () => { clearTimeout(timer); timer = setTimeout(() => attempt(search), 250); };
  // Leftovers of a recipe planned in the last few days (not leftovers themselves).
  const from = new Date(date + "T00:00:00");
  from.setDate(from.getDate() - 3);
  get("/api/plan" + qs({ from: localDate(from), days: 4 })).then((plan) => {
    const earlier = plan.days.flatMap((day) => day.meals).filter((m) => m.recipe && !m.leftovers_of &&
      (m.date < date || (m.date === date && ORDER[m.meal] < ORDER[meal])));
    if (!earlier.length) return;
    $("#lo", d).innerHTML = `<p class="mb-1 text-sm font-semibold">🍱 Leftovers from…</p><div class="flex flex-wrap gap-2">${earlier.map((m) =>
      `<button type="button" data-lo="${m.id}" class="btn-secondary min-h-0 max-w-full break-words py-1 text-left text-sm">${esc(mealName(m))}: ${esc(m.recipe.title)}</button>`).join("")}</div>`;
    $$("[data-lo]", d).forEach((b) => (b.onclick = () => add({ leftovers_of: Number(b.dataset.lo) })));
  }).catch(() => {});
  $("#note", d).onsubmit = (e) => {
    e.preventDefault();
    const title = e.target.title.value.trim();
    if (title) add({ title });
  };
  attempt(search);
}

// pickWho opens the "who's eating" toggles for a day.
export function pickWho(date, people, who, done) {
  const on = new Set(who);
  const d = sheet(`<div class="space-y-3">
    <div class="flex items-center"><h2 class="text-lg font-semibold">Who's eating, ${esc(dayLabel(date))}?</h2>
      <button type="button" data-close class="btn-ghost ml-auto">✕</button></div>
    <div class="flex flex-wrap gap-2">${people.map((p) => `<button data-p="${p.id}" class="pick ${on.has(p.id) ? "on" : ""}">${esc(p.name)}${p.is_guest ? " (guest)" : ""}</button>`).join("")}</div>
    <p class="text-xs text-slate-500">Add guests on the Family page first.</p>
    <div class="flex flex-wrap gap-2"><button id="save" class="btn-primary">Save</button>
      <button id="reset" class="btn-ghost">Everyone but guests</button></div></div>`);
  $$("[data-p]", d).forEach((b) => (b.onclick = () => {
    const id = Number(b.dataset.p);
    on.has(id) ? on.delete(id) : on.add(id);
    b.classList.toggle("on");
  }));
  const save = (body) => attempt(async () => {
    await put(`/api/plan/day/${date}`, body);
    d.close();
    done();
  });
  $("#save", d).onclick = () => save({ who: [...on] });
  $("#reset", d).onclick = () => save({ who: null });
}
