// Family: everyone who eats here, their allergies, diets and dislikes, and
// the household's pets (for Home & Care checks).
import { get, post, put, del } from "./api.js";
import { $, $$, esc, attempt, canManage, sheet, toast, HEAT } from "./ui.js";
import { refreshInfo } from "./app.js";
import { allowFields, wireAllowFields, allowFor } from "./allowfields.js";

const SEVERITY = [
  ["", "No"],
  ["avoid", "Avoid", "Doesn't want it. Only ingredients that clearly contain it count."],
  ["allergic", "Allergic", "Never. Also flags \"may contain\" and packaged foods, so you check the label."],
  ["severe", "Severe (strict)", "Every ingredient must be a plain food RecipeBank knows, or one whose label a parent checked. Anything unknown shows as \"not sure\"."],
];
const PETS = [["dog", "🐕 Dogs"], ["cat", "🐈 Cats"], ["bird", "🐦 Birds"], ["small", "🐇 Small pets"], ["fish", "🐟 Fish"]];
const SENSITIVITY_IDEAS = ["essential oils", "fragrance", "coconut", "alcohol", "tea tree", "lanolin", "beeswax"];

export async function renderFamily(view, params, state) {
  const [people, info] = [await get("/api/people"), state.info || (await refreshInfo())];
  const manage = canManage(state.user);
  const pets = new Set(info.pets || []);
  const allergenName = (k) => (info.all_allergens || []).find((a) => a.key === k)?.label || k;
  const dietName = (k) => (info.diets || []).find((d) => d.key === k)?.label || k;
  view.innerHTML = `
    <div class="mb-4 flex flex-wrap items-end gap-3">
      <div class="min-w-0 basis-full sm:basis-auto sm:flex-1"><h1 class="text-2xl font-bold">👪 Family</h1>
        <p class="text-sm text-slate-400">Everyone who eats here. Each recipe is checked against their allergies, diets and dislikes.</p></div>
      ${manage ? `<button id="add" class="btn-primary">➕ Add a person</button>` : ""}
    </div>
    ${people.length ? "" : `<div class="card mb-4 text-slate-300">Start by adding each person in the family, and anyone who eats with you often.
      Guests (like a grandparent who visits) can be added too.</div>`}
    <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">${people.map((p) => {
      const rules = p.rules.map((r) => r.kind === "allergy"
        ? `<span class="${r.severity === "avoid" ? "chip-unsure" : "chip-no"}">${esc(allergenName(r.key))}: ${esc(SEVERITY.find((s) => s[0] === r.severity)?.[1] || r.severity)}${
          r.allow?.length ? ` (can have ${esc(r.allow.join(", "))})` : ""}</span>`
        : r.kind === "diet" ? `<span class="chip-cat">${esc(dietName(r.key))}</span>`
        : r.kind === "dislike" ? `<span class="chip-info">No ${esc(r.key)}</span>`
        : `<span class="chip-info">Sensitive: ${esc(r.key)}</span>`).join(" ");
      return `<div class="card min-w-0 space-y-2">
        <div class="flex flex-wrap items-center gap-2"><h2 class="min-w-0 break-words text-lg font-semibold">${esc(p.name)}</h2>
          ${p.is_kid ? `<span class="chip-cat">Kid</span>` : ""}${p.is_guest ? `<span class="chip-info">Guest</span>` : ""}
          ${manage ? `<button data-edit="${p.id}" class="btn-ghost ml-auto min-h-0 px-2 py-1">✎ Edit</button>` : ""}</div>
        <div class="flex flex-wrap gap-1">${rules || `<span class="text-sm text-slate-400">Eats anything</span>`}
          ${p.heat_max >= 0 ? `<span class="chip-info">Heat up to ${p.heat_max ? "🌶️".repeat(p.heat_max) : "none"}</span>` : ""}</div>
      </div>`;
    }).join("")}</div>
    <div class="card mt-6 space-y-2">
      <h2 class="font-semibold">Pets at home</h2>
      <p class="text-sm text-slate-400">Home & Care recipes warn about ingredients that are dangerous for them (like essential oils for cats, or xylitol for dogs).</p>
      <div class="flex flex-wrap gap-2">${PETS.map(([k, l]) =>
        `<button data-pet="${k}" class="pick ${pets.has(k) ? "on" : ""}" ${manage ? "" : "disabled"}>${l}</button>`).join("")}</div>
    </div>`;
  const add = $("#add", view);
  if (add) add.onclick = () => editPerson(null, info, () => renderFamily(view, params, state));
  $$("[data-edit]", view).forEach((b) => (b.onclick = () =>
    editPerson(people.find((p) => p.id === Number(b.dataset.edit)), info, () => renderFamily(view, params, state))));
  $$("[data-pet]", view).forEach((b) => (b.onclick = () => attempt(async () => {
    b.classList.toggle("on");
    const on = $$("[data-pet].on", view).map((x) => x.dataset.pet);
    await put("/api/household", { pets: on });
    await refreshInfo();
  })));
}

