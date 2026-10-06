// "Getting started" for whoever set up RecipeBank: a checklist on Tonight that
// ticks itself off as the family, recipes, AI and sign-ins are added.
import { get, put } from "./api.js";
import { $, $$, attempt, sheet } from "./ui.js";

// key: [what to do, where to do it, the "not now" button's words]
const STEPS = {
  admin: ["Create the admin account"],
  family: ["Add the family and their allergies", "#/family"],
  recipe: ["Add your first recipe", "#/add"],
  ai: ["Set up the AI for photos and smart search", "#/admin", "Skip"],
  users: ["Give everyone their own sign-in (Admin → Accounts)", "#/admin", "Just me"],
  phones: ["Put RecipeBank on your phones"],
};

export async function renderGuide(box, force = false) {
  let g = await get("/api/admin/guide");
  const tick = (step, done) => attempt(async () => { g = await put("/api/admin/guide", { step, done }); draw(); });
  const draw = () => {
    const left = g.steps.filter((s) => !s.done).length;
    if ((g.hidden && !force) || !left) {
      box.innerHTML = "";
      return;
    }
    box.innerHTML = `<div class="card mb-4 space-y-3">
      <div class="flex items-center gap-2"><h2 class="mr-auto font-semibold">🧭 Getting started</h2>
        <span class="text-sm text-slate-400">${g.steps.length - left} of ${g.steps.length}</span></div>
      <ol class="space-y-2">${g.steps.map((s) => {
        const [label, href, skip] = STEPS[s.key];
        return `<li class="flex flex-wrap items-center gap-2">
          <span class="${s.done ? "text-emerald-300" : "text-slate-500"}" aria-hidden="true">${s.done ? "✓" : "○"}</span>
          <span class="min-w-0 flex-1 basis-40 break-words ${s.done ? "text-slate-400 line-through" : ""}">${label}</span>
          ${s.done ? "" : `<span class="ml-auto flex shrink-0 gap-1">${href ? `<a href="${href}" class="btn-secondary min-h-0 py-1 text-sm">Go</a>` : ""}
            ${s.key === "phones" ? `<button type="button" data-guide-phones class="btn-secondary min-h-0 py-1 text-sm">How</button>` : ""}
            ${skip ? `<button type="button" data-guide-skip="${s.key}" class="btn-ghost min-h-0 py-1 text-sm">${skip}</button>` : ""}</span>`}</li>`;
      }).join("")}</ol>
      <button type="button" data-guide-hide class="btn-ghost text-sm">Hide this guide</button></div>`;
    $$("[data-guide-skip]", box).forEach((b) => (b.onclick = () => tick(b.dataset.guideSkip, true)));
    $("[data-guide-hide]", box).onclick = () => { force = false; tick("hidden", true); };
    const phones = $("[data-guide-phones]", box);
    if (phones) phones.onclick = () => phoneHelp(() => tick("phones", true));
  };
  draw();
}

// phoneHelp explains installing on phones; done ticks the step.
function phoneHelp(done) {
  const d = sheet(`<div class="space-y-3">
    <div class="flex items-center"><h2 class="text-lg font-semibold">📱 On your phones</h2>
      <button type="button" data-close class="btn-ghost ml-auto" aria-label="Close">✕</button></div>
    <ol class="list-decimal space-y-2 pl-5 text-sm">
      <li>Phones install apps, use the live camera and get notifications only from an address starting with
        <b>https://</b>. The easiest way: <b>Admin → Remote access</b> (a free Cloudflare account), which also works away from home.</li>
      <li>Open that address on each phone and sign in.</li>
      <li><b>iPhone:</b> Safari → <b>Share</b> → <b>Add to Home Screen</b>. <b>Android:</b> Chrome → <b>⋮</b> → <b>Install app</b>.</li>
      <li>Optional: on each phone, <b>Me → Phone notifications</b> for timers and reminders.</li>
    </ol>
    <div class="flex flex-wrap gap-2"><a href="#/admin" data-close class="btn-secondary">Open Admin</a>
      <button type="button" id="phones-done" class="btn-primary">✓ Done</button></div></div>`);
  $("#phones-done", d).onclick = () => { d.close(); done(); };
}
