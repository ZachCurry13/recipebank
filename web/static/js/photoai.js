// Admin → AI: an optional second AI just for photos (recipe cards,
// receipts, the fridge, cookbook indexes), such as a bigger model on a
// computer with a stronger graphics card, or an online service.
import { $, esc } from "./ui.js";
import { presetPicker, priceFields } from "./aipresets.js";

export function photoAI(s, opt) {
  return `<details class="rounded-lg border border-slate-800 p-3" ${s.photo_provider ? "open" : ""}>
    <summary class="cursor-pointer font-semibold">📷 A second AI for photos (optional)</summary>
    <div class="mt-2 space-y-3">
      <p class="text-sm text-slate-400">Photos (handwritten cards, receipts, the fridge, cookbook indexes) go to this AI first, for
        example a bigger model on a computer with a stronger graphics card, or an online service. Everything else stays with
        the AI above. When this one is off or can't be reached, the AI above reads the photos.
        With an online service, the photos are sent to that company.</p>
      ${presetPicker("photo")}
      <label class="block"><span class="label">Kind</span><select name="photo_provider" class="input">
        ${opt("", s.photo_provider || "", "None: the AI above reads photos")}
        ${opt("openai", s.photo_provider, "OpenAI-compatible (Ollama, LM Studio, Gemini, OpenAI…)")}${opt("anthropic", s.photo_provider, "Anthropic (Claude)")}</select></label>
      <div data-photo-fields class="space-y-3">
        <label class="block" data-photo-url><span class="label">Address (base URL)</span>
          <input name="photo_base_url" class="input" value="${esc(s.photo_base_url || "")}" placeholder="http://your-pc:11434/v1"></label>
        <label class="block"><span class="label">API key ${s.photo_api_key_set ? "(saved; leave blank to keep it)" : "(not needed for Ollama)"}</span>
          <input name="photo_api_key" type="password" autocomplete="off" class="input"></label>
        <label class="block"><span class="label">Model for photos (more to try after it: comma between them)</span>
          <input name="photo_model" class="input" value="${esc(s.photo_model || "")}" placeholder="qwen2.5vl:32b, gemini-2.5-flash"></label>
        ${priceFields("photo", s)}
        <label class="toggle"><input type="checkbox" name="photo_json_mode" ${s.photo_json_mode === "false" ? "" : "checked"}> Ask for JSON answers (turn off if the server rejects it)</label>
      </div></div></details>`;
}

// wirePhotoAI hides what doesn't apply to the chosen kind.
export function wirePhotoAI(form) {
  const show = () => {
    const kind = form.photo_provider.value;
    $("[data-photo-fields]", form).classList.toggle("hidden", !kind);
    $("[data-photo-url]", form).classList.toggle("hidden", kind === "anthropic");
  };
  form.photo_provider.onchange = show;
  show();
}

// photoSettings is what Save sends for the photo AI.
export function photoSettings(form) {
  return { photo_provider: form.photo_provider.value, photo_base_url: form.photo_base_url.value, photo_api_key: form.photo_api_key.value,
    photo_model: form.photo_model.value, photo_json_mode: form.photo_json_mode.checked ? "true" : "false",
    photo_price_in: form.photo_price_in.value || "0", photo_price_out: form.photo_price_out.value || "0" };
}
