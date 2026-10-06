// Cook mode's voice: steps read aloud (the browser's own speech), and
// "next", "back", "repeat", "timer" heard where the browser can listen
// (Chrome and Safari, on the secure https address).
const KEY = "rb:voice";

export const canSpeak = () => "speechSynthesis" in window;
const Recognition = window.SpeechRecognition || window.webkitSpeechRecognition;
export const canListen = () => Boolean(Recognition) && window.isSecureContext;

export function voiceOn() {
  try { return localStorage.getItem(KEY) === "1"; } catch { return false; }
}

export function setVoice(on) {
  try { localStorage.setItem(KEY, on ? "1" : "0"); } catch { /* private mode */ }
  if (!on) stopSpeaking();
}

export function speak(text) {
  if (!canSpeak()) return;
  speechSynthesis.cancel();
  const u = new SpeechSynthesisUtterance(text.replace(/°F/g, " degrees Fahrenheit").replace(/°C/g, " degrees Celsius"));
  u.rate = 0.95;
  speechSynthesis.speak(u);
}

export function stopSpeaking() {
  if (canSpeak()) speechSynthesis.cancel();
}

const COMMANDS = [
  ["next", /\b(next|forward|continue|done)\b/],
  ["back", /\b(back|previous|go back)\b/],
  ["repeat", /\b(repeat|again|say that again|what)\b/],
  ["timer", /\b(timer|start timer|set timer)\b/],
];

// listen hears commands until stop() is called, calling onCommand("next" | "back" | "repeat" | "timer").
export function listen(onCommand, onError) {
  if (!canListen()) return () => {};
  let active = true;
  const rec = new Recognition();
  rec.continuous = true;
  rec.interimResults = false;
  rec.lang = navigator.language || "en-US";
  rec.onresult = (e) => {
    const said = e.results[e.results.length - 1][0].transcript.toLowerCase();
    const hit = COMMANDS.find(([, re]) => re.test(said));
    if (hit) onCommand(hit[0]);
  };
  rec.onerror = (e) => {
    if (e.error === "not-allowed" || e.error === "service-not-allowed") {
      active = false;
      onError?.("The microphone was blocked. Allow it for RecipeBank in the browser's settings.");
    }
  };
  // Browsers stop listening after a silence; start again while cook mode is open.
  rec.onend = () => { if (active) try { rec.start(); } catch { /* already started */ } };
  try { rec.start(); } catch { /* already started */ }
  return () => { active = false; try { rec.stop(); } catch { /* not started */ } };
}
