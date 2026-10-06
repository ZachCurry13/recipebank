// Me: units, theme, password, sign out.
import { put } from "./api.js";
import { $, esc, attempt } from "./ui.js";
import { applyAppearance } from "./appearance.js";
import { signOut } from "./app.js";
import { renderPushCard } from "./push.js";

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
    <button id="out" class="btn-secondary mt-4">Sign out</button>`;
  renderPushCard($("#push-card", view), u).catch(() => {});
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
