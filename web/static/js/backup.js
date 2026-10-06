// Admin → "Back up and move recipes": every recipe, with its photos, the
// collections and the bookshelf, as one file; loading one adds the recipes this RecipeBank
// doesn't have yet.
import { $, attempt } from "./ui.js";

export function renderBackup(box) {
  box.innerHTML = `<div class="card space-y-3">
    <h2 class="font-semibold">📦 Back up and move recipes</h2>
    <p class="text-sm text-slate-400">Download every recipe (Kitchen and Home &amp; Care) with its photos, the collections and the bookshelf as one
      file. Load it into any RecipeBank, yours after a reinstall or a relative's, to add the recipes it doesn't have yet.
      People, the pantry and the cooking history stay where they are.</p>
    <div class="flex flex-wrap gap-2">
      <a href="/api/admin/backup" download class="btn-secondary">⬇ Download a backup</a>
      <label class="btn-secondary cursor-pointer">⬆ Load a backup<input type="file" accept=".zip,application/zip" class="sr-only" id="restore-in"></label>
    </div>
    <p id="restore-out" class="text-sm" role="status"></p></div>`;
  const input = $("#restore-in", box);
  const out = $("#restore-out", box);
  input.onchange = () => attempt(async () => {
    const f = input.files[0];
    if (!f) return;
    out.textContent = "Loading the backup…";
    try {
      const resp = await fetch("/api/admin/backup", { method: "POST", body: f, credentials: "same-origin",
        headers: { "X-RecipeBank": "1", "Content-Type": "application/zip" } });
      const res = await resp.json().catch(() => ({}));
      if (!resp.ok) throw new Error(res.error || "The backup couldn't be loaded.");
      out.innerHTML = `✓ Added <b>${res.added}</b> recipe${res.added === 1 ? "" : "s"}${res.skipped ? ` (${res.skipped} ${res.skipped === 1 ? "was" : "were"} already here)` : ""},
        ${res.photos} photo${res.photos === 1 ? "" : "s"}${res.collections ? `, ${res.collections} collection${res.collections === 1 ? "" : "s"}` : ""}${res.books ? ` and ${res.books} cookbook${res.books === 1 ? "" : "s"}` : ""}.`;
    } catch (e) {
      out.textContent = "";
      throw e;
    } finally {
      input.value = "";
    }
  });

}
