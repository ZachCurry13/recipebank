// Cook mode: one step at a time in big text, the step's ingredients, timers
// that ring, and the screen kept on where the browser allows it.
import { $, $$, esc } from "./ui.js";
import { post, del } from "./api.js";
import { amountLine, temps, timersIn } from "./units.js";
import { canSpeak, canListen, voiceOn, setVoice, speak, stopSpeaking, listen } from "./cookvoice.js";

let wake = null;
const timers = []; // {label, ends, done}
let tick = null;

async function keepAwake() {
  try {
    wake = await navigator.wakeLock?.request("screen");
  } catch {
    wake = null;
  }
}

function beep() {
  try {
    const ctx = new AudioContext();
    for (let i = 0; i < 3; i++) {
      const o = ctx.createOscillator();
      const g = ctx.createGain();
      o.frequency.value = 880;
      o.connect(g).connect(ctx.destination);
      g.gain.setValueAtTime(0.3, ctx.currentTime + i * 0.5);
      o.start(ctx.currentTime + i * 0.5);
      o.stop(ctx.currentTime + i * 0.5 + 0.3);
    }
  } catch {
    /* no sound available */
  }
  navigator.vibrate?.([300, 150, 300]);
}

const clock = (s) => `${Math.floor(s / 60)}:${String(Math.floor(s % 60)).padStart(2, "0")}`;

// usedIn picks the ingredients a step mentions.
function usedIn(step, ingredients) {
  const text = step.toLowerCase();
  return ingredients.filter((ing) => {
    const words = (ing.food || ing.line).toLowerCase().split(/[^a-z]+/).filter((w) => w.length > 3);
    return words.some((w) => text.includes(w.replace(/s$/, "")));
  });
}

