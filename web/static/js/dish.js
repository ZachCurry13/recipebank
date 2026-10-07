// "From a dish": a photo of a meal → what it most likely is, the family's
// recipes most like it, and (if wanted) the AI's best-guess recipe as a draft.
import { post } from "./api.js";
import { $, esc, attempt, busy } from "./ui.js";
import { shrinkPhoto } from "./photo.js";
import { cardGrid } from "./library.js";
import { editRecipe } from "./editor.js";
import { photoPicker, wirePhotoPicker, cameraTip } from "./camera.js";

export function renderDish(box, view, ai) {
  box.innerHTML = `${ai ? "" : `<p class="box-caution">This needs the AI, and none is set up yet. An admin can add one under Admin → AI.</p>`}
    <p class="text-sm text-slate-300">Take a photo of a meal, at home or out. You'll see the family's recipes most like it, and the AI
      can draft one: a best guess, checked for everyone before you save it.</p>
    ${ai ? `<div class="flex flex-wrap gap-2">${photoPicker("dish", { take: "🍽️ Take a photo", primary: true })}</div>${cameraTip()}` : ""}
    <div id="dish-out" class="space-y-3"></div>`;
  const out = $("#dish-out", box);
  wirePhotoPicker(box, "dish", ([f]) => attempt(async () => {
    const photo = await shrinkPhoto(f, 1600);
    out.innerHTML = `<img src="${photo}" alt="Your photo" class="h-24 w-24 rounded-lg object-cover">
      <p class="text-sm text-slate-400">Looking at the photo… (this can take a minute)</p>`;
    let res;
    try {
      res = await post("/api/import/dish", { images: [photo] });
    } catch (e) {
      out.innerHTML = "";
      throw e;
    }
    const d = res.dish;
    out.innerHTML = `<div class="flex min-w-0 items-start gap-3">
        <img src="${photo}" alt="Your photo" class="h-24 w-24 shrink-0 rounded-lg object-cover">
        <div class="min-w-0"><p class="break-words text-lg font-semibold">Looks like: ${esc(d.name)}</p>
          ${d.description ? `<p class="break-words text-sm text-slate-300">${esc(d.description)}</p>` : ""}</div></div>
      <h3 class="font-semibold">Your recipes like it</h3>
      ${res.matches.length ? cardGrid(res.matches, "kitchen") : `<p class="text-sm text-slate-400">None of your recipes look like it yet.</p>`}
      <button type="button" id="dish-draft" class="btn-primary">✨ Draft a recipe for it (a best guess)</button>`;
    const draft = $("#dish-draft", out);
    draft.onclick = () => attempt(() => busy(draft, "The AI is writing it…", async () => {
      const res2 = await post("/api/import/dish/draft", d);
      editRecipe(view, res2.recipe, res2);
    }));
  }));
}
