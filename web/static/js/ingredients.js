// How ingredients and the amounts in steps read: the amount on its own (for
// the bold column), the item, and recipe wording ("to serve", "optional") as
// a quiet note.
import { US, METRIC, toMetric, toUS, usNumber, metricNumber, unitLabel, singular } from "./units.js";

// Phrases that say how a recipe uses something, not what it is.
const TRAILING = [" plus ", " for ", " to serve", " to taste", " to garnish", " as needed", " if needed", ", divided", " divided", " optional"];

// splitName: "lemon wedges to serve" → ["lemon wedges", "to serve"].
function splitName(food) {
  const low = food.toLowerCase();
  let cut = food.length;
  for (const t of TRAILING) {
    const i = low.indexOf(t);
    if (i > 0 && i < cut) cut = i;
  }
  return [food.slice(0, cut).trim().replace(/[,;]$/, ""), food.slice(cut).trim().replace(/^[,;]\s*/, "")];
}

// Spoons stay spoons in metric too (metric recipes use them).
const SPOONS = new Set(["tsp", "tbsp"]);
const needsConvert = (unit, system) => (system === "metric" && US.has(unit) && !SPOONS.has(unit)) || (system === "us" && METRIC.has(unit));
const convert = (q, unit, system) => (!needsConvert(unit, system) ? [q, unit] : system === "metric" ? toMetric(q, unit) : toUS(q, unit));
const number = (q, unit) => (METRIC.has(unit) ? metricNumber(q) : usNumber(q));

// parts is an ingredient as {amount, name, note}, scaled and in the unit system.
export function parts(ing, factor, system) {
  const [name, extra] = splitName((ing.food || ing.line || "").trim());
  const note = [extra, ing.note].filter(Boolean).map((n) => n.replace(/^\((.*)\)$/, "$1")).join(" · ");
  if (ing.qty === undefined || ing.qty === null) return { amount: "", name, note };
  const [q, unit] = convert(ing.qty * factor, ing.unit, system);
  let amount = number(q, unit);
  if (ing.qty_max) amount += "–" + number(convert(ing.qty_max * factor, ing.unit, system)[0], unit);
  const one = !unit && q <= 1 && !ing.qty_max;
  return { amount: `${amount} ${unitLabel(unit, ing.qty_max ? 2 : q)}`.trim(), name: one ? singular(name) : name, note };
}

const UNIT_WORDS = { tablespoon: "tbsp", tablespoons: "tbsp", tbsp: "tbsp", teaspoon: "tsp", teaspoons: "tsp", tsp: "tsp",
  cup: "cup", cups: "cup", ounce: "oz", ounces: "oz", oz: "oz", pound: "lb", pounds: "lb", lb: "lb", lbs: "lb",
  gram: "g", grams: "g", g: "g", kilogram: "kg", kilograms: "kg", kg: "kg", milliliter: "ml", milliliters: "ml",
  millilitre: "ml", millilitres: "ml", ml: "ml", liter: "l", liters: "l", litre: "l", litres: "l" };
const GLYPHS = { "½": 0.5, "¼": 0.25, "¾": 0.75, "⅓": 1 / 3, "⅔": 2 / 3, "⅛": 0.125 };
const AMOUNT_RE = /(\d+\s+\d+\/\d+|\d+\/\d+|\d+(?:[.,]\d+)?\s*[½¼¾⅓⅔⅛]?|[½¼¾⅓⅔⅛])\s*(tablespoons?|tbsp|teaspoons?|tsp|cups?|ounces?|oz|pounds?|lbs?|grams?|g|kilograms?|kg|millilit(?:er|re)s?|ml|lit(?:er|re)s?)\b/gi;

function parseNumber(s) {
  let total = 0;
  for (const part of s.trim().split(/\s+/)) {
    let p = part;
    for (const [g, v] of Object.entries(GLYPHS)) if (p.includes(g)) { total += v; p = p.replace(g, ""); }
    if (!p) continue;
    if (p.includes("/")) { const [a, b] = p.split("/").map(Number); if (b) total += a / b; } else total += Number(p.replace(",", ".")) || 0;
  }
  return total;
}

// stepText rewrites the amounts in a step into the unit system ("Put 100g
// flour" → "Put 3 ½ oz flour"), leaving everything else as written.
export function stepText(text, system) {
  return text.replace(AMOUNT_RE, (whole, num, word) => {
    const unit = UNIT_WORDS[word.toLowerCase()];
    if (!unit || !needsConvert(unit, system)) return whole;
    const [q, u] = convert(parseNumber(num), unit, system);
    // Non-breaking spaces keep "1 ¼ cups" on one line.
    return q > 0 ? `${number(q, u)} ${unitLabel(u, q)}`.replace(/ /g, "\u00a0") : whole;
  });
}
