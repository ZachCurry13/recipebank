// The recipe editor: for new drafts (from a link, photos or text) and edits.
// Ingredients and steps are plain text, one per line; "Filling:" starts a
// part; "(?) " marks a line the AI wasn't sure about.
import { get, post, put } from "./api.js";
import { $, $$, esc, attempt, busy, COURSES, HOME_CATS } from "./ui.js";
import { verdictList } from "./verdicts.js";
import { shrinkPhoto } from "./photo.js";
import { cardBox } from "./cardcheck.js";
import { go } from "./app.js";

export async function renderEdit(view, params) {
  const data = await get(`/api/recipes/${params.id}`);
  editRecipe(view, data.recipe, null, data.card_check);
}

function toText(list, key) {
  let section = "";
  const out = [];
  for (const it of list) {
    if ((it.section || "") !== section) {
      section = it.section || "";
      if (section) out.push(`${section}:`);
    }
    out.push((it.unsure ? "(?) " : "") + it[key]);
  }
  return out.join("\n");
}

function fromText(text, key) {
  let section = "";
  const out = [];
  for (let line of text.split("\n")) {
    line = line.trim();
    if (!line) continue;
    if (line.endsWith(":") && line.length <= 40 && !/\d/.test(line)) {
      section = line.slice(0, -1);
      continue;
    }
    const unsure = line.startsWith("(?)");
    out.push({ [key]: unsure ? line.slice(3).trim() : line, section, unsure });
  }
  return out;
}

