# Handoff

## 1. Where things stand

- Branch `feature/v0.1` holds the 0.1 MVP; `main` has only the first commit. Nothing released yet.
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

Git Bash with `export PATH="/c/Program Files/nodejs:$PATH"`. Scripted edits: node scripts in the scratchpad,
replacement text passed as a function. `.claude/launch.json` paths use forward slashes (backslashes were
lost once and test data landed in the repo folder).

## 5. Conventions

- Recipes store ingredients and steps as JSON columns; `recipe.Clean()` parses unparsed lines.
- Safety checks are word lists in `internal/safety` (allergens with hidden names and look-alikes,
  packaged foods, known plain foods for strict mode, diets, heat, Home & Care hazards).
- Imports return an unsaved draft plus verdicts; saving is a separate POST.
- Fetches use the honest RecipeBank user agent. Publishers that block apps (403/402) get a "Paste text" hint.

## 6. Not built yet

Pantry, supply closet, shopping list, meal plan (0.2); cookbooks and the to-digitize pile; share from phone;
seasons and feasts; AI collections; smart (semantic) search; kitchen tablet with PINs (only if wanted);
Mealie/Tandoor/Paprika imports (to review with the user); Google Cast; remote access; update banner.

## 7. Next immediate steps

1. Push the docs; make sure CI passes on `feature/v0.1`.
2. With the user's OK: merge to `main` and release 0.1.0 (`gh workflow run docker.yml --ref main -f version=0.1.0`).
3. The user installs on TrueNAS and tests; fix what they find.
