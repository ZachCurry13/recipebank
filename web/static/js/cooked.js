// "We cooked it": log a cook with each person's 👍 or 👎 and a note. The
// recipe page shows the history, and Tonight's ideas lean on what was liked.
import { get, post, del } from "./api.js";
import { $, $$, esc, attempt, busy, sheet, toast } from "./ui.js";
import { localDate, dayLabel } from "./planpick.js";

// cookedSheet asks how it went; done runs after it's saved.
export async function cookedSheet(r, done) {
  const people = (await get("/api/people")).filter((p) => !p.is_guest);
  const thumbs = {};
  let on = localDate(), note = "";
  const draw = () => {
    const d = sheet(`<form id="cooked" class="space-y-3">
      <div class="flex items-center"><h2 class="min-w-0 break-words text-lg font-semibold">🍳 How was ${esc(r.title)}?</h2>
        <button type="button" data-close class="btn-ghost ml-auto" aria-label="Close">✕</button></div>
      <label class="block"><span class="label">Cooked on</span><input name="on" type="date" max="${localDate()}" value="${on}" class="input"></label>
      ${people.length ? `<div class="space-y-2">${people.map((p) => `<div class="flex flex-wrap items-center gap-2">
        <span class="min-w-0 flex-1 basis-32 break-words">${esc(p.name)}</span>
        ${[[1, "👍"], [-1, "👎"]].map(([v, ico]) => `<button type="button" data-thumb="${p.id}:${v}" aria-pressed="${thumbs[p.id] === v}"
          aria-label="${esc(p.name)}: ${v > 0 ? "liked it" : "didn't like it"}"
          class="${thumbs[p.id] === v ? "btn-primary" : "btn-secondary"} min-h-0 px-4 py-1.5 text-lg">${ico}</button>`).join("")}</div>`).join("")}</div>` : ""}
      <label class="block"><span class="label">Note (optional)</span>
        <textarea name="note" rows="2" maxlength="500" class="input" placeholder="Less salt next time">${esc(note)}</textarea></label>
      <button id="cooked-save" class="btn-primary">Save</button></form>`);
    const f = $("#cooked", d);
    const keep = () => { on = f.on.value; note = f.note.value; };
    $$("[data-thumb]", d).forEach((b) => (b.onclick = () => {
      keep();
      const [id, v] = b.dataset.thumb.split(":").map(Number);
      if (thumbs[id] === v) delete thumbs[id];
      else thumbs[id] = v;
      draw();
    }));
    f.onsubmit = (e) => {
      e.preventDefault();
      keep();
      attempt(() => busy(e.submitter, "Saving…", async () => {
        await post(`/api/recipes/${r.id}/cooks`, { cooked_on: on, note, thumbs });
        d.close();
        toast("Saved. Thanks!");
        done?.();
      }));
    };
  };
  draw();
}

// cookHistory is the recipe page's list of cooks.
export function cookHistory(cooks, people, manage) {
  if (!cooks?.length) return "";
  const name = Object.fromEntries(people.map((p) => [p.id, p.name]));
  const who = (c, v) => Object.entries(c.thumbs).filter(([, t]) => t === v).map(([id]) => name[id]).filter(Boolean);
  return `<div class="card"><h2 class="mb-2 font-semibold">🍳 Cooked ${cooks.length === 1 ? "once" : `${cooks.length} times`}</h2>
    <ul class="space-y-2 text-sm">${cooks.map((c) => {
      const up = who(c, 1), down = who(c, -1);
      return `<li class="flex flex-wrap items-start gap-2"><span class="min-w-0 flex-1 basis-48 break-words">
        <b>${esc(dayLabel(c.cooked_on))}</b>${up.length ? ` · 👍 ${esc(up.join(", "))}` : ""}${down.length ? ` · 👎 ${esc(down.join(", "))}` : ""}
        ${c.note ? `<span class="block text-slate-400">${esc(c.note)}</span>` : ""}</span>
        ${manage ? `<button type="button" data-cook-rm="${c.id}" class="btn-ghost min-h-0 py-1" aria-label="Remove this">✕</button>` : ""}</li>`;
    }).join("")}</ul></div>`;
}

export function wireHistory(view, done) {
  $$("[data-cook-rm]", view).forEach((b) => (b.onclick = () => {
    if (confirm("Remove this from the history?")) attempt(async () => { await del(`/api/cooks/${b.dataset.cookRm}`); done(); });
  }));
}
