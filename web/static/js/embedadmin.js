// Admin → AI → Search by meaning: pick an embedding model (a small model that
// turns text into numbers capturing its meaning). Recipes are indexed in the
// background; searches then also find recipes close in meaning to the words
// typed. It only helps find recipes: allergy checks never use it.
import { get, put, post } from "./api.js";
import { $, esc, attempt, toast } from "./ui.js";

const SUGGEST = [["nomic-embed-text", "Ollama (about 0.3 GB)"], ["text-embedding-3-small", "OpenAI"], ["gemini-embedding-001", "Google Gemini"]];

export async function renderEmbed(box) {
  const paint = async () => {
    const st = await get("/api/admin/embed");
    const behind = st.model && st.ready && st.indexed < st.total;
    box.innerHTML = `<div class="card space-y-3"><h2 class="font-semibold">🧭 Search by meaning</h2>
      <p class="text-sm text-slate-400">Finds recipes that fit what you mean, even without the same words: "something warm for a cold night" finds soups and stews.
        Word matches always come first. It uses a small <b>embedding model</b> on the main AI's service.</p>
      ${st.anthropic_only ? `<p class="box-caution">Claude has no embedding models. Set up an Ollama, Gemini or OpenAI AI (as the main AI, or the AI for photos) to use this.</p>` : ""}
      <form class="flex flex-wrap gap-2"><input name="model" list="embed-models" value="${esc(st.model)}" placeholder="Model name, e.g. nomic-embed-text" autocapitalize="off" spellcheck="false" class="input min-w-0 flex-1 basis-56">
        <datalist id="embed-models">${SUGGEST.map(([m, l]) => `<option value="${m}" label="${l}">`).join("")}</datalist>
        <button class="btn-primary">${st.model ? "Save" : "Turn on"}</button>${st.model ? `<button type="button" data-off class="btn-ghost">Turn off</button>` : ""}</form>
      ${st.ollama && st.model !== "nomic-embed-text" ? `<button type="button" data-pull class="btn-secondary">⬇️ Download nomic-embed-text on Ollama and use it</button>
        <div data-bar class="hidden"><progress max="100" value="0" class="w-full"></progress></div>` : ""}
      ${st.model ? `<p class="text-sm ${st.error ? "text-amber-300" : "text-slate-300"}">${st.error ? `⚠️ ${esc(st.error)}`
        : !st.ready ? "The main AI isn't set up for this yet."
        : `${behind ? "⏳" : "✓"} ${st.indexed} of ${st.total} recipes indexed${behind ? " (it goes on in the background)" : ""}.`}</p>` : ""}</div>`;
    $("form", box).onsubmit = (e) => {
      e.preventDefault();
      attempt(async () => { await put("/api/admin/embed", { model: e.target.model.value }); paint(); }, "Saved");
    };
    const off = $("[data-off]", box);
    if (off) off.onclick = () => attempt(async () => { await put("/api/admin/embed", { model: "" }); paint(); });
    const pull = $("[data-pull]", box);
    if (pull) pull.onclick = () => attempt(async () => {
      await post("/api/admin/ollama/pull", { url: st.ollama_url, model: "nomic-embed-text" });
      pull.disabled = true;
      $("[data-bar]", box).classList.remove("hidden");
      const poll = setInterval(async () => {
        const p = await get("/api/admin/ollama/pull").catch(() => null);
        if (!p) return;
        $("progress", box).value = Math.round(p.percent || 0);
        if (p.active) return;
        clearInterval(poll);
        if (p.done) {
          await put("/api/admin/embed", { model: "nomic-embed-text" }).catch(() => {});
          toast("nomic-embed-text downloaded and turned on");
        }
        paint();
      }, 1000);
    });
    if (behind) setTimeout(() => box.isConnected && !box.querySelector("input:focus") && paint().catch(() => {}), 6000);
  };
  await paint();
}
