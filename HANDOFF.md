# Handoff

## 1. Where things stand

- **0.1.0 released 2026-10-05** (https://github.com/ZachCurry13/recipebank/releases/tag/v0.1.0): image
  `ghcr.io/zachcurry13/recipebank:0.1.0` and `:latest` (amd64 + arm64, public).
- **0.1.1 released 2026-10-05:** separate database, `/photos` and `/cache` folders.
- **0.1.2 released 2026-10-05:** database in `/config` (many TrueNAS setups keep each app
  as `config/<app>/{config,cache}` on a fast pool, photos on a big pool). Card photos are sent to
  the AI at 1280/896/640 px, smaller on each "exceeds the available context size" answer: a home AI
  server with a 4096-token context refused a 2000 px photo (4318 tokens).
- Repo: https://github.com/ZachCurry13/recipebank (public, MIT). Image: `ghcr.io/zachcurry13/recipebank`.
- Decisions made with the user are in Claude's project memory (`recipebank-decisions`): name, public repo,
  Home & Care area, strict allergy mode in 0.1, links and photos only for imports, US + metric, separate
  accounts on the same TrueNAS and AI machines, no kitchen-tablet PINs for now.

## 1b. 0.2.0 plan (agreed 2026-10-05; one big release at the end, with the user's OK)

Work on `feature/v0.2`, commit and push after each part. Status per part:
1. [x] Pantry + supply closet (one stock system, two areas): typed or barcode (photo; live scan on https),
   Open Food Facts lookup (name, allergens, traces), amounts, location, use-by, low-stock level; pantry
   labels help clear "Not sure" ingredients (a parent confirms).
2. [x] Shopping list: from recipes or a plan week, amounts combined, pantry items under "You probably
   have", low supplies added, store sections, tick, shared, works offline.
3. [x] Meal plan (week, who's home, guests) + Tonight page as the start page.
4. [x] Collections (by hand or AI-described, parent reviews) + seasons and feasts (NovelCheck's
   `internal/seasons` Easter/Advent).
5. [x] Smart search: a question's rules (diets, time, everyone) are read without the AI; the AI picks by meaning (or words rank them); rules have the last word. Embedding-model ranking not built (the AI pick covers meaning).
6. [x] Cook mode voice: read steps aloud; "next/back/repeat" where supported.
7. [x] Remote access: Cloudflare tunnel from NovelCheck (`internal/tunnel`, cloudflared in the image).
8. [x] Phone notifications: Web Push from NovelCheck (`internal/push`): timers, use-by, low stock, tonight.
9. [x] Phone checks (all 18 pages at 360 px, light and dark), docs (TRUENAS, REMOTE_ACCESS, CHANGELOG, README).
   [x] **Released 0.2.0 on 2026-10-06** (https://github.com/ZachCurry13/recipebank/releases/tag/v0.2.0).

## 1c. 0.3 plan (chosen 2026-10-06; the user said to build it now and test 0.2.0 later)

Checked against the original notes (concepts/recipebank_features.md, recipes.md) on 2026-10-06; the user added
the catch-ups and new ideas below. Build in this order:
0. [ ] **Cleaner recipe display** (the user found it sloppy): amounts in a bold column, the item beside it,
   recipe wording ("to serve", "plus extra for frying", "optional") as a small grey note, problems as short
   tags, tap a row to tick it off; amounts in the steps converted (and scaled) to match the list.
1. [ ] **What can I make?** Tick what's on hand (pantry items pre-ticked, extras typed) or photograph the
   fridge/pantry (vision AI lists foods; the person confirms). Recipes ranked by coverage; **food near its
   use-by date first ("use it up")**; what's missing, with swaps from the table and **AI swap ideas from what's
   on hand, checked against everyone's rules**; "only what I have"; "add missing to shopping".
2. [ ] **Dish photo → ideas:** vision AI describes a meal; matching recipes already in RecipeBank, plus an
   AI-drafted recipe (labelled a best guess) through the normal draft and checks.
3. [ ] **Photo double-check:** a second AI pass on card/page photos fixes spelling and flags odd amounts
   ("1 cup salt" in cookies) as lines to check; never changes amounts silently.
4. [ ] **Cooked it + 👍/👎:** log a cook with each person's thumbs; Tonight's ideas and suggestions favor what
   the family liked; the recipe page shows the history.
5. [ ] **Nutrition estimates:** per serving from USDA FoodData Central (a free key the user signs up for,
   Admin → Nutrition), labelled an estimate; cached per food.
6. [ ] **Share + print:** share target (Android), print layout + Print button, family cookbook to print/PDF.
7. [ ] Phone checks, docs, CHANGELOG 0.3.0, release with the user's OK.

Not chosen (2026-10-06): keto diet, cross-contact note, kids can help, kitchen tablet, cookbooks + card pile,
other-app imports, Cast.

## 2. Rules

See `CLAUDE.md` (hard rules) and the local-only `concepts/working-rules.md` (how we work: plan first,
releases only with the user's OK, phone checks, privacy, docs with every change).

## 3. Checking changes

`go test ./...`, `GOOS=linux go vet ./...`, `gofmt -l .`, the JS syntax check and a CSS rebuild
(commands in `CLAUDE.md`). Then run `recipebank-local` from `.claude/launch.json` and check touched pages
at 360×760. Test logins for the local app live in the session scratchpad, never in the repo or chat.

## 4. Working on Windows

Git Bash with `export PATH="/c/Program Files/nodejs:$PATH"`. A Go build once crashed with
"unsafe.Slice: len out of range"; running it again worked (a toolchain hiccup, not the code). Scripted edits: node scripts in the scratchpad,
replacement text passed as a function. Backslashes get lost in scripted edits (`\s` became `s` in a regex and a Go string lost one `\`): write code with backslashes using the Edit tool, then grep the result. `.claude/launch.json` paths use forward slashes (backslashes were
lost once and test data landed in the repo folder).

## 5. Conventions

- Recipes store ingredients and steps as JSON columns; `recipe.Clean()` parses unparsed lines.
- Safety checks are word lists in `internal/safety` (allergens with hidden names and look-alikes,
  packaged foods, known plain foods for strict mode, diets, heat, Home & Care hazards).
- Imports return an unsaved draft plus verdicts; saving is a separate POST.
- Folders (`internal/files`): database in `/data` if it already holds `recipebank.db`, else `/config` when
  mounted, else `/data`. `/photos` and `/cache` are used when they exist (mounted datasets), else
  `<db folder>/photos` and `/cache`; env vars override. Photos left in `<data>/photos` are moved at
  startup (copy, check size, then remove) and still served from there until moved. Previews: `?w=480|1200`.
- Fetches use the honest RecipeBank user agent. Publishers that block apps (403/402) get a "Paste text" hint.

## 6. Not built yet

"What can I make?" (photo of the fridge or ticked items), dish photo → recipe ideas, cookbooks and the
to-digitize pile, share from phone and printing / family cookbook PDF, kitchen tablet with PINs (only if
wanted), Mealie/Tandoor/Paprika imports (to review with the user), Google Cast, update banner, embedding-model
search ranking.

## 7. Next immediate steps

1. The user updates TrueNAS to 0.2.0 and tests. Not tried for real yet: live camera scanning, notifications and voice (need the https address on a phone),
   the tunnel connecting (needs the user's Cloudflare token), AI suggestions with a real model.
2. Not yet tested for real: reading card photos with an AI (no AI was set up locally).
3. Then plan 0.2 with the user (pantry, supply closet, shopping list, meal plan).
