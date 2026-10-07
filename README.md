# RecipeBank

A self-hosted recipe bank for the whole family. Every recipe is checked against **each person's allergies, diets, dislikes and heat limit**, and never says "OK" for a real allergy unless every ingredient is accounted for. A separate **Home & Care** area holds home-made cleaners, toothpaste, mouthwash and more, with safety warnings for kids, pets and surfaces.

Runs as one small Docker container (made for TrueNAS SCALE), works on phones, and keeps everything on your own server.

## What it does

- **Add recipes** from a link (most recipe sites include a standard recipe format RecipeBank reads exactly), from **photos** of handwritten cards or cookbook pages (the AI reads them and marks lines it wasn't sure about), from pasted text, typed in, or brought over from **Paprika, Mealie or Tandoor**.
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
- **Tonight and the meal plan:** a week of meals with who's home (guests too), every meal checked for the people eating, and Tonight as the first page.
- **Pantry, Supplies and the shopping list:** scan barcodes (Open Food Facts fills in the label), use-by dates and running-low levels; one shared shopping list that combines amounts, skips what's in the house, and works in the store without signal.
- **Collections and seasons:** themed shelves (the AI can suggest recipes; RecipeBank's rules check them), plus seasonal shelves and feast days from Advent baking to Lent Fridays and summer grilling.
- **Smart search:** ask in plain words ("quick dairy-free dinners everyone can eat").
- **What can I make?** from what's ticked, typed or seen in a photo of the fridge, and **Substitute** for an ingredient you don't have, only with stand-ins that suit everyone eating.
- **Card photos read carefully:** a word-for-word copy first, a check for missed or misread lines (a recipe stays "not sure" for allergies until someone checks it against the card), **read again**, and reading hints learned from the family's corrections.
- **From a dish:** a photo of a meal finds similar recipes, or the AI drafts its best guess.
- **We cooked it** with each person's 👍/👎, **nutrition estimates** from USDA FoodData Central, **printing** and a **family cookbook** to print or save as a PDF, and sharing links from an Android phone.
- **Events** (holiday dinners and potlucks, every dish checked for everyone coming), a **budget** from pantry prices, **planned leftovers**, **email** a recipe or the list, and how hard each recipe is.
- **Plan my week** in one tap (what everyone can eat, liked first, food near its date used up, within the budget), **scan a receipt** into the pantry, and **kids can help** (steps marked by what needs a grown-up).
- **Bookshelf:** cookbooks by barcode with their recipes by page (searchable), and a list of the cards still to photograph. **Share a recipe with a link** (read-only, 7 days) and **back up** every recipe as one file.
- **Cook mode voice**, **phone notifications** (timers, use soon, running low, tonight's dinner) and **remote access** through a built-in Cloudflare Tunnel ([docs/REMOTE_ACCESS.md](docs/REMOTE_ACCESS.md)).
- **AI your way:** quick setup for Ollama, LM Studio, Gemini, OpenAI or Claude; find Ollama, download models and see which fit your graphics card, all from Admin; a **second AI just for photos** (a bigger model on another computer, or an online one) with the main AI as the fallback; an AI check-up with a speed test.
- **Your menu, your order**, with 🔍 **search everything** (recipes, the pantry, the list, cookbooks, events, people, settings), and an ingredient picker for the shopping list on every recipe.
- **Event guest links:** guests say they're coming with their allergies and what they bring; the menu is checked for them.
- **Show only what you use:** turn features off for the house, and let each person keep it simple (a plan with only dinner, fewer pages). Allergies can list what someone **can have anyway** (highly refined soybean oil for a soy allergy).
- **Accounts** for parents and kids; people without accounts (guests) still get checked. A short welcome for new people, a setup checklist for the admin, update notices and **What's new** in the app.

RecipeBank helps, but it doesn't replace reading labels. When in doubt, ask a doctor, dentist or vet.

## Install

- **TrueNAS SCALE:** follow [docs/TRUENAS.md](docs/TRUENAS.md). Both **Install Custom App** (a form) and **Install via YAML** are covered.
  The database (`/config`), a cache of smaller copies (`/cache`) and the photos (`/photos`) each get their own dataset, so the first two can sit with your other apps' settings on a fast pool and the photos next to your other photos.
- **Docker Compose:** `docker compose up -d` with [docker-compose.yml](docker-compose.yml) (change the data folder first), then open `http://your-server:8080`.

The image is `ghcr.io/zachcurry13/recipebank` (`:latest` is the newest release).

## Settings from the environment

| Variable | Default | What it does |
|---|---|---|
| `RECIPEBANK_ADDR` | `:8080` | Where it listens |
| `RECIPEBANK_DATA_DIR` | `/config` if mounted, else `/data` (an existing `/data` database always wins) | The database |
| `RECIPEBANK_PHOTOS_DIR` | `/photos` if mounted, else inside the database folder | Dish photos and card scans |
| `RECIPEBANK_CACHE_DIR` | `/cache` if mounted, else inside the database folder | Smaller copies of photos (safe to delete) |
| `RECIPEBANK_TRUST_PROXY` | `true` | Use the visitor's address from a proxy or Cloudflare |
| `RECIPEBANK_SESSION_DAYS` | `30` | How long "Keep me signed in" lasts |
| `RECIPEBANK_ADMIN_USER` / `RECIPEBANK_ADMIN_PASSWORD` | | Optional: create the first admin at start instead of in the browser |

Everything else (the AI, allergen list, units) is set in the app under **Admin**.

## Building from source

Go 1.26+. The web app is plain JavaScript modules and a Tailwind stylesheet that is committed, so `go build ./cmd/recipebank` needs no Node. `make test`, `make css` (needs Node), `make run`.

## License

MIT, © 2026 ZachCurry13. Built from the same core as [NovelCheck](https://github.com/ZachCurry13/novelcheck).

Bundled: [ZXing](https://github.com/zxing-js/library) for reading barcodes (Apache License 2.0, `web/static/vendor/zxing.LICENSE.txt`).
Product details come from [Open Food Facts](https://world.openfoodfacts.org) and its sister databases (open data, ODbL); only the barcode is sent.
