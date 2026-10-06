// Adding and editing pantry and supply items, by hand or from a barcode.
import { get, post, put, del } from "./api.js";
import { $, $$, esc, attempt, sheet, toast } from "./ui.js";
import { fromPhoto, scanLive, liveBlocker, isBarcode } from "./barcode.js";

export const LOCATIONS = {
  kitchen: ["fridge", "freezer", "pantry", "spices", "other"],
  home: ["bathroom", "laundry", "kitchen sink", "cleaning closet", "other"],
};

// editStock opens the form for item (null = new); prefill fills a new one.
export function editStock(area, item, state, done, prefill = {}) {
  const it = item || { id: 0, area, name: "", brand: "", barcode: "", qty: 1, unit: "", location: "", use_by: "",
    low_at: 0, allergens: [], traces: [], ingredients_text: "", label_source: "", ...prefill };
  const allergens = state.info?.all_allergens || [];
  const shown = allergens.filter((a) => !a.eu || state.info?.allergen_list === "eu" || it.allergens.includes(a.key) || it.traces.includes(a.key));
  const src = { off: "From the product database (Open Food Facts). Check it against the package.", parent: "Checked by a parent.", "": "" };
  const d = sheet(`
    <form id="sf" class="space-y-3">
      <div class="flex items-center"><h2 class="text-lg font-semibold">${it.id ? "Edit" : "Add"} ${area === "home" ? "a supply" : "to the pantry"}</h2>
        <button type="button" data-close class="btn-ghost ml-auto">✕</button></div>
      ${prefill.note ? `<p class="box-info">${esc(prefill.note)}</p>` : ""}
      ${prefill.scanned ? `<div class="box-info flex items-center gap-3">
        ${prefill.photo ? `<img src="${esc(prefill.photo)}" alt="The package, from the product database" class="h-20 w-16 shrink-0 rounded bg-white object-contain">` : ""}
        <div class="min-w-0 flex-1 text-sm"><p class="font-semibold">Is this what you scanned?</p>
          <p class="break-words">${esc(it.name)}${it.brand ? ` · ${esc(it.brand)}` : ""}${prefill.size ? ` · ${esc(prefill.size)}` : ""}</p>
          <p class="text-xs text-slate-500">Barcode ${esc(prefill.scanned)}</p>
          <button type="button" id="rescan" class="btn-ghost mt-1 min-h-0 py-1">✕ Not it: scan again</button></div></div>` : ""}
      <label class="block"><span class="label">Name</span><input name="name" required maxlength="120" class="input" value="${esc(it.name)}"></label>
      <label class="block"><span class="label">Brand</span><input name="brand" class="input" value="${esc(it.brand)}"></label>
      <div class="grid grid-cols-2 gap-3">
        <label class="block"><span class="label">How many</span><input name="qty" type="number" min="0" step="any" inputmode="decimal" class="input" value="${it.qty}"></label>
        <label class="block"><span class="label">Of what</span><input name="unit" class="input" value="${esc(it.unit)}" placeholder="cans, bottles, lb"></label>
        <label class="block"><span class="label">Kept in</span><input name="location" list="locs" class="input" value="${esc(it.location)}">
          <datalist id="locs">${LOCATIONS[area].map((l) => `<option value="${l}">`).join("")}</datalist></label>
        <label class="block"><span class="label">Running low at</span><input name="low_at" type="number" min="0" step="any" class="input" value="${it.low_at || ""}" placeholder="not watched"></label>
        <label class="block"><span class="label">Price for one</span><input name="price" type="number" min="0" step="0.01" inputmode="decimal" class="input" value="${it.price || ""}" placeholder="not priced"></label>
        <label class="block"><span class="label">Package size</span><input name="size" class="input" value="${esc(it.size || "")}" placeholder="16 oz, 5 lb, 12 count"></label>
        ${area === "kitchen" ? `<label class="col-span-2 block"><span class="label">Use by</span><input name="use_by" type="date" class="input" value="${esc(it.use_by)}"></label>` : ""}
      </div>
      <label class="block"><span class="label">Barcode</span><input name="barcode" inputmode="numeric" class="input" value="${esc(it.barcode)}"></label>
      <details ${it.label_source ? "open" : ""} class="rounded-lg ring-1 ring-slate-800 p-3">
        <summary class="cursor-pointer text-sm font-semibold">What the label says</summary>
        <p class="mt-1 text-xs text-slate-400">${src[it.label_source] || "Tick what the package lists. Leave all unticked if you read it and it lists none."}</p>
        <div class="mt-2 space-y-1">${shown.map((a) => `<div class="flex flex-wrap items-center gap-2 text-sm">
          <span class="min-w-0 flex-1 basis-32">${esc(a.label)}</span>
          <label class="toggle"><input type="checkbox" name="has" value="${a.key}" ${it.allergens.includes(a.key) ? "checked" : ""}> Contains</label>
          <label class="toggle"><input type="checkbox" name="may" value="${a.key}" ${it.traces.includes(a.key) ? "checked" : ""}> May contain</label></div>`).join("")}</div>
        <label class="toggle mt-2"><input type="checkbox" name="read" ${it.label_source ? "checked" : ""}> I've checked this against the package</label>
      </details>
      <div class="flex flex-wrap gap-2 pt-1"><button class="btn-primary">Save</button>
        ${it.id ? `<button type="button" id="rm" class="btn-ghost ml-auto text-rose-300">Remove</button>` : ""}</div>
    </form>`);
  $("#sf", d).onsubmit = (e) => {
    e.preventDefault();
    const f = e.target;
    const picked = (n) => $$(`input[name="${n}"]:checked`, f).map((i) => i.value);
    const labelChanged = picked("has").join() !== it.allergens.join() || picked("may").join() !== it.traces.join();
    const body = { ...it, name: f.name.value, brand: f.brand.value, qty: Number(f.qty.value) || 0, unit: f.unit.value,
      location: f.location.value.trim().toLowerCase(), use_by: f.use_by?.value || "", low_at: Number(f.low_at.value) || 0,
      price: Number(f.price.value) || 0, size: f.size.value.trim(),
      barcode: f.barcode.value.trim(), allergens: picked("has"), traces: picked("may"),
      label_source: f.read.checked ? (labelChanged || !it.label_source ? "parent" : it.label_source) : "" };
    delete body.note;
    delete body.photo;
    delete body.scanned;
    attempt(async () => {
      await (it.id ? put(`/api/stock/${it.id}`, body) : post("/api/stock", body));
      d.close();
      done();
    }, "Saved");
  };
  const rescan = $("#rescan", d);
  if (rescan) rescan.onclick = () => { d.close(); scanStock(area, state, done); };
  const rm = $("#rm", d);
  if (rm) rm.onclick = () => {
    if (confirm(`Remove ${it.name}?`)) attempt(async () => { await del(`/api/stock/${it.id}`); d.close(); done(); });
  };
}

