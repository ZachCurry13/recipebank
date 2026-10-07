// The bookshelf: the family's cookbooks (scanned by the barcode on the back,
// looked up in Open Library, or typed) and "Still to photograph", the pile
// of recipe cards and clippings waiting to be scanned.
import { get, post, put, del } from "./api.js";
import { $, $$, esc, attempt, busy, sheet, toast, canManage } from "./ui.js";
import { fromPhoto, scanLive, liveBlocker } from "./barcode.js";
import { photoPicker, wirePhotoPicker } from "./camera.js";
import { shrinkPhoto } from "./photo.js";
import { setPending } from "./booklink.js";
import { go } from "./app.js";

export const coverImg = (b, cls) => (b.cover
  ? `<img src="/api/photos/${encodeURIComponent(b.cover)}?w=240" alt="" class="${cls} rounded object-cover">`
  : `<div class="${cls} flex items-center justify-center rounded bg-slate-800 text-2xl" aria-hidden="true">📕</div>`);

export async function renderBooks(view, params, state) {
  const manage = canManage(state.user);
  const reload = () => renderBooks(view, params, state);
  const [{ books }, { pile }] = await Promise.all([get("/api/books"), get("/api/pile")]);
  const todo = pile.filter((c) => !c.done_at), done = pile.filter((c) => c.done_at);
  view.innerHTML = `
    <div class="mb-4 flex flex-wrap items-end gap-3">
      <div class="min-w-0 basis-full sm:basis-auto sm:flex-1"><h1 class="text-2xl font-bold">📖 Bookshelf</h1>
        <p class="text-sm text-slate-400">Your cookbooks and the recipes in them, by page, and the recipe cards still to photograph.</p></div>
      ${manage ? `<button id="scan" class="btn-primary">📷 Scan a cookbook</button><button id="type" class="btn-secondary">➕ Add by hand</button>` : ""}
    </div>
    ${books.length ? `<div class="mb-6 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">${books.map((b) => `
      <a href="#/book/${b.id}" class="card flex min-w-0 items-center gap-3">${coverImg(b, "h-20 w-14 shrink-0")}
        <span class="min-w-0"><span class="block break-words font-semibold">${esc(b.title)}</span>
          ${b.author ? `<span class="block break-words text-sm text-slate-400">${esc(b.author)}</span>` : ""}
          <span class="block text-xs text-slate-500">${b.entries} recipe${b.entries === 1 ? "" : "s"} listed${b.saved ? ` · ${b.saved} saved here` : ""}</span></span></a>`).join("")}</div>`
      : `<div class="card mb-6 text-center text-slate-400">No cookbooks yet.${manage ? " Scan the barcode on the back of one, or add it by hand." : ""}</div>`}
    <div class="card space-y-3">
      <h2 class="text-lg font-semibold">🗂️ Still to photograph (${todo.length})</h2>
      <p class="text-sm text-slate-400">Recipe cards and clippings waiting to be scanned. Each is ticked off when a recipe is saved from it.</p>
      ${manage ? `<form id="pile-add" class="flex flex-wrap gap-2"><input name="title" class="input min-w-0 flex-1 basis-40" maxlength="150" required placeholder="Which card? (Grandma's apple pie)">
        <input name="note" class="input min-w-0 flex-1 basis-40" maxlength="300" placeholder="Where it is (optional)"><button class="btn-secondary">Add</button></form>` : ""}
      ${todo.length ? `<ul class="divide-y divide-slate-800">${todo.map((c) => `<li class="flex flex-wrap items-center gap-2 py-2">
          <span class="min-w-0 flex-1 basis-40"><span class="block break-words">${esc(c.title)}</span>
            ${c.note ? `<span class="block break-words text-xs text-slate-400">${esc(c.note)}</span>` : ""}</span>
          ${manage ? `<button data-photo="${c.id}" class="btn-primary min-h-0 px-3 py-1.5 text-sm">📷 Photograph it</button>
            <button data-done="${c.id}" class="btn-ghost min-h-0 px-2 py-1.5 text-sm" title="Done without saving a recipe">✓</button>
            <button data-drop="${c.id}" class="btn-ghost min-h-0 px-2 py-1.5 text-sm text-slate-500" aria-label="Remove">✕</button>` : ""}</li>`).join("")}</ul>`
        : `<p class="text-sm text-slate-500">Nothing waiting.</p>`}
      ${done.length ? `<details><summary class="cursor-pointer text-sm text-slate-400">Done (${done.length})</summary>
        <ul class="mt-2 space-y-1 text-sm">${done.map((c) => `<li class="flex flex-wrap items-center gap-2">
          <span class="min-w-0 flex-1 break-words text-slate-400 line-through-soft">${esc(c.title)}</span>
          ${c.recipe_id ? `<a class="underline" href="#/recipe/${c.recipe_id}">Recipe</a>` : ""}
          ${manage ? `<button data-undo="${c.id}" class="btn-ghost min-h-0 px-2 py-1 text-xs">Not done</button>` : ""}</li>`).join("")}</ul></details>` : ""}
    </div>`;
  if (!manage) return;
  $("#scan", view).onclick = () => scanBook(reload);
  $("#type", view).onclick = () => editBook(null, reload);
  const add = $("#pile-add", view);
  add.onsubmit = (e) => {
    e.preventDefault();
    attempt(async () => {
      await post("/api/pile", { title: add.title.value.trim(), note: add.note.value.trim() });
      reload();
    });
  };
  const card = (id) => pile.find((c) => c.id === Number(id));
  $$("[data-photo]", view).forEach((b) => (b.onclick = () => {
    const c = card(b.dataset.photo);
    setPending({ pile: c.id, title: c.title, note: c.note });
    go("#/add?way=photo&from=pile");
  }));
  const tick = (id, done) => attempt(async () => {
    const c = card(id);
    await put(`/api/pile/${c.id}`, { title: c.title, note: c.note, done });
    reload();
  });
  $$("[data-done]", view).forEach((b) => (b.onclick = () => tick(b.dataset.done, true)));
  $$("[data-undo]", view).forEach((b) => (b.onclick = () => tick(b.dataset.undo, false)));
  $$("[data-drop]", view).forEach((b) => (b.onclick = () => attempt(async () => {
    await del(`/api/pile/${b.dataset.drop}`);
    reload();
  })));
}

