// Navigation. On a computer: the main pages in the top bar (more of them on
// wider screens) and a More menu with the rest, so the bar never scrolls.
// On a phone: a bottom bar with the main pages and a More sheet.
import { $, $$, esc, canManage, sheet } from "./ui.js";

const all = () => true;

// Pages: [route, label, icon, who may see it, on the phone's bottom bar,
// on the computer's top bar from which width ("md" always, "lg", "xl"; "" = in More)].
export const NAV = [
  ["tonight", "Tonight", "🌙", all, true, "md"],
  ["kitchen", "Kitchen", "🍲", all, true, "md"],
  ["plan", "Plan", "📅", all, true, "md"],
  ["shopping", "Shopping", "🛒", all, true, "md"],
  ["make", "What can I make?", "🥕", all, false, "lg"],
  ["pantry", "Pantry", "🥫", all, false, "lg"],
  ["home", "Home & Care", "🧴", all, false, "xl"],
  ["supplies", "Supplies", "🧽", all, false, ""],
  ["collections", "Collections", "📚", all, false, ""],
  ["books", "Bookshelf", "📖", all, false, ""],
  ["events", "Events", "🎉", all, false, ""],
  ["add", "Add a recipe", "➕", canManage, false, ""],
  ["family", "Family", "👪", all, false, ""],
  ["admin", "Admin", "⚙️", (u) => u.role === "admin", false, ""],
  ["profile", "Me", "🙂", all, false, ""],
];

// Pages on the top bar from a width on are hidden below it, and the other way round in More.
const ON_BAR = { md: "", lg: "hidden lg:block", xl: "hidden xl:block" };
const IN_MORE = { md: "hidden", lg: "lg:hidden", xl: "xl:hidden", "": "" };

let listening = false; // the page-wide listeners that close the More menu, added once

function showMenu(open) {
  const menu = $("#nav-menu");
  if (!menu) return;
  menu.classList.toggle("hidden", !open);
  $("#nav-more").setAttribute("aria-expanded", String(open));
}

export function buildNav(user) {
  const items = NAV.filter(([, , , may]) => may(user));
  const top = $("#nav");
  top.innerHTML = items.filter((i) => i[5]).map(([r, label, ico, , , w]) =>
    `<a href="#/${r}" data-route="${r}" class="nav-link ${ON_BAR[w]}">${ico} ${esc(label)}</a>`).join("") +
    `<div class="relative"><button type="button" id="nav-more" class="nav-link" aria-haspopup="true" aria-expanded="false">☰ More</button>
      <div id="nav-menu" class="nav-menu hidden">${items.filter((i) => i[5] !== "md").map(([r, label, ico, , , w]) =>
        `<a href="#/${r}" data-route="${r}" class="nav-menu-link ${IN_MORE[w]}"><span class="text-lg">${ico}</span><span class="min-w-0">${esc(label)}</span></a>`).join("")}</div></div>`;
  $("#nav-more").onclick = (e) => { e.stopPropagation(); showMenu($("#nav-menu").classList.contains("hidden")); };
  $$("#nav-menu a").forEach((a) => (a.onclick = () => showMenu(false)));
  if (!listening) {
    listening = true;
    document.addEventListener("click", (e) => { if (!$("#nav-menu")?.contains(e.target)) showMenu(false); });
    document.addEventListener("keydown", (e) => {
      if (e.key === "Escape" && !$("#nav-menu")?.classList.contains("hidden")) { showMenu(false); $("#nav-more").focus(); }
    });
  }

  const bar = items.filter((i) => i[4]);
  const more = items.filter((i) => !i[4]);
  $("#mobile-nav").innerHTML = bar.map(([r, label, ico]) =>
    `<a href="#/${r}" data-route="${r}"><span class="ico">${ico}</span><span>${esc(label)}</span></a>`).join("") +
    `<button type="button" id="more-btn" data-more="${more.map((i) => i[0]).join(" ")}"><span class="ico">☰</span><span>More</span></button>`;
  $("#more-btn").onclick = () => {
    const d = sheet(`<div class="flex items-center"><h2 class="text-lg font-semibold">More</h2>
      <button type="button" data-close class="btn-ghost ml-auto">✕</button></div>
      <nav class="mt-2 grid grid-cols-2 gap-2">${more.map(([r, label, ico]) =>
        `<a href="#/${r}" class="card flex min-h-[3.5rem] items-center gap-2 p-3"><span class="text-2xl">${ico}</span><span class="min-w-0 break-words">${esc(label)}</span></a>`).join("")}</nav>`);
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
