// Me: units, theme, password, sign out.
import { put } from "./api.js";
import { $, esc, attempt } from "./ui.js";
import { applyAppearance } from "./appearance.js";
import { signOut } from "./app.js";
import { renderPushCard } from "./push.js";
import { welcome } from "./firstrun.js";

export function renderProfile(view, params, state) {
  const u = state.user;
  const opt = (v, cur, l) => `<option value="${v}" ${cur === v ? "selected" : ""}>${l}</option>`;
  const house = state.info?.default_units === "metric" ? "metric" : "US";
  view.innerHTML = `
    <h1 class="mb-4 text-2xl font-bold">🙂 ${esc(u.username)}</h1>
    <div class="grid gap-4 lg:grid-cols-2">
      <form id="prefs" class="card space-y-3">
        <h2 class="font-semibold">How recipes look</h2>
        <label class="block"><span class="label">Units</span><select name="units" class="input">
          ${opt("", u.units, `The house setting (${house})`)}${opt("us", u.units, "US (cups, °F)")}${opt("metric", u.units, "Metric (grams, ml, °C)")}</select></label>
        <label class="block"><span class="label">Theme</span><select name="theme" class="input">
          ${opt("", u.theme, "Match this device")}${opt("dark", u.theme, "Dark")}${opt("light", u.theme, "Light")}</select></label>
        <button class="btn-primary">Save</button>
      </form>
      <form id="pw" class="card space-y-3">
        <h2 class="font-semibold">Change password</h2>
        <input name="current" type="password" autocomplete="current-password" required placeholder="Current password" class="input">
        <input name="next" type="password" autocomplete="new-password" required minlength="8" placeholder="New password (at least 8 characters)" class="input">
        <button class="btn-primary">Change password</button>
      </form>
    </div>
    <div id="push-card" class="card mt-4 space-y-3"></div>
    <div class="card mt-4 space-y-3"><h2 class="text-lg font-semibold">❓ Help</h2>
      <div class="flex flex-wrap gap-2"><button type="button" id="welcome" class="btn-secondary">👋 The welcome tour</button>
        <a href="#/whatsnew" class="btn-secondary">🆕 What's new</a>
        ${u.role === "admin" ? `<button type="button" id="setup-guide" class="btn-secondary">🧭 Getting started</button>` : ""}</div></div>
    <button id="out" class="btn-secondary mt-4">Sign out</button>`;
  renderPushCard($("#push-card", view), u).catch(() => {});
  $("#welcome", view).onclick = () => welcome(state);
  const guide = $("#setup-guide", view);
  if (guide) guide.onclick = () => attempt(async () => { await put("/api/admin/guide", { step: "hidden", done: false }); location.hash = "#/tonight"; });
  $("#prefs", view).onsubmit = (e) => {
    e.preventDefault();
    const f = e.target;
    attempt(async () => {
      await put("/api/me/prefs", { units: f.units.value, theme: f.theme.value });
      u.units = f.units.value;
      u.theme = f.theme.value;
      applyAppearance(u);
    }, "Saved");
  };
  $("#pw", view).onsubmit = (e) => {
    e.preventDefault();
    const f = e.target;
    attempt(async () => {
      await put("/api/me/password", { current_password: f.current.value, new_password: f.next.value });
      f.reset();
    }, "Password changed");
  };
  $("#out", view).onclick = signOut;
}
