# Handoff

Shared between the owner's two computers through git. **Read it at the start of every session and again after a
few hours' gap, and `git fetch` first.** The plans for 0.2 to 0.9, as they were agreed and built, are in
`docs/HANDOFF-ARCHIVE.md`; what each version does is in `CHANGELOG.md`.

## 1. Where things stand (2026-11-12)

- **Released:** 0.9.0 (2026-11-12): Read it later, the `/tv` kitchen screen, the meal plan calendar link, search by
  meaning. `:0.9.0` = `:latest`; `main` and `feature/v0.9` are the same commit.
- **Waiting on the owner:** branch `chore/sweep-2026-11` (pushed, not merged, not released), written up as 0.9.1 in
  `CHANGELOG.md`:
  - security: `x/net` 0.62.0 (a page built for it could tie up the page reader, GO-2026-6860) and `x/crypto` 0.59.0;
  - the page reader only connects to internet addresses (`internal/api/fetchguard.go`, with tests);
  - bundled cloudflared 2026.10.7 (was 2026.9.3);
  - sqlite 1.62.0; two unused functions removed; the signed-in route lists moved from `server.go` to `routes.go`;
  - CI: tests on the newest Go 1.26 patch, and the Node 24 action versions.
- **Checked:** `go build`, `go vet`, `go test` (20 packages), `gofmt`, the JS check (77 files), a CSS rebuild
  (identical), the vulnerability scan (clean), and a dump of all 175 routes with their middleware counts, identical
  before and after the split.
- **Also looked at (2026-11-12): escaping in the page code.** Every value the scripts write into HTML was listed
  (about 1,460 places) and sorted: wrapped in `esc()`, a number or an id, a constant, or HTML built from escaped
  parts. The plain values were traced by hand and the guest page (no sign-in) was read in full. None was found
  unescaped. No guard test was added: it would need a JavaScript parser inside the Go tests. One small thing left
  alone: a recipe's source link is escaped but its scheme is not checked in the page (the content policy stops a
  `javascript:` link from running).
- **Not checked:** the image. This repository builds it only on `main`, so cloudflared 2026.10.7 is first built
  there (the build runs `cloudflared --version`), and Remote access with the new connector can only be confirmed on
  the server. The image and release jobs have not yet run with the new action versions.
- **Held back on purpose:** `chi` 5.5, `anthropic-sdk-go` 1.84, `x/image` 0.48: no reason to move yet.
- Repo: https://github.com/ZachCurry13/recipebank (public, MIT). Image: `ghcr.io/zachcurry13/recipebank`.
- Decisions made with the user are in Claude's project memory (`recipebank-decisions`): name, public repo,
  Home & Care area, strict allergy mode in 0.1, links and photos only for imports, US + metric, separate
  accounts on the same TrueNAS and AI machines, no kitchen-tablet PINs for now.

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
wanted), Mealie/Tandoor/Paprika imports (to review with the user), Google Cast, (update banner and embedding-model search ranking are built).

## 7. Next immediate steps

0. **To release the sweep as 0.9.1** (only with the user's OK, one command per call): put the date in the
   `## [0.9.1]` heading of `CHANGELOG.md`; `git checkout main`; `git pull --ff-only`;
   `git merge --ff-only chore/sweep-2026-11`; `git push origin main`;
   `gh workflow run docker.yml --ref main -f version=0.9.1`; watch the run; then look at Admin -> Remote access on
   the server and update this file.
1. The user updates TrueNAS to 0.9.0 and tries it: Read it later, the `/tv` kitchen screen (pairing from cook mode),
   the calendar link (Google needs the outside address), and search by meaning (needs an embedding model:
   Admin -> AI). Fixes go out as 0.9.x from a `feature/v0.9.x` or `fix/...` branch off `main`.
2. Not tried for real yet (need a real AI or an https address on a phone): the fridge photo, the dish photo, AI
   substitutes, two-step card reading with a big model, nutrition beyond flour (DEMO_KEY's hourly limit), live camera
   scanning, notifications, voice, the Android share target, and the import formats (need real export files).
3. Maintenance: the November 2026 sweep took sqlite and cloudflared; `anthropic-sdk-go`, `chi` and `x/image` are
   still held. Follow `docs/MAINTENANCE.md`; the weekly audit keeps one issue on GitHub.
4. The user's AI was Ollama with qwen2.5:3b / qwen2.5vl:3b (too small for handwriting); Gemini's free tier
   (OpenAI-compatible, a Flash model) or a bigger Ollama model reads cards better.
5. Ask the user what comes next (0.10 ideas: none chosen yet).
