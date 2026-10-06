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

**0.3 in progress on `feature/v0.3` (2026-10-06):** cleaner ingredient layout; better photo reading done
(photos the right way up, two-step reading, cross-check with "not sure" until checked against the card, Read
the card again, reading hints learned from corrections under Admin → AI); kitchen substitutions started.

## Next
- 0.3 (HANDOFF section 1c): "Missing something?" on the recipe page (cumin → alternatives), then wire up
  "What can I make?", dish photo → ideas, Cooked it + 👍/👎, nutrition, share + print. Ask the user which Ollama
  model reads their photos.
- The user tests 0.2.0 with a real AI; fixes go out as 0.2.x.
- Later (from the user's list): "what can I make?" from a photo of the fridge, dish photo → ideas, kitchen tablet
  (only if wanted), cookbooks and the card pile, share from the phone and printing, other-app imports, Cast to TV.
