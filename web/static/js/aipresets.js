// Admin → AI: "Quick setup" fills in sensible settings for a service (from
// NovelCheck's presets). Every model listed can read photos too. Prices are
// dollars per million tokens and change: check the service's own page.
import { $, esc } from "./ui.js";

const PRESETS = {
  ollama: { label: "Ollama (your own computer, free)", provider: "openai", home: "http://your-server:11434/v1", text: "qwen2.5:3b",
    photos: "qwen2.5vl:7b", json: true, pin: "0", pout: "0",
    hint: "Runs on your own computer; no key needed. The Ollama card below finds it and downloads models." },
  lmstudio: { label: "LM Studio (your own computer, free)", provider: "openai", home: "http://your-pc:1234/v1", text: "", photos: "",
    json: false, pin: "0", pout: "0",
    hint: "In LM Studio, start the server (Developer → Start server) and let other devices on the network use it. Then type the model names it shows." },
  gemini: { label: "Google Gemini", provider: "openai", base: "https://generativelanguage.googleapis.com/v1beta/openai",
    text: "gemini-2.5-flash-lite", photos: "gemini-2.5-flash", json: true, pin: "0.30", pout: "2.50",
    hint: "Get a key at aistudio.google.com → Get API key. There's a free tier with daily limits; Google may use free-tier requests to improve its models. Check the current model names and prices there." },
  openai: { label: "OpenAI", provider: "openai", base: "https://api.openai.com/v1", text: "gpt-4o-mini", photos: "gpt-4o-mini",
    json: true, pin: "0.15", pout: "0.60", hint: "Get a key at platform.openai.com → API keys. Check current prices there." },
  claude: { label: "Anthropic Claude", provider: "anthropic", base: "", text: "claude-haiku-4-5", photos: "claude-haiku-4-5",
    json: false, pin: "1.00", pout: "5.00", hint: "Get a key at console.anthropic.com → API Keys. Check current prices there." },
};

// presetPicker is the menu; which is "llm" (the main AI) or "photo".
export function presetPicker(which) {
  return `<label class="block"><span class="label">Quick setup</span><select data-preset="${which}" class="input">
      <option value="">Pick a service to fill in its settings…</option>
      ${Object.entries(PRESETS).map(([k, p]) => `<option value="${k}">${esc(p.label)}</option>`).join("")}</select></label>
    <p data-preset-hint="${which}" class="hidden text-xs text-slate-400"></p>`;
}

// wirePresets fills the form's fields for the chosen service.
export function wirePresets(form, which) {
  const sel = $(`[data-preset="${which}"]`, form);
  sel.onchange = () => {
    const p = PRESETS[sel.value];
    const hint = $(`[data-preset-hint="${which}"]`, form);
    hint.classList.toggle("hidden", !p);
    if (!p) return;
    hint.textContent = p.hint;
    const set = (name, v) => {
      const el = form[name];
      if (!el || v === undefined) return;
      if (el.type === "checkbox") el.checked = v;
      else el.value = v;
      el.dispatchEvent(new Event("change"));
    };
    set(`${which}_provider`, p.provider);
    const url = form[`${which}_base_url`];
    if (p.base !== undefined) set(`${which}_base_url`, p.base);
    else if (url && !/^https?:\/\/(\d+\.|[a-z0-9-]+(:|\/))/i.test(url.value)) { url.value = ""; url.placeholder = p.home; url.focus(); }
    set(`${which}_json_mode`, p.json);
    set(`${which}_price_in`, p.pin);
    set(`${which}_price_out`, p.pout);
    if (which === "llm") {
      set("llm_model", p.text);
      set("llm_vision_model", p.photos === p.text ? "" : p.photos);
    } else {
      set("photo_model", p.photos);
    }
  };
}

// priceFields are the per-million-token prices, for the monthly cost.
export function priceFields(which, s) {
  const v = (k) => esc(s[`${which}_${k}`] || "0");
  return `<div class="grid grid-cols-2 gap-3">
    <label class="block"><span class="label">$ per million tokens in</span><input name="${which}_price_in" type="number" min="0" step="0.01" class="input" value="${v("price_in")}"></label>
    <label class="block"><span class="label">$ per million tokens out</span><input name="${which}_price_out" type="number" min="0" step="0.01" class="input" value="${v("price_out")}"></label></div>
    <p class="text-xs text-slate-500">0 for a model on your own computer. Used for "about $X this month".</p>`;
}