function editPerson(p, info, done) {
  p = p || { id: 0, name: "", is_kid: false, is_guest: false, heat_max: -1, rules: [] };
  const rule = (kind, key) => p.rules.find((r) => r.kind === kind && r.key === key);
  const list = (kind) => p.rules.filter((r) => r.kind === kind).map((r) => r.key).join(", ");
  const d = sheet(`
    <form id="pf" class="space-y-4">
      <div class="flex items-center"><h2 class="text-lg font-semibold">${p.id ? "Edit " + esc(p.name) : "Add a person"}</h2>
        <button type="button" data-close class="btn-ghost ml-auto">✕</button></div>
      <label class="block"><span class="label">Name</span><input name="name" required maxlength="60" class="input" value="${esc(p.name)}"></label>
      <div class="flex flex-wrap gap-4">
        <label class="toggle"><input type="checkbox" name="is_kid" ${p.is_kid ? "checked" : ""}> A kid</label>
        <label class="toggle"><input type="checkbox" name="is_guest" ${p.is_guest ? "checked" : ""}> A guest (not picked by default)</label>
      </div>
      <div><span class="label">Allergies</span>
        <details class="mb-2 text-xs text-slate-400"><summary class="cursor-pointer">What do Avoid, Allergic and Severe mean?</summary>
          <ul class="mt-1 space-y-1">${SEVERITY.slice(1).map(([, l, h]) => `<li><b>${l}:</b> ${h}</li>`).join("")}</ul></details>
        <div class="space-y-2">${info.allergens.map((a) => `
          <label class="flex flex-wrap items-center gap-2"><span class="min-w-0 flex-1 basis-40 text-sm">${esc(a.label)}</span>
            <select name="allergy_${a.key}" class="input w-auto">${SEVERITY.map(([v, l]) =>
              `<option value="${v}" ${(rule("allergy", a.key)?.severity || "") === v ? "selected" : ""}>${l}</option>`).join("")}</select></label>
          ${allowFields(a, rule("allergy", a.key))}`).join("")}</div>
      </div>
      <div><span class="label">Diets</span><div class="flex flex-wrap gap-2">${info.diets.map((dt) =>
        `<label class="pick ${rule("diet", dt.key) ? "on" : ""}"><input type="checkbox" class="sr-only" name="diet" value="${dt.key}" ${rule("diet", dt.key) ? "checked" : ""}>${esc(dt.label)}</label>`).join("")}</div></div>
      <label class="block"><span class="label">Dislikes (comma between them)</span>
        <input name="dislikes" class="input" value="${esc(list("dislike"))}" placeholder="mushrooms, olives, cilantro"></label>
      <label class="block"><span class="label">Skin or mouth sensitivities (for Home & Care)</span>
        <input name="sensitivities" class="input" value="${esc(list("sensitivity"))}" placeholder="${SENSITIVITY_IDEAS.slice(0, 3).join(", ")}"></label>
      <label class="block"><span class="label">Most heat they like</span><select name="heat_max" class="input">
        <option value="-1">No limit</option>${HEAT.map((l, i) => `<option value="${i}" ${p.heat_max === i ? "selected" : ""}>${i ? "🌶️".repeat(i) + " " : ""}${l}</option>`).join("")}</select></label>
      <div class="flex flex-wrap gap-2 pt-2"><button class="btn-primary">Save</button>
        ${p.id ? `<button type="button" id="rm" class="btn-ghost ml-auto text-rose-300">Remove ${esc(p.name)}</button>` : ""}</div>
    </form>`);
  $$('input[name="diet"]', d).forEach((i) => (i.onchange = () => i.parentElement.classList.toggle("on", i.checked)));
  wireAllowFields(d);
  $("#pf", d).onsubmit = (e) => {
    e.preventDefault();
    const f = e.target;
    const rules = [];
    for (const a of info.allergens) {
      const sev = f["allergy_" + a.key].value;
      if (sev) rules.push({ kind: "allergy", key: a.key, severity: sev, allow: allowFor(f, a.key) });
    }
    // Keep allergies from the other list (US/EU) that aren't shown here.
    for (const r of p.rules) if (r.kind === "allergy" && !info.allergens.some((a) => a.key === r.key)) rules.push(r);
    $$('input[name="diet"]:checked', f).forEach((i) => rules.push({ kind: "diet", key: i.value }));
    const words = (s) => [...new Set(s.split(",").map((w) => w.trim()).filter(Boolean))];
    words(f.dislikes.value).forEach((w) => rules.push({ kind: "dislike", key: w }));
    words(f.sensitivities.value).forEach((w) => rules.push({ kind: "sensitivity", key: w }));
    const body = { name: f.name.value, is_kid: f.is_kid.checked, is_guest: f.is_guest.checked, heat_max: Number(f.heat_max.value), rules };
    attempt(async () => {
      await (p.id ? put(`/api/people/${p.id}`, body) : post("/api/people", body));
      d.close();
      toast("Saved");
      done();
    });
  };
  const rm = $("#rm", d);
  if (rm) rm.onclick = () => {
    if (!confirm(`Remove ${p.name}?`)) return;
    attempt(async () => {
      await del(`/api/people/${p.id}`);
      d.close();
      done();
    });
  };
}
