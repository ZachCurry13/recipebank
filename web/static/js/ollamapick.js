// The Ollama card's two pickers, one for text and one for photos: models
// already on that Ollama first, then ones to download (labelled for its
// graphics card), then any other model by name.
import { $, esc } from "./ui.js";

// How each model fits the graphics card (see internal/ollama/gpu.go).
export const FIT = {
  best: ["⭐ Best for your graphics card", "text-emerald-300"],
  best_powerful: ["⭐ Best and most powerful for your graphics card", "text-emerald-300"],
  powerful: ["💪 Most powerful that fits", "text-indigo-300"],
  fits: ["✓ Fits", "text-slate-300"],
  too_big: ["⚠️ Too big (slow)", "text-amber-300"],
  cpu_ok: ["✓ OK without a graphics card", "text-slate-300"],
  cpu_slow: ["⚠️ Slow without a graphics card", "text-amber-300"],
};
const RANK = { best_powerful: 0, best: 1, cpu_ok: 2, powerful: 3, fits: 4, "": 5, cpu_slow: 6, too_big: 7 };

const KINDS = {
  text: ["📝 For text", "Pasted recipes, organizing what a card says, swaps and ideas.", "Use for text"],
  photos: ["📷 For photos", "Recipe cards, receipts, the fridge. On the main AI's own Ollama it reads the main AI's photos; on another computer it becomes the AI for photos.", "Use for photos"],
};

export const pickRows = () => Object.entries(KINDS).map(([kind, [title, about]]) => `<div class="space-y-2" data-kind="${kind}">
    <p class="label mb-0">${title}</p><p class="text-xs text-slate-500">${about}</p>
    <div class="flex flex-wrap gap-2"><select data-pick class="input min-w-0 flex-1 basis-56" aria-label="${title}"></select>
      <button type="button" data-ol="go" class="btn-secondary"></button></div>
    <input data-other-name class="input hidden" placeholder="Model name from ollama.com/library, e.g. minicpm-v" autocapitalize="off" spellcheck="false">
    <p data-pick-note class="text-xs text-slate-400"></p></div>`).join("");

// fillPicks fills both pickers from what's on the server (srv) and the
// catalog (recs: labelled once the graphics card is known).
export function fillPicks(card, srv, recs) {
  const have = (name) => srv.models.includes(name) || srv.models.includes(`${name}:latest`);
  const sees = new Set(srv.photo_models || []);
  const sorted = (ms) => [...ms].sort((a, b) => RANK[a.fit] - RANK[b.fit] || a.size_gb - b.size_gb);
  for (const kind of Object.keys(KINDS)) {
    const row = $(`[data-kind="${kind}"]`, card);
    const sel = $("[data-pick]", row);
    const keep = sel.value;
    const mine = kind === "photos" ? srv.models.filter((m) => sees.has(m))
      : [...srv.models.filter((m) => !sees.has(m)), ...srv.models.filter((m) => sees.has(m))];
    const own = mine.map((m) => `<option value="${esc(m)}" data-have="1" data-note="${kind === "text" && sees.has(m)
      ? "Downloaded. A photo model: it works for text too, but a text model is lighter." : "Downloaded."}">${esc(m)}${sees.has(m) ? " · 📷" : ""}</option>`).join("");
    const get = sorted(kind === "photos" ? recs.photos : recs.text).filter((m) => !have(m.name)).map((m) => `<option value="${esc(m.name)}"
      data-fit="${m.fit}" data-note="${esc(m.note)}">${esc(m.label)} · ${m.size_gb} GB${m.fit ? ` · ${FIT[m.fit][0]}` : ""}</option>`).join("");
    sel.innerHTML = `${own ? `<optgroup label="On this Ollama">${own}</optgroup>` : ""}${get ? `<optgroup label="Download">${get}</optgroup>` : ""}
      <option value="" data-other="1">Other model… (type its name)</option>`;
    if (keep && [...sel.options].some((o) => o.value === keep)) sel.value = keep;
    sel.onchange = () => describe(row, kind);
    describe(row, kind);
  }
}

// describe updates a picker's button and note for the chosen model.
function describe(row, kind) {
  const o = $("[data-pick]", row).selectedOptions[0];
  const other = !!o?.dataset.other;
  $("[data-other-name]", row).classList.toggle("hidden", !other);
  $("[data-ol='go']", row).textContent = o?.dataset.have ? KINDS[kind][2] : "Download and use";
  const [label, cls] = FIT[o?.dataset.fit] || ["", "text-slate-400"];
  const note = $("[data-pick-note]", row);
  note.className = `text-xs ${cls}`;
  note.textContent = other ? (kind === "photos"
    ? "Any model from ollama.com/library that reads photos (it says “vision” there), e.g. minicpm-v or llava."
    : "Any model from ollama.com/library, e.g. mistral or phi4-mini.")
    : o ? `${o.dataset.note}${label ? ` · ${label}` : ""}` : "";
}

// picked is the chosen model: [name, already downloaded].
export function picked(card, kind) {
  const row = $(`[data-kind="${kind}"]`, card);
  const o = $("[data-pick]", row).selectedOptions[0];
  if (o?.dataset.other) return [$("[data-other-name]", row).value.trim(), false];
  return [o?.value || "", !!o?.dataset.have];
}
