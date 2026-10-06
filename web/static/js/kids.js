// "Kids can help": what each step needs a grown-up for (worked out on the
// server from word lists), shown in cook mode while a kid is helping.
const NEEDS = { knife: "🔪 a knife", sharp: "⚙️ a sharp tool", stove: "🍳 the stove", oven: "♨️ the oven", hot: "🔥 hot things" };
const KEY = "rb:kids";

export function kidsOn() {
  try { return localStorage.getItem(KEY) === "1"; } catch { return false; }
}

export function setKids(on) {
  try { localStorage.setItem(KEY, on ? "1" : "0"); } catch { /* private mode */ }
}

// stepHelp says who can do a step: a kid, or a grown-up because of what it needs.
export function stepHelp(needs) {
  if (!needs) return "";
  return needs.length ? `<p class="box-caution">🧑 Grown-up step: ${needs.map((n) => NEEDS[n] || n).join(" · ")}</p>`
    : `<p class="box-info">🧒 A kid can do this step</p>`;
}

export const kidsChip = (yes) => (yes ? `<span class="chip-ok" title="Some steps a kid can do on their own">🧒 Kids can help</span>` : "");
