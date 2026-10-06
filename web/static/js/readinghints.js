// Admin → "What the photo reader has learned": misreads corrected while
// checking recipes against their cards, sent with the next card photos.
import { get, post, del } from "./api.js";
import { $, $$, esc, attempt } from "./ui.js";

export async function renderHints(box) {
  const draw = (hints) => {
    box.innerHTML = `<div class="card space-y-3">
      <h2 class="font-semibold">What the photo reader has learned</h2>
      <p class="text-sm text-slate-400">When you fix a line while checking a recipe against its card, the mix-up is
        remembered here and mentioned to the AI with the next cards. Remove any that look wrong.</p>
      ${hints.length ? `<ul class="space-y-1 text-sm">${hints.map((h) => `<li class="flex items-center gap-2">
        <span class="min-w-0 flex-1 break-words">“${esc(h.wrong)}” was really “${esc(h.right)}”
          ${h.times > 1 ? `<span class="text-xs text-slate-500">(${h.times} times)</span>` : ""}</span>
        <button type="button" data-hint-rm="${h.id}" class="btn-ghost min-h-0 py-1" aria-label="Remove this hint">✕</button></li>`).join("")}</ul>`
        : `<p class="text-sm text-slate-500">Nothing yet.</p>`}
      <form id="hint-add" class="grid gap-2 sm:grid-cols-[1fr_1fr_auto]">
        <input name="wrong" class="input" maxlength="60" placeholder="The AI read: bell pepper flakes" aria-label="What the AI read">
        <input name="right" class="input" maxlength="60" placeholder="The card says: red pepper flakes" aria-label="What the card says">
        <button class="btn-secondary">Add</button></form>
      <p class="text-xs text-slate-500">For handwritten cards, a vision model such as qwen2.5vl:7b reads best, with the AI
        server's context length at 8192 or more.</p></div>`;
    $$("[data-hint-rm]", box).forEach((b) => (b.onclick = () => attempt(async () =>
      draw((await del(`/api/admin/hints/${b.dataset.hintRm}`)).hints))));
    const f = $("#hint-add", box);
    f.onsubmit = (e) => {
      e.preventDefault();
      attempt(async () => draw((await post("/api/admin/hints", { wrong: f.wrong.value, right: f.right.value })).hints));
    };
  };
  draw((await get("/api/admin/hints")).hints);
}
