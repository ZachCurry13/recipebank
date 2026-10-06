// "🔗 Share link": a read-only page for someone without an account, open for
// a week. It shows the recipe only: no names, allergies or card photos.
import { get, post, del } from "./api.js";
import { $, esc, attempt, busy, sheet, toast } from "./ui.js";

const until = (iso) => new Date(iso).toLocaleDateString(undefined, { weekday: "long", month: "long", day: "numeric" });

// copyText copies to the clipboard, with a fallback for plain-http addresses.
async function copyText(text, input) {
  try {
    await navigator.clipboard.writeText(text);
  } catch {
    input.select();
    document.execCommand("copy");
  }
}

export async function shareSheet(r) {
  let { share } = await get(`/api/recipes/${r.id}/share`);
  const draw = () => {
    const url = share ? `${location.origin}/s/${share.token}` : "";
    const d = sheet(`<div class="space-y-3">
      <div class="flex items-center"><h2 class="min-w-0 break-words text-lg font-semibold">🔗 Share ${esc(r.title)}</h2>
        <button type="button" data-close class="btn-ghost ml-auto" aria-label="Close">✕</button></div>
      ${share ? `<p class="text-sm text-slate-300">Anyone with this link can see the recipe until <b>${esc(until(share.expires_at))}</b>:
          the title, photo, ingredients, steps and notes. No names, allergies or card photos.</p>
        <input id="share-url" readonly class="input" value="${esc(url)}" aria-label="The link">
        <div class="flex flex-wrap gap-2"><button type="button" id="copy" class="btn-primary">Copy the link</button>
          ${navigator.share ? `<button type="button" id="native" class="btn-secondary">Share…</button>` : ""}
          <button type="button" id="stop" class="btn-ghost text-rose-300">Stop sharing</button></div>
        ${location.protocol !== "https:" ? `<p class="box-caution text-sm">This address only works at home. For a link that works anywhere,
          turn on <b>Admin → Remote access</b> and share from its https:// address.</p>` : ""}`
      : `<p class="text-sm text-slate-300">Make a link anyone can open for a week, no account needed. It shows the recipe only:
          no names, allergies or card photos.</p>
        <button type="button" id="make" class="btn-primary">Make a link</button>`}</div>`);
    const make = $("#make", d);
    if (make) make.onclick = () => attempt(() => busy(make, "Making…", async () => { share = (await post(`/api/recipes/${r.id}/share`, {})).share; draw(); }));
    const copy = $("#copy", d);
    if (copy) copy.onclick = () => attempt(async () => { await copyText(url, $("#share-url", d)); toast("Link copied"); });
    const native = $("#native", d);
    if (native) native.onclick = () => navigator.share({ title: r.title, url }).catch(() => {});
    const stop = $("#stop", d);
    if (stop) stop.onclick = () => attempt(async () => { await del(`/api/recipes/${r.id}/share`); share = null; draw(); toast("The link no longer works"); });
  };
  draw();
}
