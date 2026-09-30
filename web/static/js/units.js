// Scaling amounts and switching between US and metric units.

const ML = { tsp: 4.929, tbsp: 14.787, cup: 236.59, "fl oz": 29.574, pint: 473.18, quart: 946.35, gallon: 3785.4, ml: 1, l: 1000 };
const G = { oz: 28.35, lb: 453.59, g: 1, kg: 1000 };
const US = new Set(["tsp", "tbsp", "cup", "fl oz", "pint", "quart", "gallon", "oz", "lb"]);
const METRIC = new Set(["ml", "l", "g", "kg"]);
// Count words shown in the plural for more than one.
const PLURAL = { clove: "cloves", can: "cans", stick: "sticks", package: "packages", slice: "slices", bunch: "bunches",
  sprig: "sprigs", head: "heads", jar: "jars", bottle: "bottles", drop: "drops", pinch: "pinches", dash: "dashes",
  cup: "cups", pint: "pints", quart: "quarts", gallon: "gallons", handful: "handfuls" };

const FRACS = [[0, ""], [1 / 8, "⅛"], [1 / 4, "¼"], [1 / 3, "⅓"], [3 / 8, "⅜"], [1 / 2, "½"], [5 / 8, "⅝"], [2 / 3, "⅔"], [3 / 4, "¾"], [7 / 8, "⅞"], [1, ""]];

// usNumber writes 1.5 as "1 ½" (to the nearest eighth or third).
function usNumber(n) {
  if (n >= 10) return String(Math.round(n));
  let whole = Math.floor(n);
  const rest = n - whole;
  let best = FRACS[0];
  for (const f of FRACS) if (Math.abs(rest - f[0]) < Math.abs(rest - best[0])) best = f;
  if (best[0] === 1) {
    whole += 1;
    best = FRACS[0];
  }
  if (!whole && !best[1]) return String(Math.round(n * 100) / 100);
  return [whole || "", best[1]].filter(Boolean).join(" ");
}

function metricNumber(n) {
  if (n >= 100) return String(Math.round(n / 5) * 5);
  if (n >= 10) return String(Math.round(n));
  return String(Math.round(n * 10) / 10);
}

// toMetric / toUS convert an amount; unchanged if it's already there or a count.
function toMetric(q, unit) {
  if (ML[unit] && !METRIC.has(unit)) {
    const ml = q * ML[unit];
    return ml >= 1000 ? [ml / 1000, "l"] : [ml, "ml"];
  }
  if (G[unit] && !METRIC.has(unit)) {
    const g = q * G[unit];
    return g >= 1000 ? [g / 1000, "kg"] : [g, "g"];
  }
  return [q, unit];
}

function toUS(q, unit) {
  if (unit === "ml" || unit === "l") {
    const ml = q * ML[unit];
    if (ml < 14) return [ml / ML.tsp, "tsp"];
    if (ml < 59) return [ml / ML.tbsp, "tbsp"];
    return [ml / ML.cup, "cup"];
  }
  if (unit === "g" || unit === "kg") {
    const g = q * G[unit];
    return g < 454 ? [g / G.oz, "oz"] : [g / G.lb, "lb"];
  }
  return [q, unit];
}

function unitLabel(unit, q) {
  if (!unit) return "";
  return q > 1 && PLURAL[unit] ? PLURAL[unit] : unit;
}

// amountLine is how an ingredient reads at a scale (factor) and in a unit
// system ("us" or "metric"). Unchanged lines keep the author's wording.
export function amountLine(ing, factor, system) {
  if (ing.qty === undefined || ing.qty === null) return ing.line;
  const convert = (system === "metric" && US.has(ing.unit)) || (system === "us" && METRIC.has(ing.unit));
  if (factor === 1 && !convert) return ing.line;
  const conv = (q) => (convert ? (system === "metric" ? toMetric(q, ing.unit) : toUS(q, ing.unit)) : [q, ing.unit]);
  const [q, unit] = conv(ing.qty * factor);
  const metric = METRIC.has(unit);
  const num = metric ? metricNumber : usNumber;
  let amount = num(q);
  if (ing.qty_max) amount += "–" + num(conv(ing.qty_max * factor)[0]);
  const note = ing.note ? `, ${ing.note}` : "";
  return `${amount} ${unitLabel(unit, q)} ${ing.food}${note}`.replace(/\s+/g, " ").trim();
}

// temps rewrites oven temperatures in a step for the unit system.
export function temps(text, system) {
  if (system === "metric") {
    return text.replace(/(\d{3})\s*°?\s*(?:degrees\s*)?F\b/gi, (_, f) => `${Math.round(((f - 32) * 5) / 9 / 5) * 5}°C`);
  }
  return text.replace(/(\d{2,3})\s*°\s*C\b/g, (_, c) => `${Math.round(((c * 9) / 5 + 32) / 5) * 5}°F`);
}

// timersIn finds cooking times in a step ("bake 25-30 minutes" → 25 min).
export function timersIn(text) {
  const out = [];
  const re = /(\d+(?:\.\d+)?)(?:\s*(?:-|–|to)\s*(\d+(?:\.\d+)?))?\s*(hours?|hrs?|minutes?|mins?|seconds?|secs?)\b/gi;
  for (const m of text.matchAll(re)) {
    const n = parseFloat(m[1]);
    const u = m[3].toLowerCase();
    const secs = u.startsWith("h") ? n * 3600 : u.startsWith("m") ? n * 60 : n;
    if (secs >= 10 && secs <= 24 * 3600) out.push({ secs, label: m[0] });
  }
  return out;
}
