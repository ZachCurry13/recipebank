# Status

**Where it stands (2026-10-05):** 0.1.0 released (image `ghcr.io/zachcurry13/recipebank:0.1.0` / `:latest`).
0.1.1 released (separate folders). 0.1.2 ready on `feature/v0.1`: database in `/config` like the user's other apps;
card photos shrink to fit home AI models with a 4096-token context.
- Backend: accounts and first-run setup, people with allergies/diets/dislikes/sensitivities/heat,
  recipes (Kitchen and Home & Care), imports from links (JSON-LD, else AI), photos (AI) and pasted
  text (AI, or headings without it), per-person verdicts with strict mode, swaps, label checks,
  Home & Care hazards (mixes, kids, pets, surfaces, storage). Go tests pass.
- Web app: Library, recipe page, cook mode, Add, editor, Family, Admin, Me; checked at 360×760.
- Known limits: some publishers block apps (Allrecipes, Simply Recipes, Serious Eats), so use Paste text;
  the screen stays on in cook mode only over https.

- Install docs done: `docs/TRUENAS.md` (Custom App form and YAML), `docker-compose.yml`, README,
  CHANGELOG 0.1.0, HANDOFF. CI tests pass on `feature/v0.1`.

## Next
- Release 0.1.2 (with the user's OK); the user retests reading card photos on their home AI.
- 0.2: pantry, supply closet, shopping list, meal plan; kitchen tablet only if wanted.
