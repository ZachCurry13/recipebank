// "Read the card again": the saved card photos, turned the right way up if
// needed, read once more. The new lines are shown beside the old ones, and
// nothing changes until someone chooses the new reading.
import { post, put } from "./api.js";
import { $, $$, esc, attempt, busy, sheet, toast } from "./ui.js";
import { rotatePhoto } from "./photo.js";
import { cardBox } from "./cardcheck.js";

async function asDataURL(name) {
  const res = await fetch(`/api/photos/${encodeURIComponent(name)}`);
  if (!res.ok) throw new Error("A card photo couldn't be loaded.");
  const blob = await res.blob();
  return new Promise((resolve, reject) => {
    const fr = new FileReader();
    fr.onload = () => resolve(fr.result);
    fr.onerror = () => reject(new Error("A card photo couldn't be read."));
    fr.readAsDataURL(blob);
  });
}

// diff lines up the old and new lines: [{t: "same"|"old"|"new", text}].
export function diff(a, b) {
  const k = (s) => s.trim().toLowerCase().replace(/\s+/g, " ");
  const L = Array.from({ length: a.length + 1 }, () => new Array(b.length + 1).fill(0));
  for (let i = a.length - 1; i >= 0; i--) {
    for (let j = b.length - 1; j >= 0; j--) {
      L[i][j] = k(a[i]) === k(b[j]) ? L[i + 1][j + 1] + 1 : Math.max(L[i + 1][j], L[i][j + 1]);
    }
  }
  const out = [];
  let i = 0, j = 0;
  while (i < a.length && j < b.length) {
    if (k(a[i]) === k(b[j])) { out.push({ t: "same", text: b[j] }); i++; j++; }
    else if (L[i + 1][j] >= L[i][j + 1]) out.push({ t: "old", text: a[i++] });
    else out.push({ t: "new", text: b[j++] });
  }
  while (i < a.length) out.push({ t: "old", text: a[i++] });
  while (j < b.length) out.push({ t: "new", text: b[j++] });
  return out;
}

const ingLines = (r) => (r.ingredients || []).map((i) => i.line + (i.unsure ? " (?)" : ""));
const stepLines = (r) => (r.steps || []).map((s) => s.text + (s.unsure ? " (?)" : ""));

function diffList(title, rows) {
  const mark = { same: ["", "text-slate-300"], old: ["−", "text-rose-300 line-through"], new: ["+", "text-emerald-300"] };
  return `<h3 class="mt-3 font-semibold">${title}</h3><ul class="space-y-1 text-sm">${rows.map((d) =>
    `<li class="flex gap-2 ${mark[d.t][1]}"><span class="w-3 shrink-0" aria-hidden="true">${mark[d.t][0]}</span>
      <span class="min-w-0 break-words">${d.t === "same" ? "" : `<span class="sr-only">${d.t === "old" ? "Gone" : "New"}: </span>`}${esc(d.text)}</span></li>`).join("")}</ul>`;
}

// readAgain opens the sheet for recipe r; done() reloads the page after a change.
export async function readAgain(r, done) {
  const photos = await Promise.all(r.source_photos.map(asDataURL));
  const turned = photos.map(() => false);
  const d = sheet(`<div class="space-y-3">
    <div class="flex items-center"><h2 class="text-lg font-semibold">Read the card again</h2>
      <button type="button" data-close class="btn-ghost ml-auto" aria-label="Close">✕</button></div>
    <p class="text-sm text-slate-300">Is the writing the right way up? Tap ⟳ to turn a photo. It's read with the AI set
      under Admin → AI, and nothing changes until you choose.</p>
    <div id="ra-photos" class="flex flex-wrap gap-2"></div>
    <button type="button" id="ra-go" class="btn-primary">Read it</button>
    <div id="ra-out"></div></div>`);
  const drawPhotos = () => {
    $("#ra-photos", d).innerHTML = photos.map((p, i) => `<div class="relative">
      <img src="${p}" alt="Card photo ${i + 1}" class="h-28 rounded-lg object-cover">
      <button type="button" data-rot="${i}" class="btn-secondary absolute bottom-1 right-1 min-h-0 px-2 py-1" aria-label="Turn photo ${i + 1}">⟳</button></div>`).join("");
    $$("[data-rot]", d).forEach((b) => (b.onclick = () => attempt(async () => {
      const i = Number(b.dataset.rot);
      photos[i] = await rotatePhoto(photos[i]);
      turned[i] = true;
      drawPhotos();
    })));
  };
  drawPhotos();
  const go = $("#ra-go", d);
  go.onclick = () => attempt(() => busy(go, "Reading… (this can take a minute)", async () => {
    const res = await post("/api/import/photo/again", { images: photos, area: r.area });
    show(res.recipe, res.card_check);
  }));

  const show = (nr, problems) => {
    const ings = diff(ingLines(r), ingLines(nr));
    const steps = diff(stepLines(r), stepLines(nr));
    const changed = [...ings, ...steps].filter((x) => x.t !== "same").length;
    $("#ra-out", d).innerHTML = `<div class="space-y-2 border-t border-slate-700 pt-3">
      <p class="font-semibold">${changed ? `${changed} line${changed === 1 ? "" : "s"} differ` : "The new reading matches what you have."}
        <span class="block text-xs font-normal text-slate-500">Read by ${esc(nr.read_by || "the AI")}. <span class="text-emerald-300">+ new</span> · <span class="text-rose-300">− gone</span></span></p>
      ${diffList("Ingredients", ings)}${diffList("Steps", steps)}
      ${cardBox({ ...nr, needs_review: (problems || []).length > 0 || nr.needs_review }, problems, "")}
      <div class="flex flex-wrap gap-2 pt-2">
        ${changed ? `<button type="button" id="ra-keep" class="btn-primary">Keep the new reading</button>` : ""}
        <button type="button" data-close class="btn-secondary">Keep what I have</button></div></div>`;
    $$("[data-close]", d).forEach((b) => (b.onclick = () => d.close()));
    const keep = $("#ra-keep", d);
    if (keep) keep.onclick = () => attempt(() => busy(keep, "Saving…", async () => {
      // Turned photos replace the old ones; label checks carry over by line.
      const names = await Promise.all(r.source_photos.map(async (n, i) =>
        (turned[i] ? (await post("/api/photos", { image: photos[i] })).name : n)));
      const checked = Object.fromEntries(r.ingredients.filter((i) => i.checked?.length).map((i) => [i.line, i.checked]));
      await put(`/api/recipes/${r.id}`, { ...r, ingredients: nr.ingredients.map((i) => ({ ...i, checked: checked[i.line] })),
        steps: nr.steps, servings: r.servings || nr.servings, source_photos: names, needs_review: true,
        read_by: nr.read_by, ai_reading: nr.ai_reading });
      d.close();
      toast("Kept the new reading: check it against the card");
      await done();
    }));
  };
}
