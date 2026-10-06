// "What can I make?": tick what's in the kitchen (the pantry is ticked
// already), type more or photograph the fridge, then see the recipes that
// need least, what's missing and what can stand in for it.
import { get, post, qs } from "./api.js";
import { $, $$, esc, attempt, busy } from "./ui.js";
import { shrinkPhoto } from "./photo.js";
import { showResults } from "./makeresults.js";

export async function renderMake(view, params, state) {
  const [stock, people] = await Promise.all([get("/api/stock?area=kitchen"), get("/api/people")]);
  // items: what might be on hand; from is "pantry", "typed" or "photo".
  const items = stock.filter((it) => it.qty > 0).map((it) => ({ name: it.name, on: true, from: "pantry" }));
  let who = people.filter((p) => !p.is_guest).map((p) => p.id);
  let photoNote = "";

  const has = (name) => items.some((it) => it.name.toLowerCase() === name.toLowerCase());
  const addItems = (names, from) => names.forEach((n) => { if ((n = n.trim()) && !has(n)) items.push({ name: n, on: true, from }); });
  const haveList = () => items.filter((it) => it.on).map((it) => it.name);
  const whoParam = () => who.join(",") || "0";

  view.innerHTML = `
    <h1 class="mb-4 text-2xl font-bold">What can I make?</h1>
    <div class="card mb-4 space-y-3">
      <h2 class="font-semibold">What you have</h2>
      <p class="text-sm text-slate-400">The pantry is ticked already. Untick what's gone and add anything else.</p>
      <div id="items" class="flex flex-wrap gap-2"></div>
      <form id="more" class="flex flex-wrap gap-2">
        <input name="food" class="input min-w-0 flex-1 basis-40" placeholder="Add foods: eggs, rice, spinach" aria-label="Add foods">
        <button class="btn-secondary">Add</button></form>
      ${state.info?.ai_ready ? `<label class="btn-secondary cursor-pointer">📷 Photo of the fridge or a shelf
        <input type="file" accept="image/*" capture="environment" class="sr-only" id="fridge"></label>` : ""}
      <p id="photo-note" class="text-sm text-amber-300"></p>
      <div><span class="label">Who's eating</span><div id="who" class="flex flex-wrap gap-2"></div></div>
      <button type="button" id="find" class="btn-primary">Find recipes</button>
    </div>
    <div id="results"></div>`;

  const drawItems = () => {
    $("#items", view).innerHTML = items.length ? items.map((it, i) => `<button type="button" data-item="${i}" aria-pressed="${it.on}"
        class="${it.on ? "chip-ok" : "chip-cat line-through opacity-60"} min-h-[2.25rem] max-w-full break-words px-3 text-left">
        ${it.on ? "✓" : "○"} ${it.from === "photo" ? "📷 " : ""}${esc(it.name)}</button>`).join("")
      : `<p class="text-sm text-slate-500">Nothing in the pantry yet: type what you have below.</p>`;
    $$("[data-item]", view).forEach((b) => (b.onclick = () => { const it = items[Number(b.dataset.item)]; it.on = !it.on; drawItems(); }));
    $("#photo-note", view).textContent = photoNote;
    $("#who", view).innerHTML = people.map((p) => `<button type="button" data-who="${p.id}" aria-pressed="${who.includes(p.id)}"
      class="${who.includes(p.id) ? "chip-ok" : "chip-cat opacity-60"} min-h-[2.25rem] max-w-full break-words px-3">${who.includes(p.id) ? "✓" : "○"} ${esc(p.name)}</button>`).join("");
    $$("[data-who]", view).forEach((b) => (b.onclick = () => {
      const id = Number(b.dataset.who);
      who = who.includes(id) ? who.filter((x) => x !== id) : [...who, id];
      drawItems();
    }));
  };
  drawItems();

  $("#more", view).onsubmit = (e) => {
    e.preventDefault();
    addItems(e.target.food.value.split(","), "typed");
    e.target.food.value = "";
    drawItems();
  };
  const fridge = $("#fridge", view);
  if (fridge) fridge.onchange = () => attempt(async () => {
    const f = fridge.files[0];
    if (!f) return;
    photoNote = "Looking at the photo… (this can take a minute)";
    drawItems();
    try {
      const res = await post("/api/make/photo", { images: [await shrinkPhoto(f)] });
      addItems(res.foods, "photo");
      photoNote = `The AI saw ${res.foods.length} food${res.foods.length === 1 ? "" : "s"} (marked 📷). Untick anything that isn't really there.`;
    } catch (e) {
      photoNote = "";
      throw e;
    } finally {
      fridge.value = "";
      drawItems();
    }
  });
  const find = $("#find", view);
  find.onclick = () => attempt(() => busy(find, "Looking…", async () => {
    const answer = await post("/api/make" + qs({ who: whoParam() }), { have: haveList() });
    showResults($("#results", view), answer, haveList, whoParam);
    $("#results", view).scrollIntoView({ behavior: "smooth", block: "start" });
  }));
}
