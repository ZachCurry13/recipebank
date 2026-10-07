// An event's guest page (/e/{token}, no account): say you're coming, with
// your allergies and diets, see the menu marked for you, and add what you're
// bringing. This browser keeps a key so you can change what you said.
import { $, $$, esc, toast } from "./ui.js";

if (!matchMedia("(prefers-color-scheme: dark)").matches) document.documentElement.classList.add("light");

const token = location.pathname.split("/").pop();
const keyName = `rb:guest:${token}`;
const read = () => { try { return localStorage.getItem(keyName) || ""; } catch { return ""; } };
const keep = (k) => { try { k ? localStorage.setItem(keyName, k) : localStorage.removeItem(keyName); } catch { /* private mode */ } };
const box = $("#guest");

async function call(method, path, body) {
  const res = await fetch(`/api/guest/${encodeURIComponent(token)}${path}`, {
    method, headers: { "X-RecipeBank": "1", "X-Guest-Key": read(), ...(body ? { "Content-Type": "application/json" } : {}) },
    body: body ? JSON.stringify(body) : undefined });
  const data = res.status === 204 ? {} : await res.json().catch(() => ({}));
  if (!res.ok) throw Object.assign(new Error(data.error || `Something went wrong (${res.status})`), { status: res.status });
  return data;
}

const MARK = { ok: ["chip-ok", "✓ Fine for you"], no: ["chip-no", "✕ Not for you"], unsure: ["chip-unsure", "? Ask the cook"] };
const dateText = (d) => (d ? new Date(d + "T00:00:00").toLocaleDateString(undefined, { weekday: "long", month: "long", day: "numeric" }) : "");
const tick = (name, list, chosen) => list.map((x) => `<label class="pick ${chosen.includes(x.key) ? "on" : ""}">
  <input type="checkbox" class="sr-only" name="${name}" value="${esc(x.key)}" ${chosen.includes(x.key) ? "checked" : ""}>${esc(x.label)}</label>`).join("");

async function draw() {
  let v;
  try {
    v = await call("GET", "");
  } catch (e) {
    box.innerHTML = `<div class="card space-y-2"><h1 class="text-xl font-bold">${e.status === 410 ? "This link has run out" : "Can't open this"}</h1>
      <p class="text-sm text-slate-400">${esc(e.message)}</p></div>`;
    return;
  }
  const me = v.me;
  const allergens = (v.allergens || []).map((a) => ({ key: a.Key, label: a.Label }));
  const diets = v.diets || [];
  document.title = `${v.event.name} · RecipeBank`;
  box.innerHTML = `
    <div><h1 class="break-words text-2xl font-bold">🎉 ${esc(v.event.name)}</h1>
      <p class="text-sm text-slate-400">${esc(dateText(v.event.date))}${v.event.date ? " · " : ""}${v.coming} coming</p>
      ${v.event.notes ? `<p class="mt-2 whitespace-pre-line break-words text-slate-300">${esc(v.event.notes)}</p>` : ""}</div>
    ${me ? `<div class="card flex flex-wrap items-center gap-2"><span class="min-w-0 basis-full break-words">✓ You're coming as <b>${esc(me.name)}</b>.</span>
        <button type="button" id="change" class="btn-ghost">Change</button>
        <button type="button" id="leave" class="btn-ghost text-rose-300">Not coming after all</button></div>` : ""}
    <form id="me" class="card space-y-3 ${me ? "hidden" : ""}">
      <h2 class="font-semibold">${me ? "Change what you said" : "Are you coming?"}</h2>
      <label class="block"><span class="label">Your name</span><input name="name" required maxlength="40" class="input" value="${esc(me?.name || "")}"></label>
      <div><span class="label">Allergies (the hosts will see these)</span><div class="flex flex-wrap gap-2">${tick("allergy", allergens, Object.keys(me?.allergies || {}))}</div></div>
      <div><span class="label">Diets</span><div class="flex flex-wrap gap-2">${tick("diet", diets, me?.diets || [])}</div></div>
      <label class="block"><span class="label">Anything else? (optional)</span><input name="note" maxlength="200" class="input" value="${esc(me?.note || "")}"></label>
      <button class="btn-primary">${me ? "Save" : "I'm coming"}</button></form>
    <div class="card space-y-3"><h2 class="font-semibold">The menu</h2>
      ${v.dishes.length ? `<ul class="divide-y divide-slate-800">${v.dishes.map((d) => `<li class="flex flex-wrap items-start gap-2 py-2">
        <span class="min-w-0 flex-1 basis-48"><span class="block break-words font-medium">${esc(d.title)}</span>
          <span class="block text-xs text-slate-400">Brought by ${esc(d.brings)}${d.contains.length ? ` · has ${esc(d.contains.join(", ").toLowerCase())}` : ""}</span></span>
        ${d.status && !d.mine ? `<span class="${MARK[d.status][0]}">${MARK[d.status][1]}</span>` : ""}
        ${d.mine ? `<button type="button" data-drop="${d.id}" class="btn-ghost min-h-0 px-2 py-1 text-slate-500" aria-label="Take it off">✕</button>` : ""}</li>`).join("")}</ul>`
        : `<p class="text-sm text-slate-400">Nothing on the menu yet.</p>`}
      ${me ? "" : `<p class="text-xs text-slate-500">Say you're coming to see which dishes suit you.</p>`}</div>
    ${me ? `<form id="dish" class="card space-y-3"><h2 class="font-semibold">What are you bringing?</h2>
      <input name="title" required maxlength="80" class="input" placeholder="Green bean casserole">
      <div><span class="label">It has (so everyone can check)</span><div class="flex flex-wrap gap-2">${tick("contains", allergens, [])}</div></div>
      <button class="btn-primary">Add it to the menu</button></form>` : ""}`;

  $$("label.pick input", box).forEach((c) => (c.onchange = () => c.parentElement.classList.toggle("on", c.checked)));
  const form = $("#me", box);
  form.onsubmit = async (e) => {
    e.preventDefault();
    const allergies = Object.fromEntries($$('input[name="allergy"]:checked', form).map((c) => [c.value, "allergic"]));
    try {
      const res = await call("PUT", "/me", { name: form.name.value, allergies, note: form.note.value,
        diets: $$('input[name="diet"]:checked', form).map((c) => c.value) });
      if (res.key) keep(res.key);
      toast(me ? "Saved" : "See you there!");
      draw();
    } catch (err) { toast(err.message, true); }
  };
  const change = $("#change", box);
  if (change) change.onclick = () => form.classList.toggle("hidden");
  const leave = $("#leave", box);
  if (leave) leave.onclick = async () => {
    if (!confirm("Take yourself (and what you're bringing) off this event?")) return;
    try { await call("DELETE", "/me"); keep(""); draw(); } catch (err) { toast(err.message, true); }
  };
  const dish = $("#dish", box);
  if (dish) dish.onsubmit = async (e) => {
    e.preventDefault();
    try {
      await call("POST", "/dishes", { title: dish.title.value, contains: $$('input[name="contains"]:checked', dish).map((c) => c.value) });
      toast("Added to the menu");
      draw();
    } catch (err) { toast(err.message, true); }
  };
  $$("[data-drop]", box).forEach((b) => (b.onclick = async () => {
    try { await call("DELETE", `/dishes/${b.dataset.drop}`); draw(); } catch (err) { toast(err.message, true); }
  }));
}

draw();
