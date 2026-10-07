// Admin → AI check-up (from NovelCheck): whether each AI is answering (asked
// for its models, which costs nothing), what the online ones cost this month,
// newer versions of the Ollama models in use (Update downloads them again),
// and the speed test on a made-up recipe card.
import { get, post } from "./api.js";
import { $, $$, esc, attempt, toast } from "./ui.js";

const NAMES = { main: "Main AI", photo: "AI for photos" };
const JOBS = { text: "Words of a card", photo: "Photo of a card" };
const TEST = { main: "Test the main AI", photo: "Test the photo AI" };
const ago = (t) => (t ? new Date(t).toLocaleString([], { month: "short", day: "numeric", hour: "numeric", minute: "2-digit" }) : "");
const secs = (s) => (s >= 90 ? `${Math.round(s / 60)} min` : `${s.toFixed(s < 10 ? 1 : 0)} s`);

export function renderAITools(box) {
  let timer = null;
  const paint = async () => {
    const [health, st] = await Promise.all([get("/api/admin/ai/health").catch((e) => ({ error: e.message, ais: [] })), get("/api/admin/aitools")]);
    const results = Object.values(st.results || {}).sort((a, b) => a.ai.localeCompare(b.ai) || b.job.localeCompare(a.job) || a.seconds - b.seconds);
    box.innerHTML = `<div class="card space-y-4">
      <h2 class="font-semibold">🩺 AI check-up</h2>
      ${health.ais.length ? `<ul class="space-y-1 text-sm">${health.ais.map((h) => `<li class="break-words">${statusLine(h)}</li>`).join("")}</ul>`
        : `<p class="text-sm text-slate-400">No AI is set up yet.</p>`}
      ${health.cost?.total > 0.005 ? `<p class="text-sm text-slate-400">Online AI this month: about $${health.cost.total.toFixed(2)}.</p>` : ""}
      <div class="space-y-2"><p class="label mb-0">Newer model versions</p>
        ${st.updates.length ? st.updates.map((u) => `<div class="flex flex-wrap items-center gap-2 text-sm">
            <span class="min-w-0 flex-1 break-words">💡 A newer <b>${esc(u.model)}</b> is out.</span>
            <button type="button" data-update="${esc(u.model)}" data-server="${esc(u.server)}" class="btn-secondary min-h-0 py-1 text-sm">Update</button></div>`).join("")
          : `<p class="text-sm text-slate-400">${st.checked_at ? `Your Ollama models are up to date (checked ${ago(st.checked_at)}).` : "RecipeBank checks your Ollama models for newer versions once a day."}</p>`}
        ${st.check_error ? `<p class="text-xs text-amber-300">${esc(st.check_error)}</p>` : ""}
        <button type="button" data-check class="btn-ghost min-h-0 py-1 text-sm">Check now</button></div>
      <div class="space-y-2"><p class="label mb-0">⏱ Speed test</p>
        <p class="text-xs text-slate-400">Times each model on a made-up recipe card: its words for the text models, its photo for the photo models,
          and says whether they read it right.</p>
        <div class="flex flex-wrap gap-2">${health.ais.map((h) => `<button type="button" data-bench="${h.which}" class="btn-secondary min-h-0 py-1 text-sm"
          ${st.benching ? "disabled" : ""}>${TEST[h.which]}</button>`).join("")}</div>
        ${st.benching ? `<p class="text-sm text-slate-300">Testing… a model on your own computer can take a few minutes.</p>` : ""}
        ${results.length ? `<ul class="divide-y divide-slate-800 text-sm">${results.map((b) => `<li class="flex flex-wrap items-center gap-x-3 gap-y-1 py-2">
            <span class="min-w-0 flex-1 basis-40 break-all font-medium">${esc(b.model)}</span>
            <span class="text-xs text-slate-400">${JOBS[b.job]} · ${esc(NAMES[b.ai])}</span>
            ${b.error ? `<span class="basis-full break-words text-xs text-amber-300">${esc(b.error)}</span>`
              : `<span class="tabular-nums">${secs(b.seconds)}</span><span>${b.read_right ? "✓ read it right" : "⚠️ missed parts"}</span>`}</li>`).join("")}</ul>` : ""}
      </div></div>`;
    if (st.benching && !timer) timer = setInterval(async () => {
      const now = await get("/api/admin/aitools").catch(() => null);
      if (now && !now.benching) { clearInterval(timer); timer = null; paint(); }
    }, 3000);
    wire();
  };
  const wire = () => {
    const check = $("[data-check]", box);
    if (check) check.onclick = () => attempt(async () => { await post("/api/admin/aitools/check-updates"); toast("Checking… this takes a minute"); setTimeout(paint, 20000); });
    $$("[data-bench]", box).forEach((b) => (b.onclick = () => attempt(async () => { await post("/api/admin/aitools/bench", { which: b.dataset.bench }); paint(); })));
    $$("[data-update]", box).forEach((b) => (b.onclick = () => attempt(async () => {
      await post("/api/admin/ollama/pull", { url: b.dataset.server, model: b.dataset.update });
      b.disabled = true;
      b.textContent = "Downloading…";
      const poll = setInterval(async () => {
        const st = await get("/api/admin/ollama/pull").catch(() => null);
        if (!st || st.active) return;
        clearInterval(poll);
        if (st.done) {
          await post("/api/admin/aitools/updated", { server: b.dataset.server, model: b.dataset.update });
          toast(`${b.dataset.update} is up to date`);
          paint();
        } else toast(st.error || "The download stopped", true);
      }, 2000);
    })));
  };
  paint().catch(() => {});
  return () => clearInterval(timer);
}

function statusLine(h) {
  const name = `<b>${NAMES[h.which]}</b>`;
  if (h.ok) return `✓ ${name}: ${h.checked ? "answering" : "an online service (not checked here)"}`;
  if (h.missing?.length) return `⚠️ ${name}: answering, but it doesn't have ${esc(h.missing.join(", "))}. Download it in the Ollama card, or check the name.`;
  return `✕ ${name}: ${esc(h.error || "not answering")}`;
}
