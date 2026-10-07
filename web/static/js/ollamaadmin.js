// Admin → Ollama (from NovelCheck): find the Ollama server, see what its
// graphics card can hold, download models with a progress bar, and choose
// which RecipeBank uses for text and for photos, with no terminal needed.
import { get, post, qs } from "./api.js";
import { $, $$, esc, attempt, toast, sheet } from "./ui.js";

// How each model fits the graphics card (see internal/ollama/gpu.go).
const FIT = {
  best: ["⭐ Best for your graphics card", "text-emerald-300"],
  best_powerful: ["⭐ Best and most powerful for your graphics card", "text-emerald-300"],
  powerful: ["💪 Most powerful that fits", "text-indigo-300"],
  fits: ["✓ Fits", "text-slate-300"],
  too_big: ["⚠️ Too big (slow)", "text-amber-300"],
  cpu_ok: ["✓ OK without a graphics card", "text-slate-300"],
  cpu_slow: ["⚠️ Slow without a graphics card", "text-amber-300"],
};
const RANK = { best_powerful: 0, best: 1, cpu_ok: 2, powerful: 3, fits: 4, "": 5, cpu_slow: 6, too_big: 7 };
const GB = 1 << 30;
const vramKey = (url) => `rb:vram:${url}`;

function gpuText(g, msg) {
  const gb = (b) => (b / GB).toFixed(b >= 10 * GB ? 0 : 1);
  const text = {
    none: "⚠️ Ollama is running without a graphics card, so small models are best. If this computer has one, turn it on in the Ollama app's settings (on TrueNAS: GPU → allocate).",
    about: `🎮 About ${gb(g.vram_bytes)} GB of graphics memory (measured with ${g.basis}).`,
    at_least: `🎮 At least ${gb(g.vram_bytes)} GB of graphics memory (${g.basis} fits entirely). Bigger models may fit too: pick your card's size to be sure.`,
    manual: `🎮 ${gb(g.vram_bytes)} GB of graphics memory (your choice).`,
  }[g.kind] || "Not measured yet: press Check, or pick your card's memory.";
  return msg ? `${text} (${msg})` : text;
}

const store = (k, v) => { try { v === null ? localStorage.removeItem(k) : localStorage.setItem(k, v); } catch { /* private mode */ } };
const read = (k) => { try { return localStorage.getItem(k) || ""; } catch { return ""; } };

