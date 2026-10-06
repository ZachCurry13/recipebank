// Navigation: every page in the top menu on a computer; on a phone a bottom
// bar with the main pages and a More sheet with the rest.
import { $, $$, esc, canManage, sheet } from "./ui.js";

const all = () => true;

// Pages: [route, label, icon, who may see it, on the phone's bottom bar].
export const NAV = [
  ["tonight", "Tonight", "🌙", all, true],
  ["kitchen", "Kitchen", "🍲", all, true],
  ["plan", "Plan", "📅", all, true],
  ["shopping", "Shopping", "🛒", all, true],
  ["make", "What can I make?", "🥕", all, false],
  ["home", "Home & Care", "🧴", all, false],
  ["pantry", "Pantry", "🥫", all, false],
  ["supplies", "Supplies", "🧽", all, false],
  ["collections", "Collections", "📚", all, false],
  ["add", "Add a recipe", "➕", canManage, false],
  ["family", "Family", "👪", all, false],
  ["admin", "Admin", "⚙️", (u) => u.role === "admin", false],
  ["profile", "Me", "🙂", all, false],
];

export function buildNav(user) {
  const items = NAV.filter(([, , , may]) => may(user));
  $("#nav").innerHTML = items.map(([r, label, ico]) => `<a href="#/${r}" data-route="${r}" class="nav-link">${ico} ${esc(label)}</a>`).join("");
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

// markNav lights up the current page (or More, for pages kept there).
export function markNav(route) {
  $$("[data-route]").forEach((a) => a.classList.toggle("active", a.dataset.route === route));
  const more = $("#more-btn");
  if (more) more.classList.toggle("active", more.dataset.more.split(" ").includes(route));
}
