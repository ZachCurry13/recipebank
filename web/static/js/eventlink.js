// "🔗 Guest link" on an event: a page guests open without an account to say
// they're coming (with their allergies and diets) and what they're bringing.
// It runs out the day after the event; the host can stop it any time.
import { get, post, del } from "./api.js";
import { $, esc, attempt, busy, sheet, toast } from "./ui.js";

const until = (iso) => new Date(iso).toLocaleDateString(undefined, { weekday: "long", month: "long", day: "numeric" });

export async function eventLinkSheet(e, done) {
  const base = `/api/events/${e.id}/link`;
  let { link } = await get(base);
  const draw = () => {
    const url = link ? `${location.origin}/e/${link.token}` : "";
    const d = sheet(`<div class="space-y-3">
      <div class="flex items-center"><h2 class="min-w-0 break-words text-lg font-semibold">🔗 Guest link for ${esc(e.name)}</h2>
        <button type="button" data-close class="btn-ghost ml-auto" aria-label="Close">✕</button></div>
      <p class="text-sm text-slate-300">Guests open it without an account to say they're coming, with their allergies and diets, and
        to add what they're bringing. They see the menu marked for them, and how many are coming, but never anyone else's allergies.</p>
      ${link ? `<p class="text-sm text-slate-400">Open until <b>${esc(until(link.expires_at))}</b>.</p>
        <input id="ev-url" readonly class="input" value="${esc(url)}" aria-label="The link">
        <div class="flex flex-wrap gap-2"><button type="button" id="copy" class="btn-primary">Copy the link</button>
          ${navigator.share ? `<button type="button" id="native" class="btn-secondary">Share…</button>` : ""}
          <button type="button" id="stop" class="btn-ghost text-rose-300">Stop the link</button></div>
        ${location.protocol === "http:" ? `<p class="text-xs text-amber-300">This is RecipeBank's home-network address: guests outside your home
          need the link from its internet address (Admin → Remote access).</p>` : ""}`
      : `<button type="button" id="make" class="btn-primary">Make a guest link</button>`}</div>`);
    const make = $("#make", d);
    if (make) make.onclick = () => attempt(() => busy(make, "Making…", async () => { ({ link } = await post(base)); draw(); }));
    const copy = $("#copy", d);
    if (copy) copy.onclick = async () => {
      const input = $("#ev-url", d);
      try { await navigator.clipboard.writeText(input.value); } catch { input.select(); document.execCommand("copy"); }
      toast("Link copied");
    };
    const native = $("#native", d);
    if (native) native.onclick = () => navigator.share({ title: e.name, text: `Coming to ${e.name}? Say so here:`, url }).catch(() => {});
    const stop = $("#stop", d);
    if (stop) stop.onclick = () => attempt(async () => {
      await del(base);
      link = null;
      toast("The link is stopped. Guests who came through it stay on the list.");
      draw();
      done();
    });
  };
  draw();
}
