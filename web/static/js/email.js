// "✉️ Email": send a recipe or the shopping list to one address, through the
// mail server set up under Admin → Email. Admin → Email lives here too.
import { post, put } from "./api.js";
import { $, esc, attempt, busy, sheet, toast } from "./ui.js";
import { refreshInfo } from "./app.js";

const LAST = "recipebank.lastEmailTo";
const lastTo = () => { try { return localStorage.getItem(LAST) || ""; } catch { return ""; } };

// emailSheet asks where to send it; withNote adds a message box.
export function emailSheet(state, title, path, withNote) {
  if (!state.info?.email_ready) {
    sheet(`<div class="space-y-3"><div class="flex items-center"><h2 class="text-lg font-semibold">✉️ Email</h2>
      <button type="button" data-close class="btn-ghost ml-auto" aria-label="Close">✕</button></div>
      <p class="text-sm text-slate-300">Email isn't set up yet. ${state.user.role === "admin" ? "Set it up under <b>Admin → Email</b>." : "Ask whoever set up RecipeBank to set it up under Admin → Email."}</p></div>`);
    return;
  }
  const d = sheet(`<form id="mailf" class="space-y-3">
    <div class="flex items-center"><h2 class="min-w-0 break-words text-lg font-semibold">✉️ Email ${esc(title)}</h2>
      <button type="button" data-close class="btn-ghost ml-auto" aria-label="Close">✕</button></div>
    <label class="block"><span class="label">To</span><input name="to" type="email" required autocomplete="email" class="input"
      value="${esc(lastTo())}" placeholder="name@example.com"></label>
    ${withNote ? `<label class="block"><span class="label">Message (optional)</span>
      <textarea name="note" rows="3" maxlength="1000" class="input" placeholder="You have to try these!"></textarea></label>` : ""}
    <button class="btn-primary">Send</button></form>`);
  $("#mailf", d).onsubmit = (e) => {
    e.preventDefault();
    const f = e.target;
    attempt(() => busy(e.submitter, "Sending…", async () => {
      await post(path, { to: f.to.value.trim(), note: f.note?.value || "" });
      try { localStorage.setItem(LAST, f.to.value.trim()); } catch { /* private mode */ }
      d.close();
      toast(`Sent to ${f.to.value.trim()}`);
    }));
  };
}

// renderEmailAdmin is Admin → Email; s are the settings.
export function renderEmailAdmin(box, s) {
  box.innerHTML = `<form id="mail-admin" class="card space-y-3">
    <h2 class="font-semibold">✉️ Email</h2>
    <p class="text-sm text-slate-400">For emailing recipes and the shopping list, through your own email account.
      For Gmail: server <b>smtp.gmail.com</b>, port <b>587</b>, your Gmail address, and an
      <a href="https://myaccount.google.com/apppasswords" target="_blank" rel="noopener noreferrer" class="underline">App Password</a>.</p>
    <div class="grid gap-3 sm:grid-cols-[minmax(0,1fr)_7rem]">
      <label class="block"><span class="label">Mail server</span><input name="smtp_host" class="input" value="${esc(s.smtp_host || "")}" placeholder="smtp.gmail.com"></label>
      <label class="block"><span class="label">Port</span><input name="smtp_port" inputmode="numeric" class="input" value="${esc(s.smtp_port || "587")}"></label></div>
    <label class="block"><span class="label">Username</span><input name="smtp_username" autocomplete="off" class="input" value="${esc(s.smtp_username || "")}"></label>
    <label class="block"><span class="label">Password ${s.smtp_password_set ? "(saved; leave blank to keep it)" : ""}</span>
      <input name="smtp_password" type="password" autocomplete="new-password" class="input"></label>
    <label class="block"><span class="label">Send from</span><input name="smtp_from" type="email" class="input" value="${esc(s.smtp_from || "")}" placeholder="you@gmail.com"></label>
    <div class="flex flex-wrap gap-2"><button class="btn-primary">Save</button>
      <input id="mail-test-to" type="email" class="input min-w-0 flex-1 basis-40" placeholder="Send a test to…" aria-label="Test address">
      <button type="button" id="mail-test" class="btn-secondary">Send a test</button></div></form>`;
  const f = $("#mail-admin", box);
  f.onsubmit = (e) => {
    e.preventDefault();
    const body = Object.fromEntries(["smtp_host", "smtp_port", "smtp_username", "smtp_password", "smtp_from"].map((k) => [k, f[k].value.trim()]));
    attempt(async () => { await put("/api/admin/settings", body); f.smtp_password.value = ""; await refreshInfo(); }, "Saved");
  };
  $("#mail-test", box).onclick = (e) => attempt(() => busy(e.currentTarget, "Sending…", async () => {
    await post("/api/admin/email/test", { to: $("#mail-test-to", box).value.trim() });
    toast("Sent. Check that inbox.");
  }));
}
