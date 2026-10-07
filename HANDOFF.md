# Handoff

## 1. Where things stand

- **0.1.0 released 2026-10-05** (https://github.com/ZachCurry13/recipebank/releases/tag/v0.1.0): image
  `ghcr.io/zachcurry13/recipebank:0.1.0` and `:latest` (amd64 + arm64, public).
- **0.1.1 released 2026-10-05:** separate database, `/photos` and `/cache` folders.
- **0.1.2 released 2026-10-05:** database in `/config` (many TrueNAS setups keep each app
  as `config/<app>/{config,cache}` on a fast pool, photos on a big pool). Card photos are sent to
  the AI at 1280/896/640 px, smaller on each "exceeds the available context size" answer: a home AI
  server with a 4096-token context refused a 2000 px photo (4318 tokens).
- **0.2.0 released 2026-10-06** (pantry, shopping, meal plan, collections, remote access; section 1b).
- **0.3.0 released 2026-10-06** (https://github.com/ZachCurry13/recipebank/releases/tag/v0.3.0): What can I make?,
  Substitute, better card reading (two steps, cross-check, read again, hints), From a dish, We cooked it, nutrition,
  share/print/cookbook, first-time guide, update notices and What's new (section 1c). `:0.3.0` = `:latest`.
- **0.4.0 released 2026-10-06** (https://github.com/ZachCurry13/recipebank/releases/tag/v0.4.0): fixes from a 0.3
  review, difficulty, planned leftovers, email, budget, events (section 1d). `:0.4.0` = `:latest`. The user was away
  and gave the OK to fix bugs and release. Work continues on `feature/v0.5`.
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
7. [x] Phone checks (all 22 pages at 360×760, light and dark), docs, CHANGELOG 0.3.0; released 2026-10-06 with the user's OK.

0.4 first (chosen 2026-10-06): budget (prices, weekly estimate, cost per recipe), difficulty, planned leftovers,
email a recipe or the list, events (holiday menus, potlucks, guests per meal).

Not chosen (2026-10-06): keto diet, cross-contact note, kids can help, kitchen tablet, cookbooks + card pile,
other-app imports, Cast.

## 1d. 0.4 plan (2026-10-06; the user is away and asked to fix bugs, then build and release the next version)

Bugs fixed first (0.3.0 review): ingredient lines "2 x 400g tins", "1 tbsp + 1 tsp", "a pinch of", trailing
"to taste" now a note; metric shows grams for flour/sugar/butter… (js/cupweights.js). Checked: lint (eslint
no-undef), every page at 360×760, bad input on new endpoints, Kitchen list speed with 400 recipes (45 ms).

Build in this order (each with tests, phone checks, a commit):
1. [x] **Difficulty:** recipes.difficulty ('' = worked out from time, steps, ingredients and techniques); chip on
   cards and the recipe page; editor choice; Library filter; "easy" in smart search.
2. [x] **Planned leftovers:** a plan meal can be "Leftovers from" an earlier planned recipe; the earlier meal's
   servings grow to cover it, the shopping list skips the leftovers meal, Tonight says what to reheat.
3. [x] **Email a recipe or the list:** Admin → Email (SMTP, from NovelCheck's delivery/smtp.go, password kept
   secret, test button); ✉️ on the recipe page and the shopping list; one address at a time, rate-limited.
4. [x] **Budget:** price and package size on pantry/supply items; the shopping list estimates each priced line and
   the total against an optional weekly budget (Admin → House settings, currency); recipes show "about $X to
   make" from priced pantry items (share of the package used), saying how many lines were priced.
5. [x] **Events:** holiday menus and potlucks: name, date, who's coming (people and guests), the menu (recipes or
   "store-bought" lines) with who brings what, every dish checked for everyone coming, "everyone has N dishes
   they can eat", and my dishes to the shopping list scaled to the guest count. In More.
6. [x] Docs, CHANGELOG 0.4.0, phone checks (every page at 360×760, no failed requests), release (the user gave the OK on 2026-10-06).

## 1e. After 0.4.0 (2026-10-06, the user away)

Fixed on `feature/v0.5`, not released yet (a 0.4.1 when the user says so): email works right after setup without
a reload; a recipe's cost says "N of M priced" in words (phones show no tooltips); "No prices yet" instead of
"about $0.00" with a weekly budget; the Events list ignores people taken off the Family page; package sizes like
"500g", "1.5 L", "1 dozen"; "as written" when nobody is listed as coming to an event.

0.5 ideas for the user to pick from (nothing chosen yet):
1. **Scan a receipt:** a photo of a store receipt updates pantry amounts and prices (the AI reads it, the person
   ticks what's right). Makes the budget work without typing prices.
2. **Plan my week:** one tap fills the week's dinners from recipes everyone can eat, favouring 👍, using up food
   near its date, within the weekly budget, with leftovers planned; every pick editable.
3. **Back up and move recipes:** download every recipe (with photos) as one file, and import such a file into
   another RecipeBank (for a grandparent's or a friend's install).
4. **Share a recipe with a link:** a read-only page for someone without an account (expires after a week).
5. **Cookbooks and the card pile** (from the first notes): the family's cookbooks by barcode with their recipe
   index, and a "still to photograph" list of cards. Declined for 0.3; worth asking again.
6. **Imports from other recipe apps:** Mealie, Tandoor, Paprika (explain each in plain words first).
7. **Kids can help:** steps marked for kids (stir, measure) and a simple kid view. Declined for 0.3.
8. **Who may send email:** today every signed-in person can email recipes and the list; maybe parents only.

## 1f. 0.5 plan (2026-10-06; the user: "love them all"; one 0.5.0 at the end, released without asking again)

Build in this order, each tested, checked at 360×760 and pushed to `feature/v0.5`:
1. [x] **Who may send email:** Admin → Email "Who can send": parents only (default) or everyone.
2. [x] **Back up and move recipes:** Admin → Back up: download every recipe (Kitchen and Home & Care, photos,
   cooks, collections) as one .zip; import such a file into any RecipeBank (skips recipes it already has).
3. [x] **Share a recipe by link:** a read-only page for someone without an account, expiring after 7 days, which
   parents can stop sharing; no names or allergies on it.
4. [x] **Plan my week:** fills the empty dinners of a week from recipes everyone home can eat: liked first, food
   near its date used up, not repeated, within the weekly budget when prices are known, with leftovers for big
   recipes; shown as a draft to keep or change.
5. [x] **Scan a receipt:** a photo of a store receipt → lines with names, amounts and prices (vision AI); the
   person ticks what's right; matched to pantry items (amount added, price kept) or added as new ones.
6. [x] **Kids can help:** each step marked by what it needs (knife, stove, oven, hot liquid) from word lists;
   a kid view in cook mode shows which steps a kid can do and which need a grown-up; "kids can help" filter.
7. [x] **Cookbooks and the card pile:** cookbooks by barcode (Open Library: title, author, cover), their recipes by
   page (typed, or a photo of the index read by the AI), recipes can say "Cookbook, page N"; "Still to
   photograph": a list of cards and clippings to scan, ticked off when a photo recipe is saved from one.
8. [x] **Imports from other recipe apps:** files exported from Paprika (.paprikarecipes), Mealie and Tandoor
   (their export .zip), read into drafts/recipes with the usual checks; plain-words help for each.
9. [x] Docs, CHANGELOG 0.5.0, phone checks (360×760, light and dark), release 0.5.0.

## 1g. 0.6 plan (2026-10-06; on `feature/v0.6`; one release at the end, ASK the user before releasing)

0.5.0 released 2026-10-06 (`:0.5.0` = `:latest`). The user chose "all together later": nothing ships until
everything below is built and checked.
1. [x] **Desktop menu:** the top bar shows the main pages (more on wider screens) and ☰ More for the rest (`nav.js`).
2. [x] **A second AI for photos** (Admin → AI): photos go there first (cards, receipts, fridge, indexes), e.g. a big
   model on a PC or an online API; when it can't be reached it's skipped for 2 minutes and the main AI reads them.
   Text stays on the main AI unless the photo AI is the only one (`store/photoai.go`, `llm/chain.go`).
3. [x] **Ollama tools** ported from NovelCheck (`internal/ollama` copied; MIT, same author): find Ollama on the
   network, installed models (delete unused), download with progress, use a model for text or photos.
4. [x] **GPU picks:** measure the card, label text models and photo models (best / most powerful / fits / too big).
5. [x] **Speed test:** time each AI on a made-up sample card photo and a text job.
6. [x] **Model updates, presets, health:** daily check for newer Ollama model versions (Update button), one-click
   presets (Gemini, OpenAI, Claude, Ollama, LM Studio), "is it answering", cost for online APIs.
7. [x] **Features on and off** (asked for by the user's wife: the app felt overwhelming): Admin turns features off
   for the whole house (hidden, data kept); each person can simplify their own view on Me: which meals the plan
   shows (all by default; e.g. just dinner) and pages hidden from their menu.
7b. [x] **Allergy exceptions** (the user's sister: no soy or soy lecithin, but soybean oil is fine): Family → each
   allergy → "Can have anyway": offered ones (refined soybean oil, soy lecithin, refined peanut oil) or typed ones
   ("almond"); only that exact food is masked, cold-pressed/roasted oils still count (`safety/exceptions.go`).
7c. [x] **Photos on Android** (released as 0.5.1 on its own): in-page camera + "Choose a photo" (`camera.js`).
8. [x] Docs, CHANGELOG, phone checks (360×760, light and dark; desktop widths for the menu). Released as 0.6.0 on 2026-10-07 (user OK); `:0.6.0` = `:latest`. Work continues on `feature/v0.7`.
   The SW cache is v8 (0.5.1 used v7).

## 1h. 0.7 plan (2026-10-07; on `feature/v0.7`; one 0.7.0 at the end, ASK the user before releasing)

0.6.1 released 2026-10-07 (Gemini thinking/time-out, List models, ingredient lines kept as written).
The user's notes after using 0.6, plan agreed ("Yes, build it"):
1. [x] **Fewer pages:** "Recipes" with Kitchen | Home & Care tabs; "Pantry" with Food | Supplies tabs; Family moves
   under Me (no menu entry); "Add a recipe" near the front by default.
2. [x] **Menu order on Me:** each person can reorder their pages (the first ones go on the phone's bottom bar and the
   computer's top bar).
3. [x] **Recipe → shopping list:** the button sits near the top and opens a list to tick which ingredients to add
   (ticked: what isn't in the pantry); Home & Care recipes too.
4. [x] **🔍 Search everything** from the top bar: recipes (both areas), pantry and supplies, the list, cookbooks'
   recipes, events, collections.
5. [x] **Event guest link:** a link guests open without an account to add their name, their allergies and what
   they're bringing; the menu is checked for them; the host can stop it; it ends after the event.
6. [x] Docs, CHANGELOG, phone checks (360×760, light and dark). [ ] ASK to release 0.7.0.

Undecided (the user: "Not sure yet"): sharing a recipe with "nobody / household / family / everyone on the server"
needs several households on one server (a big split of people, pantry, plan and list by household) or one household
plus relatives with accounts. Ask again after 0.7.

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

1. The user updates TrueNAS to 0.4.0 and tests (also new: an email account under Admin → Email, prices in the
   pantry, events). Not tried for real yet (no AI locally): the fridge photo, the dish photo,
   AI substitutes and two-step card reading with a real model; nutrition beyond flour (DEMO_KEY's hourly limit);
   live camera scanning, notifications, voice and the Android share target (need the https address on a phone).
2. The user's AI was Ollama with qwen2.5:3b / qwen2.5vl:3b (too small for handwriting); Gemini's free tier was
   suggested (OpenAI-compatible, https://generativelanguage.googleapis.com/v1beta/openai, a Flash model).
3. Fix what the user finds as 0.3.x, then plan 0.4 with the user (the five extras chosen 2026-10-06, section 1c).
