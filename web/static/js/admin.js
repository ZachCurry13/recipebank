// Admin: the AI RecipeBank uses, house settings, and accounts.
import { get, post, put, del } from "./api.js";
import { $, $$, esc, attempt, busy, toast, sheet } from "./ui.js";
import { refreshInfo } from "./app.js";
import { renderRemote } from "./remoteaccess.js";
import { renderHints } from "./readinghints.js";
import { renderEmailAdmin } from "./email.js";
import { renderBackup } from "./backup.js";

const ROLES = [["admin", "Admin (everything)"], ["editor", "Parent (recipes and people)"], ["kid", "Kid (reads and cooks)"]];

export async function renderAdmin(view, params, state) {
  const [s, users, people] = await Promise.all([get("/api/admin/settings"), get("/api/admin/users"), get("/api/people")]);
  const opt = (v, cur, label) => `<option value="${v}" ${cur === v ? "selected" : ""}>${label}</option>`;
  view.innerHTML = `
    <h1 class="mb-4 text-2xl font-bold">⚙️ Admin</h1>
    <div class="grid gap-4 lg:grid-cols-2">
      <form id="ai" class="card space-y-3">
        <h2 class="font-semibold">AI</h2>
        <p class="text-sm text-slate-400">The AI reads photos of recipe cards, pages without a standard recipe, and pasted text.
          Allergy checks never depend on it. It can be a cloud service or a model on your own network (Ollama).</p>
        <label class="block"><span class="label">Kind</span><select name="llm_provider" class="input">
          ${opt("openai", s.llm_provider, "OpenAI-compatible (OpenAI, Gemini, Ollama, LM Studio…)")}${opt("anthropic", s.llm_provider, "Anthropic (Claude)")}</select></label>
        <label class="block" data-openai><span class="label">Address (base URL)</span>
          <input name="llm_base_url" class="input" value="${esc(s.llm_base_url)}" placeholder="http://your-server:11434/v1"></label>
        <label class="block"><span class="label">API key ${s.llm_api_key_set ? "(saved; leave blank to keep it)" : "(not needed for Ollama)"}</span>
          <input name="llm_api_key" type="password" autocomplete="off" class="input"></label>
        <label class="block"><span class="label">Model</span><input name="llm_model" class="input" value="${esc(s.llm_model)}" placeholder="gpt-4o-mini, qwen2.5:7b, claude-haiku-4-5"></label>
        <label class="block"><span class="label">More models to try if that one fails (comma between them)</span>
          <input name="llm_fallback_model" class="input" value="${esc(s.llm_fallback_model)}"></label>
        <label class="block"><span class="label">Model for photos (blank = the model above)</span>
          <input name="llm_vision_model" class="input" value="${esc(s.llm_vision_model)}" placeholder="qwen2.5vl:7b, llama3.2-vision"></label>
        <details class="text-sm text-slate-400"><summary class="cursor-pointer">More</summary>
          <label class="toggle mt-2"><input type="checkbox" name="llm_json_mode" ${s.llm_json_mode === "true" ? "checked" : ""}> Ask for JSON answers (turn off if the server rejects it)</label>
          <label class="mt-2 block"><span class="label">Wait for an answer (seconds, 0 = automatic)</span>
            <input name="llm_timeout_seconds" type="number" min="0" class="input" value="${esc(s.llm_timeout_seconds)}"></label></details>
        <p class="text-xs text-slate-500">Used this month: ${Number(s.tokens_this_month || 0).toLocaleString()} tokens.</p>
        <div class="flex flex-wrap gap-2"><button class="btn-primary">Save</button><button type="button" id="test" class="btn-secondary">Test the AI</button></div>
      </form>
      <form id="house" class="card space-y-3">
        <h2 class="font-semibold">House settings</h2>
        <label class="block"><span class="label">Allergen list</span><select name="allergen_list" class="input">
          ${opt("us", s.allergen_list, "US: the 9 major allergens")}${opt("eu", s.allergen_list, "EU: 14 (adds gluten, celery, mustard, lupin, molluscs, sulphites)")}</select></label>
        <label class="block"><span class="label">Units</span><select name="default_units" class="input">
          ${opt("us", s.default_units, "US (cups, °F)")}${opt("metric", s.default_units, "Metric (grams, ml, °C)")}</select></label>
        <div class="grid grid-cols-2 gap-3">
          <label class="block"><span class="label">"Use soon" reminder</span><select name="morning_hour" class="input">${hours(s.morning_hour)}</select></label>
          <label class="block"><span class="label">Tonight's dinner reminder</span><select name="tonight_hour" class="input">${hours(s.tonight_hour)}</select></label>
        </div>
        <label class="block"><span class="label">Stay signed in for (days)</span>
          <input name="session_days" type="number" min="1" max="365" class="input" value="${esc(s.session_days)}"></label>
        <div class="grid grid-cols-2 gap-3">
          <label class="block"><span class="label">Currency</span><select name="currency" class="input">${[["USD", "$ US dollar"], ["CAD", "$ Canadian dollar"],
            ["AUD", "$ Australian dollar"], ["EUR", "€ Euro"], ["GBP", "£ Pound"], ["NZD", "$ NZ dollar"]].map(([k, l]) => opt(k, s.currency, l)).join("")}</select></label>
          <label class="block"><span class="label">Weekly food budget</span><input name="budget_weekly" type="number" min="0" step="1" inputmode="decimal"
            class="input" value="${Number(s.budget_weekly) || ""}" placeholder="none"></label></div>
        <label class="block"><span class="label">USDA FoodData Central key, for nutrition estimates ${s.usda_api_key_set ? "(saved; leave blank to keep it)" : ""}</span>
          <input name="usda_api_key" type="password" autocomplete="off" class="input"></label>
        <p class="text-xs text-slate-500">Free: <a href="https://fdc.nal.usda.gov/api-key-signup" target="_blank" rel="noopener noreferrer" class="underline">sign up for a key</a>
          (1,000 lookups an hour), or type DEMO_KEY to try it (about 25 foods a day). Only food names are sent, and each food is looked up once.</p>
        <label class="toggle"><input type="checkbox" name="check_updates" ${s.check_updates === "true" ? "checked" : ""}>
          Tell me when a new version is out (asks GitHub every few hours)</label>
        <button class="btn-primary">Save</button>
        ${folders(s.folders)}
      </form>
    </div>
    <div id="hints" class="mt-4"></div>
    <div id="mail" class="mt-4"></div>
    <div id="backup" class="mt-4"></div>
    <div id="remote" class="mt-4"></div>
    <div class="card mt-4 space-y-3">
      <div class="flex flex-wrap items-center gap-2"><h2 class="mr-auto font-semibold">Accounts</h2><button id="new-user" class="btn-primary">➕ Add an account</button></div>
      <ul class="divide-y divide-slate-800">${users.map((u) => `
        <li class="flex flex-wrap items-center gap-2 py-2">
          <span class="min-w-0 basis-40 flex-1 break-words font-medium">${esc(u.username)}${u.id === state.user.id ? " (you)" : ""}</span>
          <select data-role="${u.id}" class="input w-auto">${ROLES.map(([v, l]) => opt(v, u.role, l)).join("")}</select>
          <select data-person="${u.id}" class="input w-auto" title="Their eating profile"><option value="0">No profile</option>${people.map((p) =>
            opt(p.id, u.person_id, esc(p.name))).join("")}</select>
          <button data-pw="${u.id}" class="btn-ghost">🔑 Password</button>
          ${u.id === state.user.id ? "" : `<button data-rm="${u.id}" class="btn-ghost text-rose-300">Remove</button>`}
        </li>`).join("")}</ul>
    </div>`;

  renderRemote($("#remote", view)).catch(() => {});
  renderHints($("#hints", view)).catch(() => {});
  renderEmailAdmin($("#mail", view), s);
  renderBackup($("#backup", view));
  const ai = $("#ai", view);
  const showURL = () => $("[data-openai]", ai).classList.toggle("hidden", ai.llm_provider.value === "anthropic");
  ai.llm_provider.onchange = showURL;
  showURL();
  ai.onsubmit = (e) => {
    e.preventDefault();
    const body = Object.fromEntries(["llm_provider", "llm_base_url", "llm_api_key", "llm_model", "llm_fallback_model",
      "llm_vision_model", "llm_timeout_seconds"].map((k) => [k, ai[k].value]));
    body.llm_json_mode = ai.llm_json_mode.checked ? "true" : "false";
    attempt(async () => { await put("/api/admin/settings", body); await refreshInfo(); ai.llm_api_key.value = ""; }, "Saved");
  };
  $("#test", view).onclick = (e) => attempt(() => busy(e.currentTarget, "Asking…", async () => {
    const r = await post("/api/admin/ai/test");
    toast(`The AI answered in ${r.seconds.toFixed(1)} seconds.`);
  }));
  const house = $("#house", view);
  house.onsubmit = (e) => {
    e.preventDefault();
    attempt(async () => {
      await put("/api/admin/settings", { allergen_list: house.allergen_list.value, default_units: house.default_units.value, session_days: house.session_days.value,
        morning_hour: house.morning_hour.value, tonight_hour: house.tonight_hour.value,
        check_updates: house.check_updates.checked ? "true" : "false", usda_api_key: house.usda_api_key.value.trim(),
        currency: house.currency.value, budget_weekly: String(Number(house.budget_weekly.value) || 0) });
      house.usda_api_key.value = "";
      await refreshInfo();
    }, "Saved");
  };
  const reload = () => renderAdmin(view, params, state);
  const save = (id) => attempt(() => put(`/api/admin/users/${id}`, {
    role: $(`[data-role="${id}"]`, view).value, person_id: Number($(`[data-person="${id}"]`, view).value) }), "Saved");
  $$("[data-role],[data-person]", view).forEach((sel) => (sel.onchange = () => save(sel.dataset.role || sel.dataset.person)));
  $$("[data-pw]", view).forEach((b) => (b.onclick = () => {
    const pw = prompt("New password (at least 8 characters):");
    if (pw) attempt(() => put(`/api/admin/users/${b.dataset.pw}/password`, { password: pw }), "Password changed");
  }));
  $$("[data-rm]", view).forEach((b) => (b.onclick = () => {
    if (confirm("Remove this account? Their recipes stay.")) attempt(async () => { await del(`/api/admin/users/${b.dataset.rm}`); reload(); });
  }));
  $("#new-user", view).onclick = () => {
    const d = sheet(`<form id="nu" class="space-y-3">
      <div class="flex items-center"><h2 class="text-lg font-semibold">Add an account</h2><button type="button" data-close class="btn-ghost ml-auto">✕</button></div>
      <label class="block"><span class="label">Username</span><input name="username" required minlength="2" maxlength="40" autocomplete="off" class="input"></label>
      <label class="block"><span class="label">Password (at least 8 characters)</span><input name="password" type="password" required minlength="8" autocomplete="new-password" class="input"></label>
      <label class="block"><span class="label">Can do</span><select name="role" class="input">${ROLES.map(([v, l]) => opt(v, "editor", l)).join("")}</select></label>
      <label class="block"><span class="label">Their eating profile</span><select name="person" class="input"><option value="0">None</option>${people.map((p) =>
        `<option value="${p.id}">${esc(p.name)}</option>`).join("")}</select></label>
      <button class="btn-primary">Add</button></form>`);
    $("#nu", d).onsubmit = (e) => {
      e.preventDefault();
      const f = e.target;
      const person = Number(f.person.value);
      attempt(async () => {
        await post("/api/admin/users", { username: f.username.value.trim(), password: f.password.value, role: f.role.value, person_id: person || null });
        d.close();
        reload();
      }, "Account added");
    };
  };
}

