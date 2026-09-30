# RecipeBank

A self-hosted recipe bank for the whole family. Every recipe is checked against **each person's allergies, diets, dislikes and heat limit**, and never says "OK" for a real allergy unless every ingredient is accounted for. A separate **Home & Care** area holds home-made cleaners, toothpaste, mouthwash and more, with safety warnings for kids, pets and surfaces.

Runs as one small Docker container (made for TrueNAS SCALE), works on phones, and keeps everything on your own server.

## What it does

- **Add recipes** from a link (most recipe sites include a standard recipe format RecipeBank reads exactly), from **photos** of handwritten cards or cookbook pages (the AI reads them and marks lines it wasn't sure about), from pasted text, or typed in.
- **Who can eat it:** for each person, ✓ OK, ✕ Not for (with the reason and the ingredient), or ⚠ Not sure (check the label). Allergies come in three strengths:
  - **Avoid:** only ingredients that clearly contain it count.
  - **Allergic:** "may contain" and packaged foods are flagged so you check the label.
  - **Severe (strict):** every ingredient must be a plain food RecipeBank knows, or one whose label a parent checked.
  - Hidden names are known (whey and ghee are milk, tahini is sesame, Worcestershire has fish, soy sauce has wheat) and look-alikes aren't flagged (coconut milk, nutmeg, eggplant).
  - US list of 9 major allergens, or the EU's 14.
- **Swaps** that work for everyone eating ("oat milk" is never offered to someone who can't have oats).
- **Diets:** vegetarian, vegan, pescatarian, meatless (Fridays and Lent), gluten-free, dairy-free, nut-free, halal, kosher-style. Plus dislikes and a heat limit (0–5 peppers).
- **Home & Care warnings:** dangerous mixes (bleach with ammonia, vinegar or alcohol; peroxide with vinegar), things kids must not swallow, oils toxic to cats, xylitol for dogs, acids that etch stone, storage and labels.
- **Recipe pages** with a servings scaler and US ⇄ metric, **cook mode** (one big step at a time, timers that ring), "our version" of a recipe that keeps the original, ratings and notes.
- **Accounts** for parents and kids; people without accounts (guests) still get checked.

RecipeBank helps, but it doesn't replace reading labels. When in doubt, ask a doctor, dentist or vet.

## Install

- **TrueNAS SCALE:** follow [docs/TRUENAS.md](docs/TRUENAS.md). Both **Install Custom App** (a form) and **Install via YAML** are covered.
- **Docker Compose:** `docker compose up -d` with [docker-compose.yml](docker-compose.yml) (change the data folder first), then open `http://your-server:8080`.

The image is `ghcr.io/zachcurry13/recipebank` (`:latest` is the newest release).

## Settings from the environment

| Variable | Default | What it does |
|---|---|---|
| `RECIPEBANK_ADDR` | `:8080` | Where it listens |
| `RECIPEBANK_DATA_DIR` | `/data` | The database and photos |
| `RECIPEBANK_TRUST_PROXY` | `true` | Use the visitor's address from a proxy or Cloudflare |
| `RECIPEBANK_SESSION_DAYS` | `30` | How long "Keep me signed in" lasts |
| `RECIPEBANK_ADMIN_USER` / `RECIPEBANK_ADMIN_PASSWORD` | | Optional: create the first admin at start instead of in the browser |

Everything else (the AI, allergen list, units) is set in the app under **Admin**.

## Building from source

Go 1.26+. The web app is plain JavaScript modules and a Tailwind stylesheet that is committed, so `go build ./cmd/recipebank` needs no Node. `make test`, `make css` (needs Node), `make run`.

## License

MIT, © 2026 ZachCurry13. Built from the same core as [NovelCheck](https://github.com/ZachCurry13/novelcheck).