export function startCooking(r, v) {
  const box = $("#cook-view");
  let at = 0;
  let spoken = -1; // the step last read aloud
  let stopListening = null;
  const say = () => { spoken = at; speak(`Step ${at + 1}. ${temps(r.steps[at].text, v.system)}`); };
  const startTimer = (t) => {
    const timer = { label: t.label, ends: Date.now() + t.secs * 1000, done: false, id: null };
    timers.push(timer);
    drawTimers();
    // The server buzzes the cook's phones too, even with the screen off.
    post("/api/push/timer", { label: t.label, seconds: Math.round(t.secs), recipe: r.title }).then((res) => (timer.id = res.id)).catch(() => {});
  };
  const command = (c) => {
    if (c === "next") move(1);
    if (c === "back") move(-1);
    if (c === "repeat") say();
    if (c === "timer") {
      const t = timersIn(r.steps[at].text)[0];
      if (t) { startTimer(t); speak(`Timer started: ${t.label}`); }
    }
  };
  const onVisible = () => { if (document.visibilityState === "visible" && !box.classList.contains("hidden")) keepAwake(); };
  const close = () => {
    box.classList.add("hidden");
    document.body.style.overflow = "";
    wake?.release?.();
    wake = null;
    clearInterval(tick);
    stopSpeaking();
    stopListening?.();
    stopListening = null;
    document.removeEventListener("keydown", onKey);
    document.removeEventListener("visibilitychange", onVisible);
  };
  const onKey = (e) => {
    if (e.key === "ArrowRight" || e.key === " ") move(1);
    if (e.key === "ArrowLeft") move(-1);
    if (e.key === "Escape") close();
  };
  const move = (d) => {
    at = Math.max(0, Math.min(r.steps.length - 1, at + d));
    draw();
  };
  const drawTimers = () => {
    const el = $("#cook-timers", box);
    if (!el) return;
    el.innerHTML = timers.map((t, i) => {
      const left = Math.max(0, (t.ends - Date.now()) / 1000);
      return `<span class="${left ? "chip-info" : "chip-no motion-safe:animate-pulse"} text-base">⏱ ${esc(t.label)}: ${left ? clock(left) : "Done!"}
        <button data-stop="${i}" class="ml-1" aria-label="Stop timer">✕</button></span>`;
    }).join(" ");
    $$("[data-stop]", el).forEach((b) => (b.onclick = () => {
      const [t] = timers.splice(Number(b.dataset.stop), 1);
      if (t?.id && !t.done) del(`/api/push/timer/${t.id}`).catch(() => {});
      drawTimers();
    }));
  };
  const draw = () => {
    const step = r.steps[at];
    const found = timersIn(step.text);
    const used = usedIn(step.text, r.ingredients);
    box.innerHTML = `
      <div class="mx-auto flex min-h-full max-w-3xl flex-col gap-4 p-4 pt-safe">
        <div class="flex items-center gap-2">
          <span class="text-sm text-slate-400">Step ${at + 1} of ${r.steps.length}${step.section ? ` · ${esc(step.section)}` : ""}</span>
          <span class="ml-auto flex flex-wrap justify-end gap-1">
            ${canSpeak() ? `<button id="cook-voice" class="btn-ghost px-2 ${voiceOn() ? "text-emerald-300" : ""}" aria-pressed="${voiceOn()}">${voiceOn() ? "🔊" : "🔈"} Read aloud</button>` : ""}
            ${canListen() ? `<button id="cook-listen" class="btn-ghost px-2 ${stopListening ? "text-emerald-300" : ""}" aria-pressed="${Boolean(stopListening)}">🎙️ ${stopListening ? "Listening" : "Listen"}</button>` : ""}
            <button id="cook-close" class="btn-ghost px-2">✕ Close</button></span>
        </div>
        ${stopListening ? `<p class="text-xs text-slate-400">Say "next", "back", "repeat" or "timer".</p>` : ""}
        <div class="h-1.5 w-full rounded-full bg-slate-800"><div id="cook-bar" class="h-full rounded-full bg-emerald-500"></div></div>
        <p class="cook-step min-h-[30vh] break-words font-medium">${esc(temps(step.text, v.system))}</p>
        ${found.length ? `<div class="flex flex-wrap gap-2">${found.map((t, i) => `<button data-timer="${i}" class="btn-secondary">⏱ Start ${esc(t.label)}</button>`).join("")}</div>` : ""}
        <div id="cook-timers" class="flex flex-wrap gap-2"></div>
        ${used.length ? `<div class="card"><h2 class="label">For this step</h2><ul class="space-y-1 text-lg">${used.map((ing) =>
          `<li>${esc(amountLine(ing, v.factor, v.system))}</li>`).join("")}</ul></div>` : ""}
        ${wake ? "" : `<p class="text-xs text-slate-500">Tip: this browser may let the screen go dark. Opening RecipeBank through a secure (https) address keeps it on.</p>`}
        <div class="sticky bottom-0 mt-auto grid grid-cols-2 gap-3 bg-slate-950 py-3 pb-safe">
          <button id="cook-prev" class="btn-secondary py-4 text-lg" ${at === 0 ? "disabled" : ""}>‹ Back</button>
          ${at < r.steps.length - 1 ? `<button id="cook-next" class="btn-primary py-4 text-lg">Next ›</button>`
            : `<button id="cook-done" class="btn-primary py-4 text-lg">✓ Done</button>`}
        </div>
      </div>`;
    $("#cook-bar", box).style.width = `${((at + 1) / r.steps.length) * 100}%`;
    $("#cook-close", box).onclick = close;
    $("#cook-prev", box).onclick = () => move(-1);
    const next = $("#cook-next", box);
    if (next) next.onclick = () => move(1);
    const done = $("#cook-done", box);
    if (done) done.onclick = close;
    $$("[data-timer]", box).forEach((b) => (b.onclick = () => startTimer(found[Number(b.dataset.timer)])));
    const voice = $("#cook-voice", box);
    if (voice) voice.onclick = () => { setVoice(!voiceOn()); if (voiceOn()) say(); draw(); };
    const ear = $("#cook-listen", box);
    if (ear) ear.onclick = () => {
      if (stopListening) { stopListening(); stopListening = null; }
      else stopListening = listen(command, (msg) => { stopListening = null; draw(); alert(msg); });
      draw();
    };
    if (voiceOn() && spoken !== at) say();
    drawTimers();
  };

  // Swipe left/right to change steps.
  let x0 = null;
  box.ontouchstart = (e) => { x0 = e.touches[0].clientX; };
  box.ontouchend = (e) => {
    if (x0 === null) return;
    const dx = e.changedTouches[0].clientX - x0;
    if (Math.abs(dx) > 70) move(dx < 0 ? 1 : -1);
    x0 = null;
  };

  box.classList.remove("hidden");
  document.body.style.overflow = "hidden";
  document.addEventListener("keydown", onKey);
  document.addEventListener("visibilitychange", onVisible);
  clearInterval(tick);
  tick = setInterval(() => {
    for (const t of timers) {
      if (!t.done && Date.now() >= t.ends) {
        t.done = true;
        beep();
        if (voiceOn()) speak(`Timer done: ${t.label}`);
      }
    }
    drawTimers();
  }, 1000);
  keepAwake().then(draw);
  draw();
}
