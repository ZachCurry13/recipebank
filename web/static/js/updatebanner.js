// The banner under the header: for parents when a newer RecipeBank is out
// (once per version), and for everyone when the server was updated while
// this page stayed open ("Reload").
import { get } from "./api.js";
import { $ } from "./ui.js";

const DISMISS_KEY = "recipebank.dismissedUpdate";

function banner(text, action, onDismiss) {
  $("#update-text").textContent = text;
  $("#update-action").innerHTML = action;
  $("#update-banner").classList.remove("hidden");
  $("#update-dismiss").onclick = () => {
    onDismiss?.();
    $("#update-banner").classList.add("hidden");
  };
}

export async function checkForUpdates() {
  let data;
  try {
    data = await get("/api/updates");
  } catch {
    return; // best effort
  }
  const st = data.status;
  let dismissed = null;
  try {
    dismissed = localStorage.getItem(DISMISS_KEY);
  } catch { /* private mode */ }
  if (!st?.update_available || dismissed === st.latest) return;
  banner(`RecipeBank ${st.latest.replace(/^v/, "")} is out (you have ${st.current}).`,
    `<a href="#/whatsnew" class="underline">See what's new</a>`, () => {
      try {
        localStorage.setItem(DISMISS_KEY, st.latest);
      } catch { /* private mode: hidden for now only */ }
    });
}

// watchServerVersion: each time the app comes back to the front, ask which
// version the server runs; a different one means this page is out of date.
export function watchServerVersion(loaded) {
  document.addEventListener("visibilitychange", async () => {
    if (document.visibilityState !== "visible") return;
    try {
      const me = await get("/api/me");
      if (!me.version || me.version === loaded) return;
      banner("RecipeBank was updated.", `<button type="button" id="update-reload" class="btn-secondary min-h-0 px-3 py-1 text-sm">Reload</button>`);
      $("#update-reload").onclick = () => location.reload();
    } catch { /* offline or signed out */ }
  });
}
