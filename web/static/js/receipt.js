// "🧾 Scan a receipt": photos of a store receipt → the lines the AI read,
// each matched to a pantry or supply item where one fits. The person ticks
// what's right (and fixes names, amounts, prices); saving adds the amounts
// and remembers the prices for the budget.
import { post } from "./api.js";
import { $, $$, esc, attempt, busy, sheet, toast, money } from "./ui.js";
import { shrinkPhoto } from "./photo.js";

export function scanReceipt(state, done) {
  const ai = state.info?.ai_ready;
  const d = sheet(`<div class="space-y-3">
    <div class="flex items-center"><h2 class="text-lg font-semibold">🧾 Scan a receipt</h2>
      <button type="button" data-close class="btn-ghost ml-auto" aria-label="Close">✕</button></div>
    ${ai ? "" : `<p class="box-caution">This needs the AI, and none is set up yet. An admin can add one under Admin → AI.</p>`}
    <p class="text-sm text-slate-300">Take a photo of the receipt, flat and close. A long one can take up to 3 photos, top to
      bottom. You'll check every line before anything is added.</p>
    <label class="btn-primary cursor-pointer ${ai ? "" : "pointer-events-none opacity-50"}">📷 Take or choose photos
      <input type="file" accept="image/*" multiple class="sr-only" id="rc-in" ${ai ? "" : "disabled"}></label>
    <div id="rc-out" class="space-y-3"></div></div>`);
  const input = $("#rc-in", d);
  const out = $("#rc-out", d);
  input.onchange = () => attempt(async () => {
    const files = [...input.files].slice(0, 3);
    if (!files.length) return;
    const images = [];
    for (const f of files) images.push(await shrinkPhoto(f, 2000));
    input.value = "";
    out.innerHTML = `<p class="text-sm text-slate-400">Reading the receipt… (this can take a minute)</p>`;
    let res;
    try {
      res = await post("/api/stock/receipt", { images });
    } catch (e) {
      out.innerHTML = "";
      throw e;
    }
    review(d, out, res.lines, state, done);
  });
}

function review(d, out, lines, state, done) {
  const cur = state.info?.currency;
  out.innerHTML = `<p class="text-sm text-slate-300">Untick anything that's wrong or you don't keep track of. Names, amounts
      and prices can be fixed here. Prices are for the whole line.</p>
    <ul class="space-y-3">${lines.map((l, i) => `<li class="rounded-lg border border-slate-800 p-3">
      <label class="flex items-start gap-2"><input type="checkbox" data-tick="${i}" class="mt-1" checked>
        <input data-name="${i}" class="input min-w-0 flex-1" value="${esc(l.name)}" aria-label="Name"></label>
      <div class="mt-2 flex flex-wrap items-center gap-2 pl-6 text-sm">
        <label class="flex items-center gap-1">How many <input data-qty="${i}" type="number" min="0.01" max="100" step="any"
          class="input w-20" value="${l.qty}"></label>
        <label class="flex items-center gap-1">Paid <input data-price="${i}" type="number" min="0" max="10000" step="0.01"
          class="input w-24" value="${l.price || ""}" placeholder="${esc(money(0, cur))}"></label>
        <select data-match="${i}" class="input min-w-0 flex-1 basis-48" aria-label="Where it goes">
          ${l.match ? `<option value="${l.match.id}">Adds to: ${esc(l.match.name)}</option>` : ""}
          <option value="0:kitchen" ${!l.match && l.area === "kitchen" ? "selected" : ""}>New in the pantry</option>
          <option value="0:home" ${!l.match && l.area === "home" ? "selected" : ""}>New in supplies</option>
        </select></div></li>`).join("")}</ul>
    <div class="flex flex-wrap gap-2"><button type="button" id="rc-save" class="btn-primary">Add the ticked lines</button></div>`;
  const save = $("#rc-save", out);
  const sync = () => {
    const n = $$("[data-tick]", out).filter((b) => b.checked).length;
    save.disabled = !n;
    save.textContent = n ? `Add ${n} ticked line${n === 1 ? "" : "s"}` : "Nothing ticked";
  };
  $$("[data-tick]", out).forEach((b) => (b.onchange = sync));
  sync();
  save.onclick = () => attempt(() => busy(save, "Adding…", async () => {
    const pick = [];
    for (const b of $$("[data-tick]", out)) {
      if (!b.checked) continue;
      const i = b.dataset.tick;
      const [id, area] = $(`[data-match="${i}"]`, out).value.split(":");
      pick.push({ ...lines[i], name: $(`[data-name="${i}"]`, out).value.trim(), qty: Number($(`[data-qty="${i}"]`, out).value) || 1,
        price: Number($(`[data-price="${i}"]`, out).value) || 0, match_id: Number(id) || 0, area: area || lines[i].area });
    }
    const res = await post("/api/stock/receipt/apply", { lines: pick });
    d.close();
    toast(`Added ${res.added} new, updated ${res.updated}`);
    done();
  }));
}
