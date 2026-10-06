// One cookbook: its recipes by page (typed, or read from photos of the index
// by the AI), which of them are saved in RecipeBank, and "📷 Add it" to
// photograph a page and save the recipe, linked to the book.
import { get, post, del } from "./api.js";
import { $, $$, esc, attempt, busy, sheet, toast, canManage } from "./ui.js";
import { shrinkPhoto } from "./photo.js";
import { setPending } from "./booklink.js";
import { coverImg, editBook } from "./books.js";
import { go } from "./app.js";

// parseLines reads typed entries: "Lasagna 112", "Lasagna, p. 112" or "112 Lasagna".
export function parseLines(text) {
  const out = [];
  for (const raw of text.split("\n")) {
    const line = raw.trim();
    if (!line) continue;
    let m = line.match(/^(.*?)[\s,.:…-]+(?:(?:pp?\.?|page)\s*)?(\d+(?:\s*[-–]\s*\d+)?)$/i);
    if (m && m[1].trim()) { out.push({ title: m[1].trim(), page: m[2].replace(/\s/g, "") }); continue; }
    m = line.match(/^(\d+)\s+(.+)$/);
    out.push(m ? { title: m[2].trim(), page: m[1] } : { title: line, page: "" });
  }
  return out;
}

export async function renderBook(view, params, state) {
  const manage = canManage(state.user);
  const { book: b, entries } = await get(`/api/books/${params.id}`);
  const reload = () => renderBook(view, params, state);
  let q = "";
  view.innerHTML = `
    <a href="#/books" class="text-sm text-slate-400 underline">‹ Bookshelf</a>
    <div class="my-3 flex min-w-0 items-start gap-4">${coverImg(b, "h-28 w-20 shrink-0")}
      <div class="min-w-0 flex-1"><h1 class="break-words text-2xl font-bold">${esc(b.title)}</h1>
        ${b.author ? `<p class="break-words text-slate-400">${esc(b.author)}</p>` : ""}
        <p class="text-xs text-slate-500">${b.entries} recipe${b.entries === 1 ? "" : "s"} listed · ${b.saved} saved here${b.isbn ? ` · ISBN ${esc(b.isbn)}` : ""}</p>
        ${manage ? `<div class="mt-2 flex flex-wrap gap-2"><button id="edit" class="btn-ghost min-h-0 px-2 py-1 text-sm">✏️ Change</button>
          <button id="remove" class="btn-ghost min-h-0 px-2 py-1 text-sm text-slate-500">🗑 Remove</button></div>` : ""}</div></div>
    ${manage ? `<div class="card mb-4 space-y-3"><h2 class="font-semibold">Add its recipes</h2>
      <p class="text-sm text-slate-400">Type them one per line with the page (Lasagna 112), or photograph the index and check what the AI read.</p>
      <textarea id="lines" rows="4" class="input" placeholder="Lasagna 112&#10;Garlic bread 9"></textarea>
      <div class="flex flex-wrap gap-2"><button id="add" class="btn-secondary">Add these</button>
        <label class="btn-secondary cursor-pointer ${state.info?.ai_ready ? "" : "pointer-events-none opacity-50"}">📷 Read the index
          <input id="index" type="file" accept="image/*" multiple class="sr-only" ${state.info?.ai_ready ? "" : "disabled"}></label></div>
      ${state.info?.ai_ready ? "" : `<p class="text-xs text-slate-500">Reading the index needs the AI (Admin → AI).</p>`}</div>` : ""}
    <div class="card space-y-3">
      <input id="q" type="search" class="input" placeholder="Find a recipe in this book">
      <ul id="entries" class="divide-y divide-slate-800"></ul></div>`;
  const list = $("#entries", view);
  const draw = () => {
    const shown = entries.filter((e) => !q || e.title.toLowerCase().includes(q));
    list.innerHTML = shown.length ? shown.map((e) => `<li class="flex flex-wrap items-center gap-2 py-2">
        <span class="w-14 shrink-0 break-words text-right text-sm tabular-nums text-slate-400">${e.page ? esc(e.page) : "–"}</span>
        <span class="min-w-0 flex-1 basis-32 break-words">${esc(e.title)}</span>
        ${e.recipe_id ? `<a href="#/recipe/${e.recipe_id}" class="chip-ok">✓ Saved</a>`
          : manage ? `<button data-photo="${e.id}" class="btn-ghost min-h-0 px-2 py-1 text-sm">📷 Add it</button>` : ""}
        ${manage ? `<button data-drop="${e.id}" class="btn-ghost min-h-0 px-2 py-1 text-sm text-slate-500" aria-label="Remove from the list">✕</button>` : ""}</li>`).join("")
      : `<li class="py-2 text-sm text-slate-500">${entries.length ? "Nothing matches." : "No recipes listed yet."}</li>`;
    $$("[data-photo]", list).forEach((btn) => (btn.onclick = () => {
      const e = entries.find((x) => x.id === Number(btn.dataset.photo));
      setPending({ book: b.id, book_title: b.title, page: e.page, title: e.title });
      go("#/add?way=photo&from=book");
    }));
    $$("[data-drop]", list).forEach((btn) => (btn.onclick = () => attempt(async () => {
      await del(`/api/books/entries/${btn.dataset.drop}`);
      reload();
    })));
  };
  draw();
  $("#q", view).oninput = (e) => { q = e.target.value.trim().toLowerCase(); draw(); };
  if (!manage) return;
  $("#edit", view).onclick = () => editBook(b, reload);
  $("#remove", view).onclick = () => attempt(async () => {
    if (!confirm(`Remove "${b.title}" from the shelf? Recipes saved from it stay.`)) return;
    await del(`/api/books/${b.id}`);
    go("#/books");
  });
  const addEntries = async (list_) => {
    const res = await post(`/api/books/${b.id}/entries`, { entries: list_ });
    toast(`Added ${res.added} recipe${res.added === 1 ? "" : "s"}`);
    reload();
  };
  const add = $("#add", view);
  add.onclick = () => attempt(async () => {
    const typed = parseLines($("#lines", view).value);
    if (!typed.length) return toast("Type a recipe and its page first", true);
    await busy(add, "Adding…", () => addEntries(typed));
  });
  const input = $("#index", view);
  input.onchange = () => attempt(async () => {
    const files = [...input.files].slice(0, 3);
    if (!files.length) return;
    const images = [];
    for (const f of files) images.push(await shrinkPhoto(f, 2000));
    input.value = "";
    toast("Reading the index… (this can take a minute)");
    const res = await post("/api/books/index", { images });
    reviewIndex(res.entries, addEntries);
  });
}

