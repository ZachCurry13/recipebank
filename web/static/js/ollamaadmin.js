// Admin → Ollama (from NovelCheck): find the Ollama server, see what its
// graphics card can hold, download models with a progress bar, and choose
// which RecipeBank uses for text and for photos, with no terminal needed.
import { get, post, qs } from "./api.js";
import { $, $$, esc, attempt, toast, sheet } from "./ui.js";
import { pickRows, fillPicks, picked } from "./ollamapick.js";

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
  const servers = new Map(); // address → what Find saw there
  let pollTimer = null;

  const serverCard = (srv) => `<div class="min-w-0 space-y-3 rounded-lg border border-slate-800 p-3" data-server="${esc(srv.url)}">
    <p class="break-words">✓ Ollama ${esc(srv.version)} at <code class="break-all">${esc(srv.url)}</code></p>
    <div class="space-y-2">
      <p data-gpu-text class="text-xs text-slate-300">Checking what the graphics card can hold…</p>
      <div class="flex flex-wrap items-center gap-2">
        <button type="button" data-ol="gpu" class="btn-ghost min-h-0 px-2 py-1 text-xs" title="Loads the biggest model for a moment to see what fits">🎮 Check</button>
        <select data-gpu-manual class="input w-auto py-1 text-xs" aria-label="Graphics memory">
          <option value="">…or pick its memory</option><option value="0">No graphics card</option>
          ${[4, 6, 8, 10, 12, 16, 20, 24, 32, 48].map((n) => `<option value="${n}">${n} GB</option>`).join("")}</select></div></div>
    ${pickRows()}
    <div data-ol-progress class="hidden space-y-1"><progress max="100" value="0" class="w-full"></progress><p class="text-xs text-slate-400"></p></div></div>`;

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
    servers.clear();
    data.servers.forEach((srv) => servers.set(srv.url, srv));
    $$("[data-server]", list).forEach((card) => {
      fillPicks(card, servers.get(card.dataset.server), data.catalog);
      const saved = read(vramKey(card.dataset.server));
      $("[data-gpu-manual]", card).value = saved;
      loadGPU(card, saved === "" ? {} : { vram_gb: saved });
    });
  }

  // loadGPU labels the download picks for what the graphics card can hold.
  async function loadGPU(card, params) {
    const res = await get("/api/admin/ollama/gpu" + qs({ url: card.dataset.server, ...params })).catch((e) => ({ error: e.message }));
    if (res.error) return ($("[data-gpu-text]", card).textContent = `Couldn't check the graphics card: ${res.error}`);
    $("[data-gpu-text]", card).textContent = gpuText(res.gpu, res.message);
    fillPicks(card, servers.get(card.dataset.server), res.models);
  }

  // go uses the chosen model for text or photos, downloading it first if needed.
  function go(card, kind) {
    const [model, have] = picked(card, kind);
    if (!model) return toast("Type the model's name first", true);
    if (have) return use(card, kind, model);
    pull(card, model, () => use(card, kind, model));
  }

  // pull downloads a model with a progress bar; then runs once it's done.
  async function pull(card, model, then) {
    const url = card.dataset.server;
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
          if (!(await then())) find(); // on success the page is drawn again
        }
      }
    }, 1000);
  }

  async function use(card, what, model) {
    const r = await attempt(() => post("/api/admin/ollama/use", { url: card.dataset.server, model, use: what }));
    if (!r) return false;
    toast(`${r.models} is now ${r.where}.`);
    done();
    return true;
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
    else if (act === "go") go(card, b.closest("[data-kind]").dataset.kind);
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
        <span class="block text-xs text-slate-400">${m.photos ? "📷 reads photos · " : ""}${gb(m.size)}${m.parameter_size ? ` · ${esc(m.parameter_size)} parameters` : ""}${m.quantization ? ` · ${esc(m.quantization)}` : ""}</span></span>
      ${m.in_use ? `<span class="chip-info" title="RecipeBank is set to use this model">In use</span>`
        : `<button type="button" data-del="${esc(m.name)}" class="btn-ghost min-h-0 py-1 text-sm">🗑 Delete</button>`}</li>`).join("")
      || `<li class="text-sm text-slate-500">No models yet.</li>`}</ul></div>`);
  $$("[data-del]", d).forEach((b) => (b.onclick = () => {
    const name = b.dataset.del;
    if (!confirm(`Delete ${name} from Ollama? It frees its disk space; you can download it again later.`)) return;
    attempt(async () => { await post("/api/admin/ollama/delete", { url, model: name }); d.close(); installed(url); }, `Deleted ${name}`);
  }));
}
