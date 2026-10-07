// Navigation, in each person's own order (Me → Keep it simple). On a computer:
// the first pages in the top bar (more of them on wider screens) and a More
// menu with the rest, so the bar never scrolls. On a phone: a bottom bar with
// the first five and a More sheet.
import { $, $$, esc, canManage, sheet, hiddenPages } from "./ui.js";

const all = () => true;

// Pages a person can order, in the usual order: [route, label, icon, who may
// see it, a shorter label for the phone's bar].
export const NAV = [
  ["tonight", "Tonight", "🌙", all],
  ["kitchen", "Recipes", "🍲", all],
  ["add", "Add a recipe", "➕", canManage, "Add"],
  ["plan", "Plan", "📅", all],
  ["shopping", "Shopping", "🛒", all],
  ["pantry", "Pantry", "🥫", all],
  ["make", "What can I make?", "🥕", all, "Make"],
  ["collections", "Collections", "📚", all],
  ["books", "Bookshelf", "📖", all],
  ["events", "Events", "🎉", all],
];
// LAST are always at the end of More.
const LAST = [["admin", "Admin", "⚙️", (u) => u.role === "admin"], ["profile", "Me", "🙂", all]];

// ALWAYS can't be hidden from a person's menu.
export const ALWAYS = ["tonight", "kitchen"];

// PAGE_FEATURES: the house features (Admin → Features) a page needs.
export const PAGE_FEATURES = { plan: ["plan"], shopping: ["shopping"], make: ["make"], home: ["home"], pantry: ["pantry"],
  supplies: ["home", "pantry"], collections: ["collections"], collection: ["collections"], season: ["collections"],
  books: ["books"], book: ["books"], events: ["events"], event: ["events"] };

// menuOrder is a person's order of the pages: theirs, then any they haven't placed.
export function menuOrder(user) {
  const mine = (user?.menu_order || "").split(",").filter((r) => NAV.some((n) => n[0] === r));
  return [...new Set([...mine, ...NAV.map((n) => n[0])])].map((r) => NAV.find((n) => n[0] === r));
}

// The top bar holds 4 pages, 6 on wider screens, 7 on the widest; the rest are in More.
const PHONE_BAR = 5;
const tier = (i) => (i < 4 ? "md" : i < 6 ? "lg" : i < 7 ? "xl" : "");
const ON_BAR = { md: "", lg: "hidden lg:block", xl: "hidden xl:block" };
const IN_MORE = { md: "hidden", lg: "lg:hidden", xl: "xl:hidden", "": "" };

let listening = false; // the page-wide listeners that close the More menu, added once

function showMenu(open) {
  const menu = $("#nav-menu");
  if (!menu) return;
  menu.classList.toggle("hidden", !open);
  $("#nav-more").setAttribute("aria-expanded", String(open));
}

// buildNav shows the pages this person may see, in their order, minus what the
// house turned off and what they hid from their own menu.
export function buildNav(user, info) {
  const hidden = hiddenPages(user);
  const shown = ([r, , , may]) => may(user) && (ALWAYS.includes(r) || !hidden.includes(r)) &&
    (PAGE_FEATURES[r] || []).every((f) => info?.features?.[f] !== false);
  const items = menuOrder(user).filter(shown).map((n, i) => [...n.slice(0, 4), n[4] || n[1], tier(i)]);
  const last = LAST.filter(shown).map((n) => [...n, n[1], ""]);
  const link = ([r, label, ico, , , w]) => `<a href="#/${r}" data-route="${r}" class="nav-link ${ON_BAR[w]}">${ico} ${esc(label)}</a>`;
  const menuLink = ([r, label, ico, , , w]) =>
    `<a href="#/${r}" data-route="${r}" class="nav-menu-link ${IN_MORE[w]}"><span class="text-lg">${ico}</span><span class="min-w-0">${esc(label)}</span></a>`;
  $("#nav").innerHTML = items.filter((i) => i[5]).map(link).join("") +
    `<div class="relative"><button type="button" id="nav-more" class="nav-link" aria-haspopup="true" aria-expanded="false">☰ More</button>
      <div id="nav-menu" class="nav-menu hidden">${[...items.filter((i) => i[5] !== "md"), ...last].map(menuLink).join("")}</div></div>`;
  $("#nav-more").onclick = (e) => { e.stopPropagation(); showMenu($("#nav-menu").classList.contains("hidden")); };
  $$("#nav-menu a").forEach((a) => (a.onclick = () => showMenu(false)));
  if (!listening) {
    listening = true;
    document.addEventListener("click", (e) => { if (!$("#nav-menu")?.contains(e.target)) showMenu(false); });
    document.addEventListener("keydown", (e) => {
      if (e.key === "Escape" && !$("#nav-menu")?.classList.contains("hidden")) { showMenu(false); $("#nav-more").focus(); }
    });
  }

  const bar = items.slice(0, PHONE_BAR);
  const more = [...items.slice(PHONE_BAR), ...last];
  $("#mobile-nav").innerHTML = bar.map(([r, , ico, , short]) =>
    `<a href="#/${r}" data-route="${r}"><span class="ico">${ico}</span><span>${esc(short)}</span></a>`).join("") +
    `<button type="button" id="more-btn" data-more="${more.map((i) => i[0]).join(" ")}"><span class="ico">☰</span><span>More</span></button>`;
  $("#more-btn").onclick = () => {
    const d = sheet(`<div class="flex items-center"><h2 class="text-lg font-semibold">More</h2>
      <button type="button" data-close class="btn-ghost ml-auto">✕</button></div>
      <nav class="mt-2 grid grid-cols-2 gap-2">${more.map(([r, label, ico]) =>
        `<a href="#/${r}" class="card flex min-h-[3.5rem] items-center gap-2 p-3"><span class="text-2xl">${ico}</span><span class="min-w-0 break-words">${esc(label)}</span></a>`).join("")}</nav>
      <p class="mt-3 text-xs text-slate-500">Change the order of your pages on <a href="#/profile" class="underline">Me</a>.</p>`);
    $$("a", d).forEach((a) => (a.onclick = () => d.close()));
  };
}

// markNav lights up the current page, or More when the page is kept there.
export function markNav(route) {
  $$("[data-route]").forEach((a) => a.classList.toggle("active", a.dataset.route === route));
  const more = $("#more-btn");
  if (more) more.classList.toggle("active", more.dataset.more.split(" ").includes(route));
  const navMore = $("#nav-more");
  if (navMore) {
    const onBar = [...$$("#nav > a[data-route]")].some((a) => a.dataset.route === route && a.offsetParent !== null);
    navMore.classList.toggle("active", !onBar && Boolean($(`#nav-menu [data-route="${route}"]`)));
  }
}
