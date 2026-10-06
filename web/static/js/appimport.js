// "From another app": an export file from Paprika, Mealie or Tandoor (or
// recipe .json files). Every recipe is saved and checked like any other;
// ones already here are skipped.
import { $, esc, attempt } from "./ui.js";

const HELP = [
  ["Paprika", "Select the recipes, then Export, and pick the Paprika format. You get a .paprikarecipes file."],
  ["Mealie", "A Mealie backup (.zip), or recipes exported as .json files (one, or a .zip of them)."],
  ["Tandoor", "Use Tandoor's Export page with the Default format. You get a .zip."],
];

export function renderAppImport(box, area) {
  box.innerHTML = `<p class="text-sm text-slate-300">Bring recipes over from another recipe app. Each one is saved with its photo and
      checked for everyone, like any recipe here. Recipes already in RecipeBank are skipped, so it's safe to load a file twice.</p>
    <dl class="space-y-2 text-sm">${HELP.map(([app, how]) => `<div><dt class="font-semibold">${app}</dt><dd class="text-slate-400">${how}</dd></div>`).join("")}</dl>
    <label class="btn-primary cursor-pointer">📦 Choose the file
      <input type="file" accept=".paprikarecipes,.paprikarecipe,.zip,.json,application/zip,application/json" class="sr-only" id="app-in"></label>
    <p id="app-out" class="text-sm" role="status"></p>`;
  const input = $("#app-in", box);
  const out = $("#app-out", box);
  input.onchange = () => attempt(async () => {
    const f = input.files[0];
    if (!f) return;
    out.textContent = "Reading the file… (a big one can take a minute)";
    try {
      const resp = await fetch(`/api/import/app?area=${area}`, { method: "POST", body: f, credentials: "same-origin",
        headers: { "X-RecipeBank": "1", "Content-Type": "application/octet-stream" } });
      const res = await resp.json().catch(() => ({}));
      if (!resp.ok) throw new Error(res.error || "The file couldn't be read.");
      out.innerHTML = `✓ From ${esc(res.app)}: added <b>${res.added}</b> recipe${res.added === 1 ? "" : "s"}${res.photos ? ` with ${res.photos} photo${res.photos === 1 ? "" : "s"}` : ""}${res.skipped ? ` (${res.skipped} ${res.skipped === 1 ? "was" : "were"} already here)` : ""}. <a class="underline" href="#/${area === "home" ? "home" : "kitchen"}">See them</a>`;
    } catch (e) {
      out.textContent = "";
      throw e;
    } finally {
      input.value = "";
    }
  });
}
