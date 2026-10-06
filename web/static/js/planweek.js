// "✨ Plan my week": a draft of dinners for the week's empty days, picked by
// the server (what everyone home can eat, liked first, food to use up,
// quick on weeknights, within the budget, leftovers from big recipes). The
// person removes what they don't want, tries again, or adds it to the plan.
import { post } from "./api.js";
import { $, $$, esc, attempt, busy, sheet, toast, money } from "./ui.js";
import { dayLabel } from "./planpick.js";

export async function planWeek(from, state, done) {
  let seed = 0;
  let draft = await post("/api/plan/suggest", { from, days: 7, seed });
  const draw = () => {
    const days = draft.days;
    const d = sheet(`<div class="space-y-3">
      <div class="flex items-center"><h2 class="text-lg font-semibold">✨ Dinners for the week</h2>
        <button type="button" data-close class="btn-ghost ml-auto" aria-label="Close">✕</button></div>
      ${days.length ? `<ul class="space-y-2">${days.map((p, i) => `<li class="flex flex-wrap items-start gap-2">
          <span class="w-24 shrink-0 text-sm font-semibold">${esc(dayLabel(p.date))}</span>
          <span class="min-w-0 flex-1 basis-40 break-words">${p.leftovers_of ? "🍱 Leftovers: " : ""}${esc(p.recipe.title)}
            ${p.why?.length ? `<span class="block text-xs text-slate-400">${esc(p.why.join(" · "))}</span>` : ""}</span>
          <button type="button" data-drop="${i}" class="btn-ghost min-h-0 px-2 py-1 text-slate-500" aria-label="Leave this day empty">✕</button></li>`).join("")}</ul>`
        : `<p class="text-sm text-slate-400">No dinners to suggest: the week is planned, or no recipe suits everyone eating
            (and the budget). Add recipes, or check who's eating each day.</p>`}
      ${draft.cost > 0 ? `<p class="text-xs text-slate-500">About ${money(draft.cost, state.info?.currency)} from the pantry's prices${draft.weekly > 0 ? `,
        with ${money(Math.max(0, draft.weekly - draft.spent - draft.cost), state.info?.currency)} of the weekly budget left` : ""}.</p>` : ""}
      <div class="flex flex-wrap gap-2">
        ${days.length ? `<button type="button" id="keep" class="btn-primary">Add to the plan</button>` : ""}
        <button type="button" id="again" class="btn-secondary">Try again</button></div></div>`);
    $$("[data-drop]", d).forEach((b) => (b.onclick = () => {
      const gone = draft.days[Number(b.dataset.drop)];
      // Leftovers can't stay without the dinner they come from.
      draft.days = draft.days.filter((p) => p !== gone && !(!gone.leftovers_of && p.leftovers_of === gone.date));
      draw();
    }));
    $("#again", d).onclick = () => attempt(() => busy($("#again", d), "Thinking…", async () => {
      draft = await post("/api/plan/suggest", { from, days: 7, seed: ++seed });
      draw();
    }));
    const keep = $("#keep", d);
    if (keep) keep.onclick = () => attempt(() => busy(keep, "Adding…", async () => {
      const ids = {};
      for (const p of draft.days.filter((x) => !x.leftovers_of)) {
        ids[p.date] = (await post("/api/plan", { date: p.date, meal: "dinner", recipe_id: p.recipe.id })).id;
      }
      for (const p of draft.days.filter((x) => x.leftovers_of && ids[x.leftovers_of])) {
        await post("/api/plan", { date: p.date, meal: "dinner", leftovers_of: ids[p.leftovers_of] });
      }
      d.close();
      toast(`Planned ${draft.days.length} dinner${draft.days.length === 1 ? "" : "s"}`);
      done();
    }));
  };
  draw();
}