// scanBook reads the barcode on a cookbook (its ISBN) and looks it up.
function scanBook(done) {
  const live = !liveBlocker();
  const d = sheet(`<div class="space-y-3">
    <div class="flex items-center"><h2 class="text-lg font-semibold">📷 Scan a cookbook</h2><button type="button" data-close class="btn-ghost ml-auto" aria-label="Close">✕</button></div>
    <p class="text-sm text-slate-400">The barcode on the back (it starts with 978 or 979).</p>
    <div class="flex flex-wrap gap-2">
      ${photoPicker("barcode", { take: "📷 Photo of the barcode", primary: true })}
      ${live ? `<button id="live" class="btn-secondary">🎥 Live camera</button>` : ""}</div>
    ${live ? "" : `<p class="text-xs text-slate-500">${esc(liveBlocker())}</p>`}
    <form id="typed" class="flex flex-wrap gap-2"><input name="code" class="input flex-1 basis-40" placeholder="Or type the ISBN">
      <button class="btn-secondary">Look up</button></form>
    <p id="msg" class="text-sm text-slate-400"></p></div>`);
  const msg = $("#msg", d);
  const say = (text, isError = false) => (msg.isConnected && d.open ? (msg.textContent = text) : toast(text, isError));
  const look = async (code) => {
    if (!code) return say("No barcode found. Try again closer, in good light, or type the ISBN.", true);
    say("Looking it up…");
    let res;
    try {
      res = await get(`/api/books/lookup?isbn=${encodeURIComponent(code)}`);
    } catch (e) {
      if (e.status !== 404) return say(e.message, true);
      d.close();
      return editBook({ isbn: code, note: "Open Library doesn't know this book yet. Type its title." }, done);
    }
    d.close();
    if (res.existing) {
      toast(`${res.existing.title} is on the shelf already`);
      return go(`#/book/${res.existing.id}`);
    }
    editBook({ ...res.book, note: "Found in Open Library. Is this the right book?" }, done);
  };
  wirePhotoPicker(d, "barcode", ([f]) => attempt(async () => {
    say("Reading the photo…");
    await look(await fromPhoto(f));
  }));
  const liveBtn = $("#live", d);
  if (liveBtn) liveBtn.onclick = () => attempt(async () => {
    const code = await scanLive(d);
    if (code) await look(code);
  });
  $("#typed", d).onsubmit = (e) => { e.preventDefault(); attempt(() => look(e.target.code.value.trim())); };
}

// editBook adds a cookbook (b: what was found, or null) or changes one (b.id).
export function editBook(b, done) {
  b = b || {};
  const d = sheet(`<form class="space-y-3">
    <div class="flex items-center"><h2 class="text-lg font-semibold">${b.id ? "Change the cookbook" : "📖 Add a cookbook"}</h2>
      <button type="button" data-close class="btn-ghost ml-auto" aria-label="Close">✕</button></div>
    ${b.note ? `<p class="text-sm text-slate-300">${esc(b.note)}</p>` : ""}
    <label class="block"><span class="label">Title</span><input name="title" class="input" required maxlength="200" value="${esc(b.title || "")}"></label>
    <label class="block"><span class="label">Author</span><input name="author" class="input" maxlength="200" value="${esc(b.author || "")}"></label>
    <label class="block"><span class="label">ISBN (optional)</span><input name="isbn" class="input" maxlength="20" value="${esc(b.isbn || "")}"></label>
    <label class="block"><span class="label">Cover photo (optional${b.isbn && !b.id ? ": Open Library's is used when there is one" : ""})</span>
      <input name="cover" type="file" accept="image/*" class="input"></label>
    <button class="btn-primary">${b.id ? "Save" : "Add to the shelf"}</button></form>`);
  const f = $("form", d);
  f.onsubmit = (e) => {
    e.preventDefault();
    attempt(() => busy($("button.btn-primary", f), "Saving…", async () => {
      const body = { title: f.title.value.trim(), author: f.author.value.trim(), isbn: f.isbn.value.trim() };
      if (f.cover.files[0]) body.image = await shrinkPhoto(f.cover.files[0], 800);
      const saved = b.id ? await put(`/api/books/${b.id}`, body) : await post("/api/books", body);
      d.close();
      if (b.id) done();
      else go(`#/book/${saved.id}`);
    }));
  };
}