// editRecipe shows the form for r; preview is the import's check (or null).
export function editRecipe(view, r, preview, cardCheck) {
  const isNew = !r.id;
  let photo = r.photo || "";
  const field = (name, label, value, attrs = "") =>
    `<label class="block"><span class="label">${label}</span><input name="${name}" value="${esc(value ?? "")}" class="input" ${attrs}></label>`;
  view.innerHTML = `
    <a href="${isNew ? "#/add" : `#/recipe/${r.id}`}" class="text-sm text-slate-400 hover:text-slate-200">‹ ${isNew ? "Add a recipe" : "Back to the recipe"}</a>
    <h1 class="mb-4 mt-2 text-2xl font-bold">${isNew ? "Check it, then save" : "Edit recipe"}</h1>
    ${r.needs_review ? `<div class="mb-4">${cardBox(r, preview?.card_check || cardCheck, "edit")}</div>` : ""}
    ${preview ? `<div class="card mb-4 space-y-3"><h2 class="font-semibold">Who can ${r.area === "home" ? "use" : "eat"} it</h2>
      ${verdictList(preview.verdicts, r.ingredients)}
      ${(preview.hazards || []).filter((h) => h.level !== "info").map((h) => `<div class="${h.level === "danger" ? "box-danger" : "box-caution"}">${esc(h.text)}</div>`).join("")}</div>` : ""}
    <form id="edit" class="grid gap-4 lg:grid-cols-2">
      <div class="space-y-4">
        <div class="flex flex-wrap gap-2" role="radiogroup" aria-label="Where it goes">
          ${[["kitchen", "🍲 Kitchen"], ["home", "🧴 Home & Care"]].map(([a, l]) =>
            `<label class="pick ${r.area === a ? "on" : ""}"><input type="radio" name="area" value="${a}" class="sr-only" ${r.area === a ? "checked" : ""}>${l}</label>`).join("")}
        </div>
        ${field("title", "Title", r.title, "required maxlength=200")}
        ${field("summary", "Short description", r.summary)}
        <div><span class="label">Photo of the finished ${r.area === "home" ? "product" : "dish"}</span>
          <div id="photo-box" class="flex flex-wrap items-center gap-3"></div></div>
        <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
          ${field("servings", "Servings", r.servings || "", 'type="number" min="0" step="any" inputmode="decimal"')}
          ${field("prep_min", "Prep (min)", r.prep_min || "", 'type="number" min="0" inputmode="numeric"')}
          ${field("cook_min", "Cook (min)", r.cook_min || "", 'type="number" min="0" inputmode="numeric"')}
          ${field("total_min", "Total (min)", r.total_min || "", 'type="number" min="0" inputmode="numeric"')}
        </div>
        ${field("yield_text", "Makes (optional)", r.yield_text, 'placeholder="2 dozen cookies, one 16 oz bottle"')}
        <div class="grid grid-cols-2 gap-3">
          ${field("course", "Course or kind", r.course, 'list="courses"')}
          <datalist id="courses">${[...COURSES, ...HOME_CATS].map((c) => `<option value="${c}">`).join("")}</datalist>
          ${field("cuisine", "Cuisine", r.cuisine)}
          ${field("protein", "Main protein", r.protein)}
          <label class="block"><span class="label">Heat</span><select name="heat" class="input">
            <option value="-1">Work it out</option>${[0, 1, 2, 3, 4, 5].map((h) =>
              `<option value="${h}" ${r.heat === h ? "selected" : ""}>${h ? "🌶️".repeat(h) : "No heat"}</option>`).join("")}</select></label>
        </div>
      </div>
      <div class="space-y-4">
        <label class="block"><span class="label">Ingredients (one per line)</span>
          <textarea name="ingredients" rows="10" class="input font-mono text-sm">${esc(toText(r.ingredients, "line"))}</textarea></label>
        <label class="block"><span class="label">Steps (one per line)</span>
          <textarea name="steps" rows="10" class="input text-sm">${esc(toText(r.steps, "text"))}</textarea></label>
        <label class="block"><span class="label">Our notes</span><textarea name="notes" rows="3" class="input">${esc(r.notes)}</textarea></label>
        <label class="block"><span class="label">Storage (how to keep it, how long)</span><textarea name="storage" rows="2" class="input">${esc(r.storage)}</textarea></label>
        ${field("source_url", "Web address it came from", r.source_url, 'type="url"')}
        ${field("source_note", "Where it's from", r.source_note, 'placeholder="Grandma\'s card, 1962 · cookbook, page 42"')}
        ${(r.source_photos || []).length ? `<div class="flex flex-wrap gap-2">${r.source_photos.map((p) =>
          `<a href="/api/photos/${esc(p)}" target="_blank" rel="noopener"><img src="/api/photos/${esc(p)}?w=480" alt="The original" class="h-24 rounded-lg object-cover"></a>`).join("")}</div>` : ""}
      </div>
      <div class="sticky bottom-0 flex flex-wrap gap-2 border-t border-slate-800 bg-slate-950/95 py-3 lg:col-span-2 above-tabbar">
        <button class="btn-primary">💾 Save</button>
        <a href="${isNew ? "#/add" : `#/recipe/${r.id}`}" class="btn-ghost">Cancel</a>
      </div>
    </form>`;

  const form = $("#edit", view);
  const drawPhoto = () => {
    $("#photo-box", view).innerHTML = `${photo ? `<img src="/api/photos/${esc(photo)}?w=480" alt="" class="h-24 w-32 rounded-lg object-cover">` : ""}
      <label class="btn-secondary cursor-pointer">📷 ${photo ? "Change" : "Add a photo"}<input type="file" accept="image/*" class="sr-only" id="photo-in"></label>
      ${photo ? `<button type="button" id="photo-rm" class="btn-ghost">Remove</button>` : ""}`;
    $("#photo-in", view).onchange = (e) => attempt(async () => {
      const f = e.target.files[0];
      if (!f) return;
      photo = (await post("/api/photos", { image: await shrinkPhoto(f, 1600) })).name;
      drawPhoto();
    });
    const rm = $("#photo-rm", view);
    if (rm) rm.onclick = () => { photo = ""; drawPhoto(); };
  };
  drawPhoto();
  $$('input[name="area"]', form).forEach((i) => (i.onchange = () =>
    $$('input[name="area"]', form).forEach((j) => j.parentElement.classList.toggle("on", j.checked))));

  form.onsubmit = (e) => {
    e.preventDefault();
    const f = form.elements;
    const checked = Object.fromEntries(r.ingredients.filter((i) => i.checked?.length).map((i) => [i.line, i.checked]));
    const ingredients = fromText(f.ingredients.value, "line").map((i) => ({ ...i, checked: checked[i.line] }));
    const steps = fromText(f.steps.value, "text");
    const cardOK = $('input[name="card_checked"]', view)?.checked;
    if (cardOK) [...ingredients, ...steps].forEach((x) => (x.unsure = false));
    const num = (n) => Number(f[n].value) || 0;
    const out = {
      ...r, area: form.querySelector('input[name="area"]:checked')?.value || "kitchen", title: f.title.value, summary: f.summary.value,
      photo, servings: num("servings"), prep_min: num("prep_min"), cook_min: num("cook_min"), total_min: num("total_min"),
      yield_text: f.yield_text.value, course: f.course.value.trim().toLowerCase(), cuisine: f.cuisine.value.trim(),
      protein: f.protein.value.trim().toLowerCase(), heat: Number(f.heat.value), ingredients, steps, notes: f.notes.value,
      storage: f.storage.value, source_url: f.source_url.value.trim(), source_note: f.source_note.value,
      // A photo recipe stays "check it" until someone ticks that they did.
      needs_review: !cardOK && (ingredients.some((i) => i.unsure) || steps.some((s) => s.unsure) ||
        (r.needs_review && r.source_kind === "photo")),
    };
    const btn = form.querySelector("button.btn-primary");
    attempt(() => busy(btn, "Saving…", async () => {
      const { id } = isNew ? await post("/api/recipes", out) : await put(`/api/recipes/${r.id}`, out);
      go(`#/recipe/${id}`);
    }));
  };
}

