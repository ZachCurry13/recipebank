// "Substitute": tap an ingredient you don't have to see what can stand in for
// it. Ideas come from the substitution table, the allergy swaps and (when
// asked) the AI; only ones that suit everyone eating are shown, and what's
// already in the pantry comes first.
import { post, qs } from "./api.js";
import { $, $$, esc, attempt, busy, toast } from "./ui.js";

// subButton is the toggle in the ingredients header.
export const subButton = (on) => `<button type="button" data-subst aria-pressed="${on}"
  class="${on ? "btn-primary" : "btn-secondary"} min-h-0 px-3 py-1.5 text-sm">↔ Substitute</button>`;

const ask = (r, line, ai, who) => post(`/api/recipes/${r.id}/substitute` + qs({ who }), { line, ai });

function panel(i, a) {
  const ideas = a.ideas.map((x) => `<li class="flex flex-wrap items-center gap-x-2 gap-y-1">
    <span class="min-w-0 flex-1 basis-40 break-words"><b>${esc(x.to)}</b>${x.note ? ` <span class="text-slate-400">· ${esc(x.note)}</span>` : ""}
      ${x.have ? `<span class="chip-ok">✓ You have it</span>` : ""}${x.by_ai ? ` <span class="chip-info">AI idea</span>` : ""}</span>
    ${x.have ? "" : `<button type="button" data-subadd="${esc(x.to)}" class="btn-ghost min-h-0 py-1" aria-label="Add ${esc(x.to)} to the shopping list">+ List</button>`}</li>`).join("");
  const skipped = a.skipped ? `<p class="text-xs text-slate-500">${a.skipped} more ${a.skipped === 1 ? "idea doesn't" : "ideas don't"} suit everyone eating.</p>` : "";
  return `<li data-subpanel class="box-info mb-2 space-y-2 text-sm">
    <p class="font-semibold">Instead of ${esc(a.food)}:</p>
    ${ideas ? `<ul class="space-y-2">${ideas}</ul>` : `<p>No swap we know suits everyone eating.</p>`}
    ${skipped}${a.ai_error ? `<p class="text-xs text-rose-300">${esc(a.ai_error)}</p>` : ""}
    ${a.ai_ready && !a.asked ? `<button type="button" data-subai="${i}" class="btn-ghost min-h-0 py-1">✨ Ask the AI for more ideas</button>` : ""}</li>`;
}

// wireSubstitute runs after each draw. st is { on, open: Map(line → answer) };
// who() gives the people eating, as the page's ?who=.
export function wireSubstitute(view, r, st, who, redraw) {
  const btn = $("[data-subst]", view);
  if (btn) btn.onclick = () => { st.on = !st.on; st.open.clear(); redraw(); };
  if (!st.on) return;
  $("[data-ing]", view)?.parentElement.insertAdjacentHTML("beforebegin",
    `<p class="mb-2 text-sm text-slate-400">Tap what you don't have. Only ideas that suit everyone eating are shown.</p>`);
  $$("[data-ing]", view).forEach((li) => {
    const i = Number(li.dataset.ing);
    const answer = st.open.get(i);
    if (answer) {
      li.classList.add("bg-amber-500/10");
      li.insertAdjacentHTML("afterend", panel(i, answer));
    }
    li.onclick = (e) => {
      if (e.target.closest("button")) return;
      attempt(async () => {
        if (st.open.has(i)) st.open.delete(i);
        else st.open.set(i, await ask(r, i, false, who()));
        redraw();
      });
    };
  });
  $$("[data-subai]", view).forEach((b) => (b.onclick = () => attempt(() => busy(b, "Asking…", async () => {
    const i = Number(b.dataset.subai);
    st.open.set(i, { ...(await ask(r, i, true, who())), asked: true });
    redraw();
  }))));
  $$("[data-subadd]", view).forEach((b) => (b.onclick = () => attempt(async () => {
    await post("/api/shopping", { text: b.dataset.subadd, area: r.area });
    toast(`Added ${b.dataset.subadd} to the shopping list`);
  })));
}
