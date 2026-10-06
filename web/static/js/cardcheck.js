// "Check this against the card": what the cross-check found on a recipe the
// AI read, and the way to say someone has checked it.
import { esc } from "./ui.js";

// cardBox explains what to look at. mode "page" adds the ✓ button (for
// parents), "edit" a tick box, "" the note only.
export function cardBox(r, problems, mode) {
  if (!r.needs_review) return "";
  const photo = r.source_kind === "photo";
  const list = (problems || []).map((p) => `<li>${esc(p)}</li>`).join("");
  const unsure = [...(r.ingredients || []), ...(r.steps || [])].some((x) => x.unsure);
  return `<div class="box-caution space-y-2">
    <p class="font-semibold">${photo ? "Check this against the card" : "Check the lines marked ?"}</p>
    ${list ? `<ul class="list-disc space-y-1 pl-5 text-sm">${list}</ul>` : ""}
    ${unsure ? `<p class="text-sm">The AI wasn't sure about lines marked <b>?</b>.</p>` : ""}
    <p class="text-sm">Until it's checked, it shows as "not sure" for anyone with an allergy.</p>
    ${mode === "page" ? `<button type="button" data-cardok class="btn-secondary">✓ ${photo ? "Checked against the card" : "Checked: it's right"}</button>` : ""}
    ${mode === "edit" ? `<label class="flex items-center gap-2 text-sm"><input type="checkbox" name="card_checked" class="h-5 w-5"> I've checked every line${photo ? " against the card" : ""}</label>` : ""}
  </div>`;
}