// reviewIndex shows what the AI read from the index, to tick and fix before adding.
function reviewIndex(found, addEntries) {
  const d = sheet(`<div class="space-y-3">
    <div class="flex items-center"><h2 class="text-lg font-semibold">From the index</h2>
      <button type="button" data-close class="btn-ghost ml-auto" aria-label="Close">✕</button></div>
    <p class="text-sm text-slate-300">Untick anything that isn't a recipe; fix names and pages here.</p>
    <ul class="space-y-2">${found.map((e, i) => `<li class="flex items-center gap-2">
      <input type="checkbox" data-tick="${i}" checked aria-label="Add this one">
      <input data-title="${i}" class="input min-w-0 flex-1" value="${esc(e.title)}" aria-label="Name">
      <input data-page="${i}" class="input w-16 shrink-0" value="${esc(e.page)}" aria-label="Page"></li>`).join("")}</ul>
    <button type="button" id="keep" class="btn-primary">Add the ticked ones</button></div>`);
  const keep = $("#keep", d);
  keep.onclick = () => attempt(() => busy(keep, "Adding…", async () => {
    const pick = $$("[data-tick]", d).filter((t) => t.checked).map((t) => ({
      title: $(`[data-title="${t.dataset.tick}"]`, d).value.trim(), page: $(`[data-page="${t.dataset.tick}"]`, d).value.trim() }))
      .filter((e) => e.title);
    if (!pick.length) return toast("Nothing ticked", true);
    d.close();
    await addEntries(pick);
  }));
}
