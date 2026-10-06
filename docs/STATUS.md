# Status

**Where it stands (2026-10-05):** 0.1.0 released (image `ghcr.io/zachcurry13/recipebank:0.1.0` / `:latest`).
0.1.1 released (separate folders), 0.1.2 released (`/config` folder; card photos fit 4096-token home AI models).
**0.2.0 released 2026-10-06:** pantry and supplies, shopping list, meal plan + Tonight, collections and seasons,
smart search, cook-mode voice, remote access, phone notifications (HANDOFF section 1b).
- Backend: accounts and first-run setup, people with allergies/diets/dislikes/sensitivities/heat,
  recipes (Kitchen and Home & Care), imports from links (JSON-LD, else AI), photos (AI) and pasted
  text (AI, or headings without it), per-person verdicts with strict mode, swaps, label checks,
  Home & Care hazards (mixes, kids, pets, surfaces, storage). Go tests pass.
- Web app: Library, recipe page, cook mode, Add, editor, Family, Admin, Me; checked at 360×760.
- Known limits: some publishers block apps (Allrecipes, Simply Recipes, Serious Eats), so use Paste text;
  the screen stays on in cook mode only over https.

- Install docs done: `docs/TRUENAS.md` (Custom App form and YAML), `docker-compose.yml`, README,
  CHANGELOG 0.1.0, HANDOFF. CI tests pass on `feature/v0.1`.

**0.3.0 released 2026-10-06:** cleaner ingredient layout; better photo reading done
(photos the right way up, two-step reading, cross-check with "not sure" until checked against the card, Read
the card again, reading hints learned from corrections under Admin → AI); "Substitute" on the recipe page; barcode
scans confirmed with the package photo; "What can I make?" (ticked/typed foods or a fridge photo); a first-time
welcome and the admin's "Getting started"; update notices, a Reload banner and What's new in the app; From a dish;
We cooked it with 👍/👎; nutrition estimates (USDA FoodData Central); share target, printing and the family cookbook.
CHANGELOG 0.3.0 written; every page checked at 360×760 (light and dark).

## Next
- 0.4.0 released 2026-10-06: bug fixes from a review of 0.3 (ingredient lines, metric grams), difficulty, planned
  leftovers, email (SMTP), budget (pantry prices, list and recipe costs, weekly budget), events (menus, who
  brings what, checks for everyone coming).
- 0.5.0 released 2026-10-06 (built while the user was away; release OK given in advance): Plan my week, scan a receipt,
  kids can help (word lists), Bookshelf (cookbooks by ISBN via Open Library, recipes by page, card pile), imports
  from Paprika/Mealie/Tandoor, share links, back up and move recipes, who may email; fixes incl. dish drafts that
  couldn't be saved (recipes table rebuilt once on upgrade). Checked at 360×760, light and dark.
- Next: the user tries 0.5 with real exports, receipts and cookbooks (the import formats were built from the apps'
  documented shapes, not real files); fixes go out as 0.5.x. On feature/v0.6: the desktop top bar
  with a More menu (unreleased). Asked: a separate AI for photos (HANDOFF 1g).
- The user tests 0.2.0 with a real AI; fixes go out as 0.2.x.
- Later (from the user's list): "what can I make?" from a photo of the fridge, dish photo → ideas, kitchen tablet
  (only if wanted), cookbooks and the card pile, share from the phone and printing, other-app imports, Cast to TV.
