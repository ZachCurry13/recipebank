// "What's new": the running version, whether a newer one is out, and the
// release notes (newer ones from GitHub, the rest bundled with this version).
import { get } from "./api.js";
import { esc, canManage } from "./ui.js";
import { renderMarkdown, changelogSections } from "./markdown.js";

const GUIDE = "https://github.com/ZachCurry13/recipebank/blob/main/docs/TRUENAS.md#updating-recipebank";

export async function renderWhatsNew(view, params, state) {
  const data = await get("/api/updates");
  const st = data.status || { current: "unknown", releases: [] };
  const manager = canManage(state.user);
  const newer = st.releases.filter((r) => r.newer);

  let banner = "";
  if (st.update_available && manager) {
    banner = `<div class="box-info mb-4 space-y-2">
      <p class="font-semibold">RecipeBank ${esc(st.latest.replace(/^v/, ""))} is out. You have ${esc(st.current)}.</p>
      <p class="text-sm">To update on TrueNAS: <b>Apps → recipebank → Update</b>. Your recipes, people and photos are kept.
        If there's no Update button yet, see <a href="${GUIDE}" target="_blank" rel="noopener noreferrer" class="underline">Updating RecipeBank</a>.</p></div>`;
  } else if (manager && data.checks_enabled && st.latest && !st.error && /^v?\d/.test(st.current)) {
    banner = `<p class="mb-4 text-sm text-emerald-300">✓ You have the newest version.</p>`;
  } else if (manager && st.error) {
    banner = `<p class="mb-4 text-sm text-amber-300">Couldn't reach GitHub to look for a newer version. Showing the notes that came with this one.</p>`;
  }

  const upcoming = newer.map((r) => section(`${r.tag.replace(/^v/, "")} (new)`, r.notes, r.published_at, r.url)).join("");
  const seen = new Set(newer.map((r) => r.tag.replace(/^v/, "")));
  const bundled = changelogSections(data.changelog).filter((s) => !seen.has(s.version))
    .map((s) => section(`${s.version}${sameVersion(s.version, st.current) ? " (this version)" : ""}`, s.body)).join("");

  view.innerHTML = `
    <h1 class="mb-1 text-2xl font-bold">🆕 What's new</h1>
    <p class="mb-4 text-sm text-slate-400">You're using RecipeBank <b>${esc(st.current)}</b>.</p>
    ${banner}
    <div class="space-y-4">${upcoming}${bundled}</div>`;
}

const sameVersion = (a, b) => String(b || "").replace(/^v/, "").split("-")[0] === a;

function section(title, notes, date, url) {
  return `<article class="card min-w-0">
    <div class="flex flex-wrap items-baseline justify-between gap-2">
      <h2 class="text-lg font-bold">${esc(title)}</h2>
      <span class="text-xs text-slate-500">${date ? esc(new Date(date).toLocaleDateString()) : ""}
        ${url ? ` · <a href="${esc(url)}" target="_blank" rel="noopener noreferrer" class="underline">GitHub</a>` : ""}</span>
    </div>
    <div class="break-words text-sm leading-relaxed text-slate-300">${renderMarkdown(notes)}</div>
  </article>`;
}
