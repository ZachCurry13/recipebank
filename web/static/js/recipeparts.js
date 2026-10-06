// The recipe page's ingredient list (amounts in a bold column, the item, a
// quiet note, short tags for problems and swaps) and steps.
import { esc } from "./ui.js";
import { flagsFor } from "./verdicts.js";
import { parts, stepText } from "./ingredients.js";
import { temps, timersIn } from "./units.js";

// shortWhy is a problem in a word or two: "milk", "vegetarian", "dislikes olives".
function shortWhy(f, nm) {
  if (f.allergen) return nm(f.allergen);
  if (f.rule === "Dislikes") return `dislikes ${f.text}`;
  if (f.rule === "Sensitive to") return `sensitive: ${f.text}`;
  if (f.rule === "Heat") return "too spicy";
  return f.rule.toLowerCase();
}

// tags groups everyone's problems with one line: "✕ Kid, Sam: milk".
function tags(flags, nm) {
  const groups = new Map();
  for (const f of flags) {
    const key = `${f.status}|${shortWhy(f, nm)}`;
    if (!groups.has(key)) groups.set(key, { status: f.status, why: shortWhy(f, nm), people: [] });
    const g = groups.get(key);
    if (!g.people.includes(f.person)) g.people.push(f.person);
  }
  return [...groups.values()].map((g) => g.status === "no"
    ? `<span class="chip-no">✕ ${esc(g.people.join(", "))}: ${esc(g.why)}</span>`
    : `<span class="chip-unsure">⚠ ${esc(g.people.join(", "))}: check ${esc(g.why)}</span>`).join("");
}

// pantryLabel: a labelled product in the pantry that settles a "not sure" line.
function pantryLabel(i, allergen, hints, manage, nm) {
  const h = (hints || []).find((x) => x.ingredient === i);
  if (!h) return "";
  const name = esc([h.brand, h.name].filter(Boolean).join(" "));
  if (h.allergens.includes(allergen)) return `<div class="text-rose-300">🥫 Your pantry's ${name}: the label says it contains ${esc(nm(allergen))}.</div>`;
  if (h.traces.includes(allergen)) return `<div class="text-amber-300">🥫 Your pantry's ${name}: the label says it may contain ${esc(nm(allergen))}.</div>`;
  return `<div class="text-emerald-300">🥫 Your pantry's ${name}: its label lists no ${esc(nm(allergen))}.
    ${manage ? `<button data-label="${i}:${allergen}:1" class="ml-1 underline">Use this label</button>` : ""}</div>`;
}

export function ingredientList(r, data, manage, v, nm) {
  let section = "";
  return r.ingredients.map((ing, i) => {
    const head = ing.section && ing.section !== section
      ? `<li class="pt-3 text-xs font-semibold uppercase tracking-wide text-slate-400">${esc((section = ing.section))}</li>` : "";
    const p = parts(ing, v.factor, v.system);
    const flags = flagsFor(i, data.verdicts);
    const unsure = flags.filter((f) => f.status === "unsure" && f.allergen);
    const swaps = data.swaps.filter((s) => s.ingredient === i);
    const tone = flags.some((f) => f.status === "no") ? "text-rose-200" : flags.length ? "text-amber-200" : "";
    const allergens = [...new Set(unsure.map((f) => f.allergen))];
    const extras = [
      tags(flags, nm),
      swaps.map((s) => `<span class="chip-ok" title="${esc(s.note || "OK for everyone picked")}">↔ ${esc(s.to)}</span>`).join(""),
      (ing.checked || []).map((a) => `<span class="chip-ok">✓ label: no ${esc(nm(a))}${manage ? ` <button data-label="${i}:${a}:0" class="ml-1 underline">undo</button>` : ""}</span>`).join(""),
    ].join("");
    const help = allergens.map((a) => `${manage ? `<button data-label="${i}:${a}:1" class="underline">I checked the label: no ${esc(nm(a))}</button>` : ""}
      ${pantryLabel(i, a, data.pantry, manage, nm)}`).join("");
    return `${head}<li data-ing="${i}" class="grid min-w-0 cursor-pointer grid-cols-[4.75rem_minmax(0,1fr)] gap-x-3 rounded-lg px-1 py-1.5 hover:bg-slate-800/40">
      <span class="break-words text-right font-semibold tabular-nums ${tone}">${esc(p.amount)}</span>
      <div class="min-w-0">
        <span class="break-words ${tone}">${ing.unsure ? `<b class="text-amber-300" title="The AI wasn't sure about this line">?</b> ` : ""}${esc(p.name)}</span>
        ${p.note ? `<span class="block text-xs text-slate-500">${esc(p.note)}</span>` : ""}
        ${extras ? `<div class="mt-1 flex flex-wrap gap-1">${extras}</div>` : ""}
        ${help.trim() ? `<div class="mt-1 space-y-0.5 text-xs text-slate-300">${help}</div>` : ""}
      </div></li>`;
  }).join("");
}

export function stepList(r, v) {
  let section = "";
  return r.steps.map((s) => {
    const head = s.section && s.section !== section
      ? `<li class="list-none pt-2 text-xs font-semibold uppercase tracking-wide text-slate-400">${esc((section = s.section))}</li>` : "";
    const t = timersIn(s.text).map((x) => `<span class="chip-info">⏱ ${esc(x.label)}</span>`).join(" ");
    return `${head}<li class="ml-5 list-decimal break-words pl-1 text-slate-200">${s.unsure ? `<b class="text-amber-300">?</b> ` : ""}${esc(temps(stepText(s.text, v.system), v.system))} ${t}</li>`;
  }).join("");
}
