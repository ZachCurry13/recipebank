// Events: holiday dinners and potlucks, each with who's coming, the menu and
// who brings what. Every dish is checked for everyone coming.
import { get, post, put } from "./api.js";
import { $, esc, attempt, busy, canManage, sheet } from "./ui.js";
import { dayLabel, localDate } from "./planpick.js";
import { go } from "./app.js";

export async function renderEvents(view, params, state) {
  const manage = canManage(state.user);
  const today = localDate();
  const { events } = await get("/api/events?today=" + today);
  view.innerHTML = `
    <div class="mb-4 flex flex-wrap items-end gap-3">
      <div class="min-w-0 basis-full sm:basis-auto sm:flex-1"><h1 class="text-2xl font-bold">🎉 Events</h1>
        <p class="text-sm text-slate-400">Holiday dinners and potlucks: who's coming, the menu, and who brings what.
          Every dish is checked for everyone coming.</p></div>
      ${manage ? `<button id="new" class="btn-primary">➕ New event</button>` : ""}</div>
    ${events.length ? `<div class="grid gap-3 sm:grid-cols-2">${events.map((e) => `<a href="#/event/${e.id}"
        class="card block min-w-0 hover:ring-emerald-700 ${e.date && e.date < today ? "opacity-70" : ""}">
        <h2 class="break-words text-lg font-semibold">${esc(e.name)}</h2>
        <p class="text-sm text-slate-400">${e.date ? esc(dayLabel(e.date, true)) : "No date yet"} · ${e.people} coming ·
          ${e.dishes} dish${e.dishes === 1 ? "" : "es"}</p></a>`).join("")}</div>`
      : `<div class="card text-center text-slate-400">No events yet.${manage ? " Plan one: a holiday dinner, a birthday, a potluck…" : ""}</div>`}`;
  const n = $("#new", view);
  if (n) n.onclick = () => editEvent(null);
}

// editEvent names and dates an event (null = a new one); done runs after a change.
export function editEvent(e, done) {
  const d = sheet(`<form id="evf" class="space-y-3">
    <div class="flex items-center"><h2 class="text-lg font-semibold">${e ? "Edit the event" : "New event"}</h2>
      <button type="button" data-close class="btn-ghost ml-auto" aria-label="Close">✕</button></div>
    <label class="block"><span class="label">Name</span><input name="name" required maxlength="80" class="input"
      value="${esc(e?.name || "")}" placeholder="Thanksgiving dinner"></label>
    <label class="block"><span class="label">Date</span><input name="date" type="date" class="input" value="${esc(e?.date || "")}"></label>
    <label class="block"><span class="label">Notes (optional)</span><textarea name="notes" rows="3" maxlength="2000"
      class="input" placeholder="At Grandma's, 4 pm">${esc(e?.notes || "")}</textarea></label>
    <button class="btn-primary">Save</button></form>`);
  $("#evf", d).onsubmit = (ev) => {
    ev.preventDefault();
    const f = ev.target;
    const body = { ...(e || { who: [], extra: 0 }), name: f.name.value.trim(), date: f.date.value, notes: f.notes.value };
    attempt(() => busy(ev.submitter, "Saving…", async () => {
      const res = e ? await put(`/api/events/${e.id}`, body) : await post("/api/events", body);
      d.close();
      if (done) done();
      else go(`#/event/${res.id}`);
    }));
  };
}
