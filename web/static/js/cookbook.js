// The family cookbook: every Kitchen recipe (or one collection) laid out to
// print, or to save as a PDF from the print menu. Each recipe gets its own page.
import { get, qs } from "./api.js";
import { $, esc, fmtMin, cap } from "./ui.js";
import { parts, stepText } from "./ingredients.js";

export async function renderCookbook(view, params, state) {
  const book = await get("/api/cookbook" + qs(params.collection ? { collection: params.collection } : {}));
  const system = state.user.units || state.info?.default_units || "us";
  let title = book.title;
  let photos = true;
  const month = new Date().toLocaleDateString(undefined, { month: "long", year: "numeric" });

  const ingredients = (r) => {
    let section = "";
    return r.ingredients.map((ing) => {
      const head = (ing.section || "") !== section ? `<li class="mt-2 font-semibold">${esc((section = ing.section || ""))}</li>` : "";
      const p = parts(ing, 1, system);
      return `${head}<li><b>${esc(p.amount)}</b> ${esc(p.name)}${p.note ? ` <span class="text-slate-400">${esc(p.note)}</span>` : ""}</li>`;
    }).join("");
  };
  const page = (r) => `<section class="print-page card min-w-0 space-y-3">
    <h2 class="break-words text-2xl font-bold">${esc(r.title)}</h2>
    ${r.summary ? `<p class="text-slate-300">${esc(r.summary)}</p>` : ""}
    <p class="text-sm text-slate-400">${[r.servings ? `Serves ${+r.servings}` : "", r.prep_min ? `Prep ${fmtMin(r.prep_min)}` : "",
      r.total_min ? `Total ${fmtMin(r.total_min)}` : "", r.course ? cap(r.course) : ""].filter(Boolean).join(" · ")}</p>
    ${photos && r.photo ? `<img src="/api/photos/${esc(r.photo)}?w=1200" alt="" class="max-h-72 w-full rounded-lg object-cover">` : ""}
    <div class="grid gap-4 sm:grid-cols-[minmax(0,1fr)_minmax(0,1.4fr)]">
      ${r.ingredients.length ? `<div><h3 class="mb-1 font-semibold">Ingredients</h3><ul class="space-y-1 text-sm">${ingredients(r)}</ul></div>` : ""}
      ${r.steps.length ? `<div><h3 class="mb-1 font-semibold">Steps</h3><ol class="list-decimal space-y-2 pl-5 text-sm">${r.steps.map((s) =>
        `<li class="break-words">${esc(stepText(s.text, system))}</li>`).join("")}</ol></div>` : ""}
    </div>
    ${r.notes ? `<p class="whitespace-pre-line text-sm"><b>Our notes:</b> ${esc(r.notes)}</p>` : ""}
    ${r.source_note ? `<p class="text-xs text-slate-400">From: ${esc(r.source_note)}</p>` : ""}
  </section>`;

  const draw = () => {
    view.innerHTML = `
      <div class="no-print card mb-6 space-y-3">
        <h1 class="text-2xl font-bold">📖 ${esc(book.title)}</h1>
        <p class="text-sm text-slate-400">${book.recipes.length} recipe${book.recipes.length === 1 ? "" : "s"}, each on its own page.
          Print it, or choose <b>Save as PDF</b> in the print menu.</p>
        <label class="block"><span class="label">Title on the cover</span>
          <input id="cb-title" class="input" maxlength="80" value="${esc(title)}"></label>
        <label class="toggle"><input type="checkbox" id="cb-photos" ${photos ? "checked" : ""}> Include photos</label>
        <button type="button" id="cb-print" class="btn-primary" ${book.recipes.length ? "" : "disabled"}>🖨 Print or save as PDF</button>
      </div>
      <article class="space-y-8">
        <section class="card py-16 text-center"><p class="text-5xl">📖</p>
          <h2 class="mt-4 break-words text-4xl font-bold">${esc(title)}</h2>
          <p class="mt-2 text-slate-400">${book.recipes.length} recipe${book.recipes.length === 1 ? "" : "s"} · ${esc(month)}</p></section>
        ${book.recipes.length ? `<section class="print-page card"><h2 class="mb-3 text-2xl font-bold">Contents</h2>
          <ol class="list-decimal space-y-1 pl-6">${book.recipes.map((r) => `<li class="break-words">${esc(r.title)}${r.course ? ` <span class="text-slate-400">· ${esc(cap(r.course))}</span>` : ""}</li>`).join("")}</ol></section>` : ""}
        ${book.recipes.map(page).join("")}
      </article>`;
    $("#cb-title", view).onchange = (e) => { title = e.target.value.trim() || book.title; draw(); };
    $("#cb-photos", view).onchange = (e) => { photos = e.target.checked; draw(); };
    $("#cb-print", view).onclick = () => window.print();
  };
  draw();
}
