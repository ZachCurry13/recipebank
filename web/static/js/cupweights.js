// Grams in a US cup of foods that metric recipes weigh rather than measure
// (flour, sugar, butter…). The first name a food contains wins, so liquids
// and look-alikes that must stay in ml come first with 0.
const CUP_GRAMS = [
  ["buttermilk", 0], ["oat milk", 0], ["almond milk", 0], ["rice milk", 0], ["rice vinegar", 0], ["rice wine", 0],
  ["sugar snap", 0], ["cream of", 0], ["sour cream", 0], ["milk", 0], ["juice", 0], ["syrup", 0], ["oil", 0],
  ["almond flour", 96], ["coconut flour", 112], ["whole wheat flour", 120], ["bread flour", 127], ["cake flour", 114], ["flour", 125],
  ["powdered sugar", 120], ["icing sugar", 120], ["confectioners", 120], ["brown sugar", 213], ["sugar", 200],
  ["peanut butter", 258], ["butter", 227], ["shortening", 205], ["cream cheese", 232],
  ["rolled oats", 90], ["oats", 90], ["cocoa", 85], ["cornstarch", 128], ["corn starch", 128], ["cornmeal", 138],
  ["chocolate chips", 170], ["raisins", 150], ["shredded coconut", 85], ["breadcrumbs", 108], ["bread crumbs", 108],
  ["panko", 50], ["walnuts", 120], ["pecans", 110], ["almonds", 140], ["rice", 185], ["quinoa", 170],
  ["shredded", 113], ["grated parmesan", 100], ["parmesan", 100], ["honey", 340],
];

export function gramsPerCup(food) {
  const f = String(food || "").toLowerCase();
  for (const [name, g] of CUP_GRAMS) if (f.includes(name)) return g;
  return 0;
}

// weigh turns cups (or sticks of butter) into grams in metric; null when the
// food isn't one that's weighed.
export function weigh(q, unit, food, system) {
  if (system !== "metric" || (unit !== "cup" && unit !== "stick")) return null;
  const g = gramsPerCup(food);
  if (!g || (unit === "stick" && g !== 227)) return null; // a stick is butter: half a cup
  const grams = q * (unit === "stick" ? 0.5 : 1) * g;
  return grams >= 1000 ? [grams / 1000, "kg"] : [grams, "g"];
}
