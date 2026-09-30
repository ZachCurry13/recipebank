// RecipeBank bootstrap: session check, hash router, navigation.
import { get, post, setUnauthorizedHandler } from "./api.js";
import { $, $$, esc, canManage } from "./ui.js";
import { applyAppearance } from "./appearance.js";
import { renderLibrary } from "./library.js";
import { renderRecipe } from "./recipe.js";
import { renderAdd } from "./add.js";
import { renderEdit } from "./editor.js";
import { renderFamily } from "./family.js";
import { renderAdmin } from "./admin.js";
import { renderProfile } from "./profile.js";

export const state = { user: null, info: null };

// Pages: [route, label, icon, who may see it].
const NAV = [
  ["kitchen", "Kitchen", "🍲", () => true],
  ["home", "Home & Care", "🧴", () => true],
  ["add", "Add", "➕", canManage],
  ["family", "Family", "👪", () => true],
  ["admin", "Admin", "⚙️", (u) => u.role === "admin"],
  ["profile", "Me", "🙂", () => true],
];

const routes = {
  kitchen: (v, p) => renderLibrary(v, "kitchen", p),
  home: (v, p) => renderLibrary(v, "home", p),
  recipe: renderRecipe,
  add: renderAdd,
  edit: renderEdit,
  family: renderFamily,
  admin: renderAdmin,
  profile: renderProfile,
};
const NAV_OF = { recipe: null, edit: "add" };

function showOnly(id) {
  for (const v of ["#setup-view", "#login-view", "#app-view"]) $(v).classList.toggle("hidden", v !== id);
}

function buildNav() {
  const items = NAV.filter(([, , , may]) => may(state.user));
  $("#nav").innerHTML = items.map(([r, label, ico]) => `<a href="#/${r}" data-route="${r}" class="nav-link">${ico} ${esc(label)}</a>`).join("");
  $("#mobile-nav").innerHTML = items.map(([r, label, ico]) =>
    `<a href="#/${r}" data-route="${r}"><span class="ico">${ico}</span><span>${esc(label)}</span></a>`).join("");
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
  buildNav();
  const v = state.user.version || "dev";
  $("#version-label").textContent = `Version ${v}`;
  $("#header-version").textContent = v;
  route();
}

let leaving = null; // the open page's clean-up (timers, wake lock)

async function route() {
  if (!state.user) return;
  const [path, query = ""] = location.hash.replace(/^#\/?/, "").split("?");
  const [name, id] = path.split("/");
  const page = routes[name] ? name : "kitchen";
  const params = Object.fromEntries(new URLSearchParams(query));
  if (id) params.id = id;
  const lit = page in NAV_OF ? NAV_OF[page] : page;
  $$("[data-route]").forEach((a) => a.classList.toggle("active", a.dataset.route === lit));
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
  location.hash = "";
  location.reload();
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
    state.user = await get("/api/me");
    await showApp();
  } catch {
    const setup = await get("/api/setup").catch(() => ({ needed: false }));
    showOnly(setup.needed ? "#setup-view" : "#login-view");
  }
  if ("serviceWorker" in navigator) navigator.serviceWorker.register("/sw.js").catch(() => {});
}

boot();
