// How each person's verdict looks: compact chips and the full reasons.
import { esc } from "./ui.js";

const ICON = { ok: "✓", no: "✕", unsure: "⚠" };
const CHIP = { ok: "chip-ok", no: "chip-no", unsure: "chip-unsure" };
const WORD = { ok: "OK for", no: "Not for", unsure: "Not sure for" };

// verdictChips groups people by status: "✓ Anna, Ben" "✕ Cara".
export function verdictChips(verdicts) {
  const out = [];
  for (const s of ["no", "unsure", "ok"]) {
    const names = verdicts.filter((v) => v.status === s).map((v) => v.name);
    if (names.length) out.push(`<span class="${CHIP[s]}" title="${WORD[s]} ${esc(names.join(", "))}">${ICON[s]} ${esc(names.join(", "))}</span>`);
  }
  return out.join(" ");
}

// verdictList is the recipe page's "who can eat it" box.
export function verdictList(verdicts, ingredients) {
  if (!verdicts.length) return `<p class="text-sm text-slate-400">Nobody picked. Add people on the <a class="underline" href="#/family">Family</a> page.</p>`;
  return `<ul class="space-y-2">${verdicts.map((v) => `
    <li class="text-sm">
      <span class="${CHIP[v.status]} text-sm">${ICON[v.status]} ${WORD[v.status]} ${esc(v.name)}</span>
      ${v.reasons.length ? `<ul class="mt-1 space-y-0.5 pl-4 text-slate-300">${[...v.reasons].sort((a, b) => (a.status === "no" ? 0 : 1) - (b.status === "no" ? 0 : 1)).map((r) => `
        <li>${r.status === "no" ? "✕" : "⚠"} <b>${esc(r.rule)}</b>: ${esc(r.text)}${lineNames(r.ingredients, ingredients)}</li>`).join("")}</ul>` : ""}
    </li>`).join("")}</ul>`;
}

function lineNames(idx, ingredients) {
  const names = (idx || []).map((i) => ingredients[i]?.food || ingredients[i]?.line).filter(Boolean);
  return names.length ? ` <span class="text-slate-400">(${esc(names.join(", "))})</span>` : "";
}

// flagsFor lists, for one ingredient line, who has a problem with it.
export function flagsFor(i, verdicts) {
  const out = [];
  for (const v of verdicts) {
    for (const r of v.reasons) {
      if (r.ingredients.includes(i)) out.push({ person: v.name, personId: v.person_id, ...r });
    }
  }
  return out;
}