// scanStock reads a barcode (photo, live camera or typed), looks it up and
// opens the form, or counts one more of something already in the house.
export function scanStock(area, state, done) {
  const live = !liveBlocker();
  const d = sheet(`<div class="space-y-3">
    <div class="flex items-center"><h2 class="text-lg font-semibold">Scan a barcode</h2><button type="button" data-close class="btn-ghost ml-auto">✕</button></div>
    <div class="flex flex-wrap gap-2">
      <label class="btn-primary cursor-pointer">📷 Photo of the barcode<input id="ph" type="file" accept="image/*" capture="environment" class="sr-only"></label>
      ${live ? `<button id="live" class="btn-secondary">🎥 Live camera</button>` : ""}
    </div>
    ${live ? "" : `<p class="text-xs text-slate-500">${esc(liveBlocker())}</p>`}
    <form id="typed" class="flex flex-wrap gap-2"><input name="code" inputmode="numeric" class="input flex-1 basis-40" placeholder="Or type the numbers">
      <button class="btn-secondary">Look up</button></form>
    <p id="msg" class="text-sm text-slate-400"></p></div>`);
  const msg = $("#msg", d);
  // say shows a message in the sheet, or as a toast once the sheet is gone.
  const say = (text, isError = false) => (msg.isConnected && d.open ? (msg.textContent = text) : toast(text, isError));
  const look = async (code) => {
    if (!isBarcode(code)) return say("No barcode found. Try again closer, in good light, or type the numbers.", true);
    say("Looking it up…");
    let res;
    try {
      res = await get(`/api/stock/lookup?barcode=${code}&area=${area}`);
    } catch (e) {
      if (e.status !== 404) return say(e.message, true);
      d.close();
      return editStock(area, null, state, done, { barcode: code, note: `Barcode ${code} isn't in the open product databases yet. Type the name, and tick what the label lists.` });
    }
    if (res.existing) return confirmOneMore(res.existing, code, area, state, done);
    const p = res.product;
    d.close();
    editStock(area, null, state, done, { name: p.name, brand: p.brand, barcode: code, allergens: p.allergens, traces: p.traces,
      ingredients_text: p.ingredients_text, label_source: "off", size: p.quantity || "",
      photo: p.photo || "", scanned: code });
  };
  $("#ph", d).onchange = (e) => attempt(async () => {
    const f = e.target.files[0];
    if (!f) return;
    say("Reading the photo…");
    await look(await fromPhoto(f));
  });
  const liveBtn = $("#live", d);
  if (liveBtn) liveBtn.onclick = () => attempt(async () => {
    const code = await scanLive(d); // the camera takes over the sheet, then closes it
    if (code) await look(code);
  });
  $("#typed", d).onsubmit = (e) => { e.preventDefault(); attempt(() => look(e.target.code.value.trim())); };
}

// confirmOneMore asks before counting one more of something already in the
// house, so a misread barcode never quietly changes the wrong item.
function confirmOneMore(ex, code, area, state, done) {
  const d = sheet(`<div class="space-y-3">
    <div class="flex items-center"><h2 class="text-lg font-semibold">Is this it?</h2><button type="button" data-close class="btn-ghost ml-auto">✕</button></div>
    <p class="break-words text-lg">${esc(ex.name)}${ex.brand ? ` <span class="text-slate-400">· ${esc(ex.brand)}</span>` : ""}</p>
    <p class="text-sm text-slate-400">In the house: ${+ex.qty.toFixed(2)}${ex.unit ? " " + esc(ex.unit) : ""} · barcode ${esc(code)}</p>
    <div class="flex flex-wrap gap-2"><button type="button" id="one" class="btn-primary">✓ Yes, add one</button>
      <button type="button" id="again" class="btn-secondary">✕ Not it: scan again</button></div></div>`);
  $("#one", d).onclick = () => attempt(async () => {
    const it = await post(`/api/stock/${ex.id}/adjust`, { delta: 1 });
    d.close();
    toast(`${it.name}: now ${+it.qty.toFixed(2)}`);
    done();
  });
  $("#again", d).onclick = () => { d.close(); scanStock(area, state, done); };
}