// renderOllama draws the card; done runs after a model is chosen (settings changed).
export function renderOllama(box, s, done) {
  box.innerHTML = `<div class="card space-y-3">
    <h2 class="font-semibold">🦙 Ollama: find, download and choose models</h2>
    <p class="text-sm text-slate-400">For an AI on your own network. Install the Ollama app first (on TrueNAS: Apps → Discover Apps, and turn
      its graphics card on there). Then find it here. To use a bigger computer for photos, type its address and press Find.</p>
    <div class="flex flex-wrap gap-2"><input data-ol-url class="input min-w-0 flex-1 basis-56" placeholder="Address, e.g. 192.168.1.50:11434"
        value="${esc((s.llm_base_url || "").replace(/\/v1\/?$/, ""))}">
      <button type="button" data-ol="find" class="btn-primary">Find Ollama</button>
      <button type="button" data-ol="models" class="btn-secondary">🧹 Installed models</button></div>
    <div data-ol-servers class="space-y-3"></div></div>`;
  const list = $("[data-ol-servers]", box);
  let pollTimer = null;

  const serverCard = (srv) => `<div class="min-w-0 space-y-3 rounded-lg border border-slate-800 p-3" data-server="${esc(srv.url)}">
    <p class="break-words">✓ Ollama ${esc(srv.version)} at <code class="break-all">${esc(srv.url)}</code></p>
    ${srv.models.length ? `<div class="space-y-2"><p class="label mb-0">Use a downloaded model</p>
      <div class="flex flex-wrap gap-2"><select data-installed class="input min-w-0 flex-1 basis-48">${srv.models.map((m) => `<option>${esc(m)}</option>`).join("")}</select>
        <button type="button" data-ol="text" class="btn-secondary">Use for text</button>
        <button type="button" data-ol="photos" class="btn-secondary">Use for photos</button></div>
      <p class="text-xs text-slate-500">Text: pasted recipes, organizing what a card says, swaps and ideas. Photos: recipe cards, receipts, the
        fridge. A photo model on the main AI's own Ollama reads its photos; on another computer it becomes the second AI for photos.</p></div>` : ""}
    <div class="space-y-2"><p class="label mb-0">Download a model</p>
      <p data-gpu-text class="text-xs text-slate-300">Checking what the graphics card can hold…</p>
      <div class="flex flex-wrap items-center gap-2">
        <button type="button" data-ol="gpu" class="btn-ghost min-h-0 px-2 py-1 text-xs" title="Loads the biggest model for a moment to see what fits">🎮 Check</button>
        <select data-gpu-manual class="input w-auto py-1 text-xs" aria-label="Graphics memory">
          <option value="">…or pick its memory</option><option value="0">No graphics card</option>
          ${[4, 6, 8, 10, 12, 16, 20, 24, 32, 48].map((n) => `<option value="${n}">${n} GB</option>`).join("")}</select></div>
      <div class="flex flex-wrap gap-2"><select data-ol-model class="input min-w-0 flex-1 basis-56"></select>
        <button type="button" data-ol="pull" class="btn-secondary">Download</button></div>
      <p data-model-note class="text-xs text-slate-400"></p>
      <div data-ol-progress class="hidden space-y-1"><progress max="100" value="0" class="w-full"></progress><p class="text-xs text-slate-400"></p></div>
    </div></div>`;

  async function find() {
    list.innerHTML = `<p class="text-sm text-slate-400">Looking for Ollama…</p>`;
    const data = await attempt(() => get("/api/admin/ollama/find" + qs({ url: $("[data-ol-url]", box).value.trim() })));
    if (!data) return (list.innerHTML = "");
    if (!data.servers.length) {
      list.innerHTML = `<p class="text-sm text-amber-300">Ollama wasn't found. Check the Ollama app is running, then type its address above
        (the computer's address and the port shown on the app, usually 11434) and try again.</p>`;
      return;
    }
    list.innerHTML = data.servers.map(serverCard).join("");
    $$("[data-server]", list).forEach((card) => {
      const saved = read(vramKey(card.dataset.server));
      $("[data-gpu-manual]", card).value = saved;
      loadGPU(card, saved === "" ? {} : { vram_gb: saved });
    });
  }

  // loadGPU labels the download list for what the graphics card can hold.
  async function loadGPU(card, params) {
    const res = await get("/api/admin/ollama/gpu" + qs({ url: card.dataset.server, ...params })).catch((e) => ({ error: e.message }));
    if (res.error) return ($("[data-gpu-text]", card).textContent = `Couldn't check the graphics card: ${res.error}`);
    $("[data-gpu-text]", card).textContent = gpuText(res.gpu, res.message);
    const sorted = (ms) => [...ms].sort((a, b) => RANK[a.fit] - RANK[b.fit] || a.size_gb - b.size_gb);
    const opts = (ms) => sorted(ms).map((m) => `<option value="${esc(m.name)}" data-fit="${m.fit}" data-note="${esc(m.note)}">${esc(m.label)} · ${m.size_gb} GB${
      m.fit ? " · " + FIT[m.fit][0] : ""}</option>`).join("");
    const sel = $("[data-ol-model]", card);
    sel.innerHTML = `<optgroup label="For text">${opts(res.models.text)}</optgroup><optgroup label="For photos">${opts(res.models.photos)}</optgroup>`;
    const note = () => {
      const o = sel.selectedOptions[0];
      const [label, cls] = FIT[o?.dataset.fit] || ["", "text-slate-400"];
      $("[data-model-note]", card).className = `text-xs ${cls}`;
      $("[data-model-note]", card).textContent = o ? `${o.dataset.note}${label ? ` · ${label}` : ""}` : "";
    };
    sel.onchange = note;
    note();
  }

  async function pull(card) {
    const url = card.dataset.server, model = $("[data-ol-model]", card).value;
    if (!(await attempt(() => post("/api/admin/ollama/pull", { url, model })))) return;
    const prog = $("[data-ol-progress]", card);
    prog.classList.remove("hidden");
    clearInterval(pollTimer);
    pollTimer = setInterval(async () => {
      const st = await get("/api/admin/ollama/pull").catch(() => null);
      if (!st) return;
      $("progress", prog).value = Math.round(st.percent || 0);
      $("p", prog).textContent = st.error ? `Problem: ${st.error}` : `${st.status || "working"} · ${Math.round(st.percent || 0)}%`;
      if (!st.active) {
        clearInterval(pollTimer);
        if (st.done) {
          toast(`${st.model} downloaded`);
          find();
        }
      }
    }, 1000);
  }

  async function use(card, what) {
    const r = await attempt(() => post("/api/admin/ollama/use", { url: card.dataset.server, model: $("[data-installed]", card).value, use: what }));
    if (!r) return;
    toast(`${r.models} is now ${r.where}.`);
    done();
  }

  box.addEventListener("change", (e) => {
    const m = e.target.closest("[data-gpu-manual]");
    if (!m) return;
    const card = m.closest("[data-server]");
    store(vramKey(card.dataset.server), m.value);
    loadGPU(card, m.value === "" ? {} : { vram_gb: m.value });
  });
  box.addEventListener("click", (e) => {
    const b = e.target.closest("[data-ol]");
    if (!b) return;
    const card = b.closest("[data-server]");
    const act = b.dataset.ol;
    if (act === "find") find();
    else if (act === "models") installed($("[data-ol-url]", box).value.trim());
    else if (act === "pull") pull(card);
    else if (act === "text" || act === "photos") use(card, act);
    else if (act === "gpu") {
      $("[data-gpu-text]", card).textContent = "Checking… this loads the biggest model for a moment (up to a minute).";
      $("[data-gpu-manual]", card).value = "";
      store(vramKey(card.dataset.server), null);
      loadGPU(card, { probe: "1" });
    }
  });
  return () => clearInterval(pollTimer);
}