// folders shows where the database, photos and previews are kept.
function folders(f) {
  if (!f) return "";
  const row = (label, path, own, help) => `<li class="min-w-0"><b>${label}:</b> <code class="break-all text-xs">${esc(path)}</code>
    ${own === undefined ? "" : own ? `<span class="chip-ok">own dataset</span>` : `<span class="chip-info">inside the data folder</span>`}
    <div class="text-xs text-slate-500">${help}</div></li>`;
  return `<div class="border-t border-slate-800 pt-3"><h3 class="mb-1 text-sm font-semibold">Where files are kept</h3>
    <ul class="space-y-2 text-sm text-slate-300">
      ${row("Database", f.data, undefined, "Small and important: keep it on a fast pool with snapshots.")}
      ${row("Photos", f.photos, f.photos_own, "Dish photos and card scans: back these up.")}
      ${row("Previews", f.cache, f.cache_own, "Smaller copies of photos; safe to delete, made again when needed.")}
    </ul></div>`;
}

// hours are the choices for a daily reminder (the server's time zone).
function hours(current) {
  const opts = [["-1", "Off"]];
  for (let h = 5; h <= 21; h++) opts.push([String(h), new Date(2000, 0, 1, h).toLocaleTimeString(undefined, { hour: "numeric" })]);
  return opts.map(([v, l]) => `<option value="${v}" ${String(current) === v ? "selected" : ""}>${l}</option>`).join("");
}
