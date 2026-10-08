// "Read it later" (the Add page): a card photo or pasted recipe no AI can
// read right now waits on a list. RecipeBank reads it in the background when
// an AI works again; it then shows here (and as a phone notification) ready
// to check and save. Nothing is saved without being checked.
import { get, post, del } from "./api.js";
import { $, $$, esc, attempt, toast } from "./ui.js";
import { editRecipe } from "./editor.js";
import { setPending, peekPending, clearPending } from "./booklink.js";

const soon = (iso) => {
  const min = Math.round((new Date(iso) - Date.now()) / 60000);
  return min <= 0 ? "any moment now" : min < 90 ? `in ${min} min` : `in about ${Math.round(min / 60)} hours`;
};

function row(it) {
  const what = it.kind === "photo" ? `📷 ${it.photos.length > 1 ? `${it.photos.length} photos` : "A card photo"}` : "📋 Pasted text";
  const thumb = it.photos[0] ? `<img src="/api/photos/${esc(it.photos[0])}?w=480" alt="" class="h-12 w-12 shrink-0 rounded object-cover">` : "";
  const by = it.added_by ? ` · added by ${esc(it.added_by)}` : "";
  const rm = `<button type="button" data-later-rm="${it.id}" class="btn-ghost min-h-0 px-2 py-1 text-sm" aria-label="Take it off the list">✕</button>`;
  const [line, note, act] = {
    ready: [`✅ <b>${esc(it.title)}</b> is ready to check`, `${what}${by}`,
      `<button type="button" data-later-open="${it.id}" class="btn-primary min-h-0 py-1 text-sm">Check and save</button>`],
    failed: [`⚠️ ${what} couldn't be read`, esc(it.error),
      `<button type="button" data-later-now="${it.id}" class="btn-secondary min-h-0 py-1 text-sm">Try again</button>`],
  }[it.status] || [`⏳ ${what}: waiting for an AI`, `Next try ${soon(it.next_try)}${by}${it.error ? `. Last time: ${esc(it.error)}` : ""}`,
    `<button type="button" data-later-now="${it.id}" class="btn-secondary min-h-0 py-1 text-sm">Try now</button>`];
  return `<li class="flex flex-wrap items-center gap-2 py-2">${thumb}<span class="min-w-0 flex-1 basis-48 break-words">${line}
    <span class="block text-xs ${it.status === "failed" ? "text-amber-300" : "text-slate-400"}">${note}</span></span>${act}${rm}</li>`;
}

// laterList shows what's waiting or ready, when there is anything.
export async function laterList(box, view) {
  const data = await get("/api/later").catch(() => null);
  if (!box.isConnected) return;
  if (!data?.items.length) return (box.innerHTML = "");
  const ready = data.items.filter((it) => it.status === "ready").length;
  box.innerHTML = `<div class="card mb-4 space-y-1"><h2 class="font-semibold">📥 Read later${ready ? ` · ${ready} ready to check` : ""}</h2>
    <p class="text-xs text-slate-400">Read by themselves when an AI works again.</p>
    <ul class="divide-y divide-slate-800">${data.items.map(row).join("")}</ul></div>`;
  const again = () => laterList(box, view);
  $$("[data-later-open]", box).forEach((b) => (b.onclick = () => attempt(async () => {
    const it = data.items.find((x) => String(x.id) === b.dataset.laterOpen);
    const draft = await get(`/api/later/${it.id}`);
    let from = {};
    try { from = JSON.parse(it.source || "{}") || {}; } catch { /* kept as it came */ }
    setPending({ ...from, later: it.id }); // saving takes it off the list (and links its cookbook page or card)
    editRecipe(view, draft.recipe, draft);
  })));
  $$("[data-later-now]", box).forEach((b) => (b.onclick = () => attempt(async () => {
    await post(`/api/later/${b.dataset.laterNow}/now`);
    toast("Trying now…");
    setTimeout(again, 5000);
  })));
  $$("[data-later-rm]", box).forEach((b) => (b.onclick = () => {
    if (!confirm("Take it off the list? Its photos are deleted with it.")) return;
    attempt(async () => { await del(`/api/later/${b.dataset.laterRm}`); again(); });
  }));
}

// offerLater shows why the AI couldn't read it, with "Read it later".
export function offerLater(box, err, item, done) {
  $("[data-later-offer]", box)?.remove();
  const offer = document.createElement("div");
  offer.dataset.laterOffer = "";
  offer.className = "box-caution space-y-2";
  offer.innerHTML = `<p class="break-words">${esc(err.message)}</p>
    <button type="button" class="btn-primary">📥 Read it later</button>
    <p class="text-xs">RecipeBank keeps it, reads it by itself when an AI works again, and shows it here ready to check.</p>`;
  box.prepend(offer);
  offer.scrollIntoView({ block: "nearest" });
  $("button", offer).onclick = () => attempt(async () => {
    await post("/api/later", { ...item, source: peekPending() });
    clearPending();
    toast("Kept. It's read as soon as an AI works again.");
    done();
  });
}
