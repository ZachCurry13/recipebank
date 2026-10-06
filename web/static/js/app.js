// RecipeBank bootstrap: session check, hash router, navigation.
import { api, get, post, setUnauthorizedHandler, setOffline } from "./api.js";
import { $, esc, canManage } from "./ui.js";
import { buildNav, markNav } from "./nav.js";
import { renderStock } from "./stock.js";
import { renderShopping, flush } from "./shopping.js";
import { renderTonight } from "./tonight.js";
import { renderPlan } from "./plan.js";
import { renderCollections, renderCollection, renderSeason } from "./collections.js";
import { applyAppearance } from "./appearance.js";
import { renderLibrary } from "./library.js";
import { renderRecipe } from "./recipe.js";
import { renderAdd } from "./add.js";
import { renderEdit } from "./editor.js";
import { renderFamily } from "./family.js";
import { renderAdmin } from "./admin.js";
import { renderProfile } from "./profile.js";
import { renderMake } from "./make.js";
import { renderWhatsNew } from "./whatsnew.js";
import { renderCookbook } from "./cookbook.js";
import { renderEvents } from "./events.js";
import { renderEvent } from "./event.js";
import { firstRun } from "./firstrun.js";
import { checkForUpdates, watchServerVersion } from "./updatebanner.js";

export const state = { user: null, info: null };

const routes = {
  kitchen: (v, p, s) => renderLibrary(v, "kitchen", p, s),
  make: renderMake,
  home: (v, p, s) => renderLibrary(v, "home", p, s),
  pantry: (v, p, s) => renderStock(v, "kitchen", p, s),
  supplies: (v, p, s) => renderStock(v, "home", p, s),
  shopping: renderShopping,
  tonight: renderTonight,
  plan: renderPlan,
  collections: renderCollections,
  collection: renderCollection,
  season: renderSeason,
  recipe: renderRecipe,
  add: renderAdd,
  edit: renderEdit,
  family: renderFamily,
  admin: renderAdmin,
  profile: renderProfile,
  whatsnew: renderWhatsNew,
  cookbook: renderCookbook,
  events: renderEvents,
  event: renderEvent,
};
const NAV_OF = { recipe: null, edit: "add", collection: "collections", season: "collections", whatsnew: "profile", cookbook: "collections", event: "events" };

function showOnly(id) {
  for (const v of ["#setup-view", "#login-view", "#app-view"]) $(v).classList.toggle("hidden", v !== id);
}

export async function refreshInfo() {
  state.info = await get("/api/info");
  return state.info;
}

async function showApp() {
  showOnly("#app-view");
  document.body.dataset.role = state.user.role;
  applyAppearance(state.user);
  await refreshInfo().catch(() => {});
  buildNav(state.user);
  const v = state.user.version || "dev";
  $("#version-label").textContent = `Version ${v}`;
  $("#header-version").textContent = v;
  route();
  firstRun(state).catch(() => {});
  if (canManage(state.user)) checkForUpdates();
  watchServerVersion(v);
}

let leaving = null; // the open page's clean-up (timers, wake lock)

async function route() {
  if (!state.user) return;
  const [path, query = ""] = location.hash.replace(/^#\/?/, "").split("?");
  const [name, id] = path.split("/");
  const page = routes[name] ? name : "tonight";
  const params = Object.fromEntries(new URLSearchParams(query));
  if (id) params.id = id;
  const lit = page in NAV_OF ? NAV_OF[page] : page;
  markNav(lit);
  if (leaving) {
    leaving();
    leaving = null;
  }
  const view = $("#view");
  view.innerHTML = `<p class="text-slate-500">Loading…</p>`;
  try {
    leaving = (await routes[page](view, params, state)) || null;
  } catch (e) {
    view.innerHTML = `<div class="box-danger">${esc(e.message || e)}</div>`;
  }
  window.scrollTo(0, 0);
}

export function go(hash) {
  if (location.hash === hash) route();
  else location.hash = hash;
}

export async function signOut() {
  await post("/api/auth/logout").catch(() => {});
  try { localStorage.removeItem(ME); localStorage.removeItem("rb:shopping"); } catch { /* private mode */ }
  location.hash = "";
  location.reload();
}

const ME = "rb:me";
function remember(user) {
  try { localStorage.setItem(ME, JSON.stringify(user)); } catch { /* private mode */ }
}
function recall() {
  try { return JSON.parse(localStorage.getItem(ME)); } catch { return null; }
}

async function boot() {
  setUnauthorizedHandler(() => {
    state.user = null;
    showOnly("#login-view");
  });
  $("#login-form").onsubmit = async (e) => {
    e.preventDefault();
    const f = e.target;
    const err = $("#login-error");
    err.classList.add("hidden");
    try {
      state.user = await post("/api/auth/login", { username: f.username.value, password: f.password.value, remember: f.remember.checked });
      remember(state.user);
      await showApp();
    } catch (ex) {
      err.textContent = ex.message;
      err.classList.remove("hidden");
    }
  };
  $("#setup-form").onsubmit = async (e) => {
    e.preventDefault();
    const f = e.target;
    const err = $("#setup-error");
    err.classList.add("hidden");
    if (f.password.value !== f.confirm.value) {
      err.textContent = "The two passwords don't match.";
      err.classList.remove("hidden");
      return;
    }
    try {
      state.user = await post("/api/setup", { username: f.username.value.trim(), password: f.password.value });
      location.hash = "#/family";
      await showApp();
    } catch (ex) {
      err.textContent = ex.message;
      err.classList.remove("hidden");
    }
  };
  window.addEventListener("hashchange", route);
  try {
    state.user = await api("/api/me", { retries: 1 });
    remember(state.user);
    flush().catch(() => {}); // shopping-list ticks made without signal
    await showApp();
  } catch (e) {
    // No signal (in the store): open as the last person signed in on this
    // phone; the shopping list works from its saved copy.
    const cached = e.status === 0 && recall();
    if (cached) {
      setOffline(true);
      state.user = cached;
      if (!location.hash) location.hash = "#/shopping";
      await showApp();
    } else {
      const setup = await get("/api/setup").catch(() => ({ needed: false }));
      showOnly(setup.needed ? "#setup-view" : "#login-view");
    }
  }
  if ("serviceWorker" in navigator) navigator.serviceWorker.register("/sw.js").catch(() => {});
}

boot();
