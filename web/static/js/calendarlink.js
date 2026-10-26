// Me → 📅 Meal plan in your calendar: a private link a phone's calendar app
// subscribes to, so planned meals (and events) show at the person's meal
// times. "New link" stops the old one.
import { get, post, put, del } from "./api.js";
import { $, $$, esc, attempt, toast, featureOn } from "./ui.js";

const MEALS = [["breakfast", "Breakfast"], ["lunch", "Lunch"], ["dinner", "Dinner"], ["snack", "Snack"]];

export async function renderCalendar(box, state) {
  if (!featureOn(state, "plan")) return (box.innerHTML = "");
  const paint = async (c) => {
    c = c || await get("/api/me/calendar");
    const link = c.path ? `${c.public || location.origin}${c.path}` : "";
    box.innerHTML = `<div class="card space-y-3"><h2 class="text-lg font-semibold">📅 Meal plan in your calendar</h2>
      <p class="text-sm text-slate-400">Your planned meals show in your phone's calendar (Google or Apple) at the times below, each with a link to the recipe, and stay up to date.</p>
      ${c.on ? `<div class="space-y-2">
        <div class="flex flex-wrap gap-2"><input readonly value="${esc(link)}" class="input min-w-0 flex-1 basis-56 text-xs" aria-label="Calendar link">
          <button type="button" data-copy class="btn-secondary">Copy link</button></div>
        <p class="text-xs text-slate-400">Anyone with this link can see your plan, so keep it private. <b>Apple Calendar:</b> open the link on your iPhone or Mac.
          <b>Google Calendar:</b> Other calendars → From URL${c.public ? "" : ". Google can't reach a home-only address: turn on remote access in Admin first, then copy the link again"}.</p>
        <form data-times class="grid grid-cols-2 gap-2 sm:grid-cols-4">${MEALS.map(([k, l]) => `<label class="block"><span class="label">${l}</span>
          <input type="time" name="${k}" value="${esc(c.times[k])}" class="input"></label>`).join("")}</form>
        <label class="flex items-center gap-2 text-sm"><input type="checkbox" data-events ${c.events ? "checked" : ""}> Show events too</label>
        <div class="flex flex-wrap gap-2"><button type="button" data-new class="btn-ghost">🔄 New link (stops the old one)</button>
          <button type="button" data-off class="btn-ghost">Turn off</button></div></div>`
        : `<button type="button" data-new class="btn-primary">Make my calendar link</button>`}</div>`;
    const save = () => attempt(async () => {
      const times = Object.fromEntries($$("[data-times] input", box).map((i) => [i.name, i.value]));
      await put("/api/me/calendar", { times, events: $("[data-events]", box).checked });
    }, "Saved");
    $$("[data-times] input", box).forEach((i) => (i.onchange = save));
    const ev = $("[data-events]", box);
    if (ev) ev.onchange = save;
    const copy = $("[data-copy]", box);
    if (copy) copy.onclick = async () => {
      const input = $("input[readonly]", box);
      try { await navigator.clipboard.writeText(input.value); toast("Link copied"); } catch { input.select(); document.execCommand("copy"); }
    };
    $$("[data-new]", box).forEach((b) => (b.onclick = () => {
      if (c.on && !confirm("The old link stops working, so calendars using it need the new one. Make a new link?")) return;
      attempt(async () => paint(await post("/api/me/calendar")));
    }));
    const off = $("[data-off]", box);
    if (off) off.onclick = () => attempt(async () => { await del("/api/me/calendar"); paint(); });
  };
  await paint();
}
