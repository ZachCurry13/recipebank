// Add a recipe: from a link, photos of a card or page, pasted text, or typed in.
// Every import comes back as an unsaved draft, checked for everyone.
import { post } from "./api.js";
import { $, $$, esc, attempt, busy } from "./ui.js";
import { shrinkPhoto, rotatePhoto } from "./photo.js";
import { editRecipe } from "./editor.js";
import { renderDish } from "./dish.js";

const WAYS = [
  ["link", "🔗", "From a link", "Paste the address of a recipe page."],
  ["photo", "📷", "From photos", "A handwritten card, a cookbook page or a clipping (up to 4 photos)."],
  ["text", "📋", "Paste text", "Copy the recipe from anywhere and paste it here."],
  ["dish", "🍽️", "From a dish", "A photo of a meal: similar recipes, or the AI's best guess."],
  ["type", "✍️", "Type it in", "Start from an empty recipe."],
];

export function renderAdd(view, params, state) {
  const area = params.area === "home" ? "home" : "kitchen";
  const ai = state.info?.ai_ready;
  let way = params.way || "link";
  if (way === "dish" && area === "home") way = "link";
  const ways = WAYS.filter(([k]) => k !== "dish" || area === "kitchen");
  let photos = []; // data: URLs
  const draw = () => {
    view.innerHTML = `
      <h1 class="text-2xl font-bold">➕ Add a recipe</h1>
      <p class="mb-4 text-sm text-slate-400">It's checked for everyone before you save it.</p>
      <div class="mb-4 flex flex-wrap gap-2" role="radiogroup" aria-label="Goes in">
        ${[["kitchen", "🍲 Kitchen"], ["home", "🧴 Home & Care"]].map(([a, l]) =>
          `<a href="#/add?area=${a}&way=${way}" class="pick ${area === a ? "on" : ""}">${l}</a>`).join("")}
      </div>
      <div class="mb-4 grid gap-2 sm:grid-cols-3 lg:grid-cols-5">${ways.map(([k, ico, label, help]) => `
        <button data-way="${k}" class="card min-w-0 text-left ${way === k ? "ring-2 ring-emerald-600" : ""}">
          <div class="text-2xl">${ico}</div><div class="font-semibold">${label}</div>
          <div class="text-xs text-slate-400">${help}</div></button>`).join("")}</div>
      <div id="way" class="card space-y-3"></div>`;
    $$("[data-way]", view).forEach((b) => (b.onclick = () => { way = b.dataset.way; draw(); }));
    const box = $("#way", view);
    const needAI = !ai ? `<p class="box-caution">This needs the AI, and none is set up yet. An admin can add one under Admin → AI.</p>` : "";
    if (way === "link") {
      box.innerHTML = `<form id="f" class="space-y-3"><label class="block"><span class="label">Recipe page address</span>
        <input name="url" type="url" required class="input" placeholder="https://…" inputmode="url" value="${esc(params.url || "")}"></label>
        <p class="text-xs text-slate-400">Most recipe sites include the recipe in a standard format RecipeBank reads exactly. For other pages the AI reads it${ai ? "" : " (not set up yet)"}.</p>
        <button class="btn-primary">Read the recipe</button></form>`;
      $("#f", box).onsubmit = (e) => { e.preventDefault(); run(e.submitter, "Reading the page…", "/api/import/url", { url: e.target.url.value }); };
    } else if (way === "photo") {
      box.innerHTML = `${needAI}
        <div class="flex flex-wrap gap-2">
          <label class="btn-secondary cursor-pointer">📷 Take a photo<input type="file" accept="image/*" capture="environment" class="sr-only" data-add></label>
          <label class="btn-secondary cursor-pointer">🖼️ Choose photos<input type="file" accept="image/*" multiple class="sr-only" data-add></label>
        </div>
        <p class="text-xs text-slate-400">Lay the card flat in good light. Add the back or the next page as another photo.</p>
        <div id="thumbs" class="flex flex-wrap gap-2">${photos.map((p, i) => `
          <div class="relative"><img src="${p}" alt="Photo ${i + 1}" class="h-28 rounded-lg object-cover">
          <button data-rm="${i}" class="absolute right-1 top-1 rounded-full bg-slate-900/80 px-2 text-sm text-white" aria-label="Remove">✕</button>
          <button data-rot="${i}" class="absolute bottom-1 left-1 rounded-full bg-slate-900/80 px-2 text-sm text-white" aria-label="Turn a quarter turn">⟳</button></div>`).join("")}</div>
        ${photos.length ? `<p class="text-xs text-slate-400">Is the writing the right way up? Tap ⟳ to turn a photo.</p>` : ""}
        <button id="go" class="btn-primary" ${photos.length && ai ? "" : "disabled"}>Read ${photos.length > 1 ? `these ${photos.length} photos` : "the photo"}</button>
        <p class="text-xs text-slate-500">Reading handwriting can take a minute, longer on a home AI server.</p>`;
      $$("[data-add]", box).forEach((inp) => (inp.onchange = () => attempt(async () => {
        for (const f of [...inp.files].slice(0, 4 - photos.length)) photos.push(await shrinkPhoto(f, 2000));
        draw();
      })));
      $$("[data-rm]", box).forEach((b) => (b.onclick = () => { photos.splice(Number(b.dataset.rm), 1); draw(); }));
      $$("[data-rot]", box).forEach((b) => (b.onclick = () => attempt(async () => {
        const i = Number(b.dataset.rot);
        photos[i] = await rotatePhoto(photos[i]);
        draw();
      })));
      $("#go", box).onclick = (e) => run(e.currentTarget, "The AI is reading it…", "/api/import/photo", { images: photos });
    } else if (way === "dish") {
      renderDish(box, view, ai);
    } else if (way === "text") {
      box.innerHTML = `<form id="f" class="space-y-3"><label class="block"><span class="label">The recipe</span>
        <textarea name="text" rows="12" required class="input" placeholder="Title, ingredients and steps…">${esc(params.text || "")}</textarea></label>
        ${ai ? "" : `<p class="text-xs text-slate-400">Without the AI, put the title first, then a line saying <b>Ingredients</b>, then a line saying <b>Directions</b>.</p>`}
        <button class="btn-primary">Read the recipe</button></form>`;
      $("#f", box).onsubmit = (e) => { e.preventDefault(); run(e.submitter, "The AI is reading it…", "/api/import/text", { text: e.target.text.value }); };
    } else {
      editRecipe(view, { id: 0, area, title: "", summary: "", servings: 0, heat: -1, ingredients: [], steps: [], notes: "",
        storage: "", source_kind: "manual", source_url: "", source_note: "", source_photos: [], photo: "" }, null);
    }
  };
  const run = (btn, label, path, body) => attempt(() => busy(btn, label, async () => {
    const draft = await post(path, { ...body, area });
    editRecipe(view, draft.recipe, draft);
  }));
  draw();
}