// installed lists an Ollama's models with their disk space, to delete unused ones.
async function installed(url) {
  if (!url) return toast("Type the Ollama address first (or press Find Ollama)", true);
  const data = await attempt(() => get("/api/admin/ollama/models" + qs({ url })));
  if (!data) return;
  const gb = (bytes) => `${(bytes / 1e9).toFixed(1)} GB`;
  const d = sheet(`<div class="space-y-3">
    <div class="flex items-start gap-3"><h2 class="min-w-0 break-words text-lg font-semibold">🧹 Models on ${esc(url)}</h2>
      <button type="button" data-close class="btn-ghost ml-auto" aria-label="Close">✕</button></div>
    <p class="text-sm text-slate-400">${data.models.length} model${data.models.length === 1 ? "" : "s"} using <b>${gb(data.total)}</b>. Delete the ones you
      don't use to free space; you can download them again any time.</p>
    <ul class="space-y-2">${data.models.map((m) => `<li class="flex flex-wrap items-center gap-2 rounded-lg border border-slate-800 p-3">
      <span class="min-w-0 flex-1"><b class="break-all">${esc(m.name)}</b>
        <span class="block text-xs text-slate-400">${gb(m.size)}${m.parameter_size ? ` · ${esc(m.parameter_size)} parameters` : ""}${m.quantization ? ` · ${esc(m.quantization)}` : ""}</span></span>
      ${m.in_use ? `<span class="chip-info" title="RecipeBank is set to use this model">In use</span>`
        : `<button type="button" data-del="${esc(m.name)}" class="btn-ghost min-h-0 py-1 text-sm">🗑 Delete</button>`}</li>`).join("")
      || `<li class="text-sm text-slate-500">No models yet.</li>`}</ul></div>`);
  $$("[data-del]", d).forEach((b) => (b.onclick = () => {
    const name = b.dataset.del;
    if (!confirm(`Delete ${name} from Ollama? It frees its disk space; you can download it again later.`)) return;
    attempt(async () => { await post("/api/admin/ollama/delete", { url, model: name }); d.close(); installed(url); }, `Deleted ${name}`);
  }));
}
