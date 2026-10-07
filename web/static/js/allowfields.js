// Family → allergies → "Can have anyway": foods made from the allergen that
// this person's doctor says are fine (highly refined soybean oil for a soy
// allergy). Only that exact food is let through; everything else made from
// the allergen still counts (see internal/safety/exceptions.go).
import { $, $$, esc } from "./ui.js";

const offeredKeys = (a) => (a.exceptions || []).map((x) => x.key);

// allowFields is the box under one allergy, shown while the allergy is set.
export function allowFields(a, rule) {
  const has = rule?.allow || [];
  const typed = has.filter((k) => !offeredKeys(a).includes(k)).join(", ");
  return `<div data-allow-for="${a.key}" class="space-y-2 rounded-lg border border-slate-800 p-3 ${rule ? "" : "hidden"}">
    <p class="text-xs text-slate-400">Can have anyway, only if their doctor says so. Everything else made from ${esc(a.label.toLowerCase())} still counts.</p>
    ${(a.exceptions || []).map((x) => `<label class="flex items-start gap-2 text-sm">
      <input type="checkbox" data-allow="${a.key}" value="${esc(x.key)}" class="mt-1" ${has.includes(x.key) ? "checked" : ""}>
      <span class="min-w-0"><span class="block">${esc(x.label)}</span><span class="block text-xs text-slate-500">${esc(x.note)}</span></span></label>`).join("")}
    <input data-allow-typed="${a.key}" class="input" maxlength="300" value="${esc(typed)}"
      placeholder="${a.exceptions?.length ? "Other foods" : "Foods they can have"}, comma between${a.key === "treenut" ? " (e.g. almond)" : ""}" aria-label="Other foods they can have">
  </div>`;
}

// wireAllowFields shows each box while its allergy is chosen.
export function wireAllowFields(form) {
  $$("select[name^='allergy_']", form).forEach((sel) => {
    const box = $(`[data-allow-for="${sel.name.slice(8)}"]`, form);
    if (box) sel.addEventListener("change", () => box.classList.toggle("hidden", !sel.value));
  });
}

// allowFor is what's ticked and typed for one allergy.
export function allowFor(form, key) {
  const ticked = $$(`[data-allow="${key}"]`, form).filter((c) => c.checked).map((c) => c.value);
  const typed = ($(`[data-allow-typed="${key}"]`, form)?.value || "").split(",").map((w) => w.trim().toLowerCase()).filter(Boolean);
  return [...new Set([...ticked, ...typed])];
}
