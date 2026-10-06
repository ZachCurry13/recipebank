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
0b. [x] **Better photo reading** (from the user's test card on 2026-10-06: a handwritten two-column card came
   back with every amount dropped, 6 of 20 ingredients missing from the right column, "red pepper flakes" read as
   "bell pepper flakes", and invented "Filling"/"Topping" sections copied from the prompt's examples; the photo was
   also uploaded rotated). Build first, in this order:
   1. DONE Photos the right way up: decode through an <img> (honours the phone's rotation info) and ⟳ buttons.
   2. DONE (llm/cardread.go, api readPhotos + organize) Smarter reading: no example sections in the prompt; "two columns: left column first, then right"; keep every
      amount and abbreviation; read in two steps (vision copies the card as plain text, then a text step
      organizes it); record which model read each card.
   3. DONE (safety/crosscheck.go, POST /recipes/{id}/card-checked, js/cardcheck.js; also: steps naming an
      allergen no line has → "not sure"; Staple no longer counts bell pepper/garlic salt) Automatic cross-check (no AI): foods the steps mention that the list lacks, lines without amounts, odd
      amounts; while unchecked, verdicts for allergic/severe people say "not sure"; "Checked against the card" clears it.
   4. DONE (POST /import/photo/again keeps nothing; js/readagain.js compares, ⟳ turns, Keep) Read again: re-read the saved photos (optionally another model), show line-by-line changes, keep either.
   5. DONE (reading_hints table, recipe.Misreads, api/hints_handlers.go, js/readinghints.js; learned only on a
      photo recipe's first save or while it still needs checking) Teach it: corrections become short reading hints (abbreviations, misreads) sent with the next cards;
      listed and editable under Admin → AI.
0c. [x] **Substitute** (the user's name for it; was "Missing something?"): recipe page → "↔ Substitute", tap
   what you don't have → ideas from the substitution table, the allergy swaps and (on request) the AI, only
   ones that suit everyone eating (OKForAll), pantry items first ("✓ You have it"), "+ List" adds to shopping.
   POST /api/recipes/{id}/substitute, js/substitute.js. The AI sees only the title, the line and the pantry.
0d. [x] **Scan confirmation** (the user asked 2026-10-06): a scanned product shows "Is this what you scanned?" with
   the package photo (fetched by the server from the database's own image host, sent as a data: URL because of
   the CSP), name, brand, size and the digits; something already in the house asks "Is this it?" before +1.
1. [x] **What can I make?** (#/make, js/make.js + makeresults.js, POST /api/make and /api/make/photo; in More and on
   Tonight; "Substitute" per missing item uses the ticked foods too; "kindWords" lets "cheddar" cover "cheddar cheese") Tick what's on hand (pantry items pre-ticked, extras typed) or photograph the
   fridge/pantry (vision AI lists foods; the person confirms). Recipes ranked by coverage; **food near its
   use-by date first ("use it up")**; what's missing, with swaps from the table and **AI swap ideas from what's
   on hand, checked against everyone's rules**; "only what I have"; "add missing to shopping".
2. [x] (Add → "From a dish", js/dish.js, POST /api/import/dish and /dish/draft, source_kind "ai") **Dish photo → ideas:** vision AI describes a meal; matching recipes already in RecipeBank, plus an
   AI-drafted recipe (labelled a best guess) through the normal draft and checks.
3. [ ] **Photo double-check:** (folded into 0b) a second AI pass on card/page photos fixes spelling and flags odd amounts
   ("1 cup salt" in cookies) as lines to check; never changes amounts silently.
4. [x] (cooks + cook_thumbs tables, js/cooked.js, /api/recipes/{id}/cooks, likeTier in Tonight ideas) **Cooked it + 👍/👎:** log a cook with each person's thumbs; Tonight's ideas and suggestions favor what
   the family liked; the recipe page shows the history.
5. [x] (internal/nutrition, nutrition_foods cache, GET /api/recipes/{id}/nutrition, js/nutrition.js; tested live with
   DEMO_KEY: dataType must be repeated params, search ranks badly so plainNames + score pick the food; 2 requests
   per new food) **Nutrition estimates:** per serving from USDA FoodData Central (a free key the user signs up for,
   Admin → Nutrition), labelled an estimate; cached per food.
6. [x] (manifest share_target → GET /share redirect; 🖨 Print + @media print in tailwind.input.css; #/cookbook,
   GET /api/cookbook, js/cookbook.js) **Share + print:** share target (Android), print layout + Print button, family cookbook to print/PDF.
6b. [x] **First-time guide** (chosen 2026-10-06: both): a "Getting started" checklist for the admin on Tonight that
   ticks itself off (admin account, family + allergies, first recipe, AI or skip, sign-ins for everyone, on the
   phones) with "Hide this guide"; a short welcome for each person on first sign-in (what ✓/✕/⚠ mean, where
   things are, put it on your phone). Both reopen from Me. Built: js/guide.js, js/firstrun.js, /api/admin/guide,
   users.seen_version (existing users migrated to 0.2.0 so they get "What's new" instead of the welcome).
6c. [x] **Update notices + changelog in the app** (the user asked 2026-10-06): internal/updates (from NovelCheck,
   GitHub releases every 6 h, Admin → House settings toggle), banner for parents (js/updatebanner.js), a "Reload"
   banner when the server changed under an open page, a once-per-release push to admins ("updates" kind),
   #/whatsnew (bundled CHANGELOG.md via changelog.go + GitHub notes), "What's new" sheet once per person after an
   update. The 0.3.0 CHANGELOG section must exist before release, or nobody sees notes after updating.
7. [ ] Phone checks, docs, CHANGELOG 0.3.0, release with the user's OK.

0.4 first (chosen 2026-10-06): budget (prices, weekly estimate, cost per recipe), difficulty, planned leftovers,
email a recipe or the list, events (holiday menus, potlucks, guests per meal).

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

From the original notes (checked again 2026-10-06), not built or planned yet: budget (prices per item, a
weekly estimate, cost per recipe), difficulty, planned leftovers ("Monday's roast into Tuesday's tacos"; only
free-text plan entries now), emailing a recipe or the list, events (holiday menus, potlucks, guests per meal),
kid skills (declined for now). Also: "What can I make?" (photo of the fridge or ticked items), dish photo →
recipe ideas, cookbooks and the to-digitize pile, share from phone and printing / family cookbook PDF, kitchen tablet with PINs (only if
wanted), Mealie/Tandoor/Paprika imports (to review with the user), Google Cast, update banner, embedding-model
search ranking.

## 7. Next immediate steps

1. The user updates TrueNAS to 0.2.0 and tests. Not tried for real yet: live camera scanning, notifications and voice (need the https address on a phone),
   the tunnel connecting (needs the user's Cloudflare token), AI suggestions with a real model.
2. Not yet tested for real: reading card photos with an AI (no AI was set up locally).
3. Then plan 0.2 with the user (pantry, supply closet, shopping list, meal plan).
