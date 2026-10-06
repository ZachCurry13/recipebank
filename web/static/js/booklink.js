// Where a recipe came from: a cookbook's page, or a card from the pile. The
// Add page remembers it ("pending") while the recipe is read and checked;
// saving the new recipe links it to the page or ticks the card off.
import { get, put } from "./api.js";
import { $, esc, attempt, sheet, toast, canManage } from "./ui.js";

const KEY = "rb:pending-source";

export function setPending(p) {
  try { sessionStorage.setItem(KEY, JSON.stringify({ ...p, at: Date.now() })); } catch { /* private mode */ }
}

export function peekPending() {
  try {
    const p = JSON.parse(sessionStorage.getItem(KEY) || "null");
    return p && Date.now() - p.at < 3 * 3600e3 ? p : null; // a few hours at most
  } catch { return null; }
}

export function clearPending() {
  try { sessionStorage.removeItem(KEY); } catch { /* private mode */ }
}

// afterSave links a newly saved recipe to where it came from, if anywhere.
export async function afterSave(id) {
  const p = peekPending();
  clearPending();
  if (!p) return;
  try {
    if (p.book) await put(`/api/recipes/${id}/book`, { book_id: p.book, page: p.page || "" });
    if (p.pile) await put(`/api/pile/${p.pile}`, { title: p.title, note: p.note || "", done: true, recipe_id: id });
  } catch (e) {
    toast(`Saved, but it couldn't be linked to where it came from: ${e.message}`, true);
  }
}

// pendingNote is the line the Add page shows while a source is remembered.
export function pendingNote(p) {
  if (!p) return "";
  const what = p.book ? `${esc(p.book_title || "the cookbook")}${p.page ? `, page ${esc(p.page)}` : ""}` : `the card "${esc(p.title)}"`;
  return `<p class="box-info">Adding a recipe from ${what}. It's linked when you save it.
    <button type="button" id="pending-clear" class="underline">Not from there</button></p>`;
}

// bookLine is the recipe page's "From <cookbook>, page N" (and a button for parents).
export function bookLine(book, user) {
  const from = book ? `📖 From <a class="underline" href="#/book/${book.book_id}">${esc(book.book_title)}</a>${book.page ? `, page ${esc(book.page)}` : ""}` : "";
  const btn = canManage(user) ? `<button type="button" id="book-link" class="btn-ghost min-h-0 px-2 py-1 text-xs">${book ? "Change" : "📖 From a cookbook?"}</button>` : "";
  return from || btn ? `<p class="mt-1 flex flex-wrap items-center gap-x-2 text-sm text-slate-400">${from}${btn}</p>` : "";
}

// wireBookLink lets a parent say which cookbook page a recipe is from.
export function wireBookLink(view, r, book, done) {
  const btn = $("#book-link", view);
  if (btn) btn.onclick = () => attempt(async () => {
    const { books } = await get("/api/books");
    if (!books.length) {
      toast("Add your cookbooks on the Bookshelf first");
      return;
    }
    const d = sheet(`<form class="space-y-3">
      <div class="flex items-center"><h2 class="text-lg font-semibold">📖 Which cookbook is it from?</h2>
        <button type="button" data-close class="btn-ghost ml-auto" aria-label="Close">✕</button></div>
      <label class="block"><span class="label">Cookbook</span><select name="book" class="input">
        <option value="0">None</option>${books.map((b) => `<option value="${b.id}" ${book?.book_id === b.id ? "selected" : ""}>${esc(b.title)}</option>`).join("")}
      </select></label>
      <label class="block"><span class="label">Page</span><input name="page" class="input" maxlength="12" inputmode="numeric" value="${esc(book?.page || "")}"></label>
      <button class="btn-primary">Save</button></form>`);
    const f = $("form", d);
    f.onsubmit = (e) => {
      e.preventDefault();
      attempt(async () => {
        await put(`/api/recipes/${r.id}/book`, { book_id: Number(f.book.value), page: f.page.value.trim() });
        d.close();
        done();
      });
    };
  });
}

// bookHits is the Library's "In your cookbooks" list for a search.
export function bookHits(entries) {
  if (!entries?.length) return "";
  return `<div class="card mt-4"><h2 class="label">📖 In your cookbooks</h2><ul class="space-y-1">${entries.map((e) =>
    `<li class="flex flex-wrap items-center gap-x-2"><span class="min-w-0 break-words">${esc(e.title)}</span>
      <a class="text-sm text-slate-400 underline" href="#/book/${e.book_id}">${esc(e.book_title)}${e.page ? `, page ${esc(e.page)}` : ""}</a>
      ${e.recipe_id ? `<a class="chip-ok" href="#/recipe/${e.recipe_id}">✓ Saved</a>` : ""}</li>`).join("")}</ul></div>`;
}
