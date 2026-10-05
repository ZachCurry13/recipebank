# RecipeBank

A self-hosted family recipe bank (Go + vanilla JS, one Docker image, TrueNAS app), built like
NovelCheck (`C:\novelcheck`). Kitchen recipes are checked per person against allergies, diets,
dislikes and heat; Home & Care recipes (cleaners, toothpaste, mouthwash) get safety warnings.

## Layout
- `cmd/recipebank/` entry point · `internal/config` env vars (`RECIPEBANK_*`)
- `internal/api/` HTTP routes and handlers · `internal/store/` SQLite access · `internal/db/` schema + migrations
- `internal/recipe/` recipe type, ingredient parser, web page (JSON-LD) and pasted-text readers
- `internal/safety/` allergens, diets, strict mode, swaps, Home & Care hazards (no AI)
- `internal/llm/` AI client (OpenAI-compatible, Anthropic), photo reading, recipe prompt
- `internal/files/` where files live (`/data` database, `/photos`, `/cache` previews), moving photos, previews
- `web/static/` embedded web app (ES modules in `js/`, compiled `css/app.css`) · `web/tailwind.input.css`
- `docs/` TrueNAS guide, STATUS · `concepts/` local-only notes (git-excluded, never commit)

## Run, build, test (Git Bash)
- `export PATH="/c/Program Files/nodejs:$PATH"` first
- Test: `go test ./...` · `GOOS=linux go vet ./...` · `gofmt -l .` must print nothing
- JS check: `for f in web/static/js/*.js; do node --input-type=module --check < $f || echo BAD $f; done`
- CSS: `npx tailwindcss@3 -c tailwind.config.js -i web/tailwind.input.css -o web/static/css/app.css --minify`
- Run locally: `.claude/launch.json` config `recipebank-local` (port 18110, data in the session scratchpad;
  use forward slashes in paths). Web files are embedded: restart after edits.

## Hard rules
- Safety: allergy verdicts come from word lists, never from the AI alone; unknown is never "OK" for a
  real allergy. Every change to `internal/safety` needs tests proving it.
- Releases only with the user's OK. Work on `feature/vX.Y`; `main` publishes `:main`, a release `:X.Y.Z`.
- Privacy: no personal setup (IPs, ports, pool names, family names) and no personal email in commits
  (use the configured noreply identity). Neutral wording in the app and docs.
- CSP: no inline scripts or `style=""` attributes (set styles from JS). Non-GET requests send `X-RecipeBank: 1`.
- SQLite has one connection: never query inside a rows loop or call the store inside a transaction.
- Outside pages are fetched with the honest RecipeBank user agent; sites that block apps get "Paste text".
- Phones first: check touched pages at 360×760 with long names (`scrollWidth` = 360).
- Keep files under ~300 lines; new web files go in the service-worker cache name bump (`web/static/sw.js`).
