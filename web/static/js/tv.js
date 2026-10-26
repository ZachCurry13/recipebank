// The kitchen screen (/tv): shows a code to type on the phone in cook mode,
// then the recipe step big, with its ingredients and timers. It shows only
// what the phone sends. Built with textContent, never HTML from the phone.
const root = document.getElementById("tv");
let code = "", secret = "", version = -1, view = null, paired = false, got = 0, wake = null;

const el = (tag, cls, text) => {
  const e = document.createElement(tag);
  if (cls) e.className = cls;
  if (text !== undefined) e.textContent = text;
  return e;
};
const clock = (s) => `${Math.floor(s / 60)}:${String(Math.floor(s % 60)).padStart(2, "0")}`;

async function start() {
  const res = await fetch("/api/tv/new", { method: "POST", headers: { "X-RecipeBank": "1", "Content-Type": "application/json" }, body: "{}" });
  if (!res.ok) throw new Error((await res.json().catch(() => ({}))).error || "can't start");
  ({ code, secret } = await res.json());
  version = -1; view = null; paired = false;
  draw();
}

function draw() {
  root.replaceChildren();
  if (!paired) {
    const box = el("div", "my-auto space-y-6 text-center");
    box.append(el("p", "text-3xl text-slate-300", "🍲 RecipeBank kitchen screen"),
      el("p", "font-mono text-8xl font-bold tracking-widest text-emerald-300 md:text-9xl", code),
      el("p", "mx-auto max-w-2xl text-2xl text-slate-300", "On your phone, open a recipe, start cooking, tap 📺 and type this code."));
    root.append(box);
    return;
  }
  if (!view) {
    root.append(el("p", "my-auto text-center text-4xl text-slate-300", "✅ Connected. Start cooking on your phone."));
    return;
  }
  const head = el("div", "flex flex-wrap items-baseline gap-x-4 text-slate-400");
  head.append(el("span", "text-2xl font-semibold text-slate-200", view.title),
    el("span", "text-xl", `Step ${view.at + 1} of ${view.total}${view.section ? ` · ${view.section}` : ""}`));
  const bar = el("div", "h-2 w-full rounded-full bg-slate-800");
  const fill = el("div", "h-full rounded-full bg-emerald-500");
  fill.style.width = `${view.total ? ((view.at + 1) / view.total) * 100 : 0}%`;
  bar.append(fill);
  const text = el("p", "break-words font-medium leading-tight", view.text);
  text.style.fontSize = view.text.length > 260 ? "2.2rem" : view.text.length > 120 ? "3rem" : "4rem";
  root.append(head, bar, text);
  if (view.timers.length) {
    const row = el("div", "flex flex-wrap gap-4");
    view.timers.forEach((t, i) => { const c = el("span", "rounded-xl px-5 py-3 text-4xl font-semibold"); c.dataset.timer = i; row.append(c); });
    root.append(row);
  }
  if (view.items.length) {
    const card = el("div", "rounded-2xl border border-slate-800 p-5");
    card.append(el("h2", "mb-2 text-xl uppercase tracking-wide text-slate-400", "For this step"));
    const list = el("ul", "space-y-1 text-3xl");
    view.items.forEach((it) => {
      const li = el("li", "grid grid-cols-[9rem_minmax(0,1fr)] gap-x-4");
      li.append(el("b", "text-right tabular-nums", it.amount), el("span", "min-w-0 break-words", it.name));
      list.append(li);
    });
    card.append(list);
    root.append(card);
  }
  tick();
}

// Timers count down here from the seconds the phone sent.
function tick() {
  if (!view) return;
  root.querySelectorAll("[data-timer]").forEach((c) => {
    const t = view.timers[Number(c.dataset.timer)];
    const left = t.done ? 0 : Math.max(0, t.left - (Date.now() - got) / 1000);
    c.textContent = `⏱ ${t.label}: ${left > 0 ? clock(left) : "Done!"}`;
    c.className = `rounded-xl px-5 py-3 text-4xl font-semibold ${left > 0 ? "bg-slate-800 text-slate-100" : "bg-red-900 text-white motion-safe:animate-pulse"}`;
  });
}

async function poll() {
  try {
    const res = await fetch(`/api/tv/screen?code=${encodeURIComponent(code)}&secret=${encodeURIComponent(secret)}`);
    if (res.status === 404) return await start(); // ended (12 hours, or the server restarted): a new code
    const st = await res.json();
    if (st.version !== version) {
      version = st.version; paired = st.paired; view = st.view; got = Date.now();
      draw();
    }
  } catch { /* the network blinked: try again */ }
}

async function keepAwake() {
  try { wake = await navigator.wakeLock?.request("screen"); } catch { wake = null; }
}
document.addEventListener("visibilitychange", () => { if (document.visibilityState === "visible" && !wake) keepAwake(); });

start().then(() => { setInterval(poll, 1500); setInterval(tick, 500); keepAwake(); })
  .catch((e) => root.replaceChildren(el("p", "my-auto text-center text-3xl text-amber-300", `The kitchen screen couldn't start: ${e.message}`)));
