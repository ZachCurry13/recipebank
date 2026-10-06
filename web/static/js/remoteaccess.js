// Admin → Remote access: the built-in Cloudflare Tunnel, so RecipeBank (and
// the shopping list) works away from home, on a secure https address.
import { get, put } from "./api.js";
import { $, esc, attempt, busy } from "./ui.js";

const STATES = {
  off: ["chip-info", "Off"],
  starting: ["chip-unsure", "Connecting…"],
  connected: ["chip-ok", "Connected"],
  retrying: ["chip-no", "Trying again"],
  not_installed: ["chip-no", "Not available in this install"],
};
const GUIDE = "https://github.com/ZachCurry13/recipebank/blob/main/docs/REMOTE_ACCESS.md";

export async function renderRemote(box) {
  const t = await get("/api/admin/tunnel");
  const [cls, label] = STATES[t.status.state] || STATES.off;
  box.innerHTML = `<form id="rf" class="card space-y-3">
    <div class="flex flex-wrap items-center gap-2"><h2 class="mr-auto font-semibold">🌍 Remote access</h2><span class="${cls}">${label}</span></div>
    <p class="text-sm text-slate-400">Use RecipeBank away from home (the shopping list in the store) through a free Cloudflare Tunnel.
      It also gives the secure https address phones need for live barcode scanning, voice in cook mode and notifications.
      <a href="${GUIDE}" target="_blank" rel="noopener noreferrer" class="underline">Step-by-step guide</a></p>
    <label class="block"><span class="label">Tunnel token ${t.has_token ? "(saved; leave blank to keep it)" : ""}</span>
      <input name="token" type="password" autocomplete="off" class="input" placeholder="eyJ… (or paste the whole command from Cloudflare)"></label>
    <label class="block"><span class="label">Public address</span>
      <input name="hostname" class="input" value="${esc(t.hostname)}" placeholder="recipes.example.com"></label>
    <label class="toggle"><input type="checkbox" name="enabled" ${t.enabled ? "checked" : ""}> Turn on remote access</label>
    ${t.status.last_error ? `<p class="box-caution">${esc(t.status.last_error)}</p>` : ""}
    ${t.hostname && t.status.state === "connected" ? `<p class="text-sm">Open <a class="underline" href="https://${esc(t.hostname)}" target="_blank" rel="noopener noreferrer">https://${esc(t.hostname)}</a> on your phone, sign in, and add it to the home screen.</p>` : ""}
    <div class="flex flex-wrap gap-2"><button class="btn-primary">Save &amp; connect</button><button type="button" id="rf-refresh" class="btn-ghost">Refresh</button></div>
    ${t.status.log?.length ? `<details class="text-xs text-slate-500"><summary class="cursor-pointer">Connector log</summary>
      <pre class="mt-1 max-h-48 overflow-auto whitespace-pre-wrap break-all">${esc(t.status.log.join("\n"))}</pre></details>` : ""}
  </form>`;
  const f = $("#rf", box);
  f.onsubmit = (e) => {
    e.preventDefault();
    attempt(() => busy(f.querySelector(".btn-primary"), "Connecting…", async () => {
      await put("/api/admin/tunnel", { enabled: f.enabled.checked, token: f.token.value, hostname: f.hostname.value });
      await new Promise((r) => setTimeout(r, 2500));
      await renderRemote(box);
    }));
  };
  $("#rf-refresh", box).onclick = () => attempt(() => renderRemote(box));
}
