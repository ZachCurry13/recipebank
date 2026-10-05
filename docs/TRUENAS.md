# Installing RecipeBank on TrueNAS SCALE (step by step)

This guide is for TrueNAS SCALE **24.10 (Electric Eel) or newer**. It takes about 10 minutes and happens entirely in web pages. You don't need a terminal.

You'll need your TrueNAS web address (for example `http://192.168.1.50`).

---

## Step 1: Make the folders (datasets)

RecipeBank keeps three kinds of files apart, so each can live where it fits best:

| Folder | What's in it | Where it fits |
|---|---|---|
| **Config** | The database: recipes, people, settings. Small, changes often. | Your fast (SSD) pool, with snapshots |
| **Cache** | Smaller copies of photos for quick pages. Safe to delete; made again when needed. | Your fast pool; no snapshots or backups needed |
| **Photos** | Dish photos and the scans of recipe cards. Grows over time and can't be replaced. | Any pool, for example your big pool next to your other photos; back it up |

If you keep each app's settings in one place on the fast pool (for example a `config` dataset with one dataset per app inside it), RecipeBank fits right in:

```
fast pool
└── config
    └── recipebank        ← add this
        ├── config        ← the database
        └── cache         ← previews
big pool
└── media (or wherever your photos live)
    └── recipebank-photos ← dish photos and card scans
```

To make each dataset:

1. In TrueNAS, open **Datasets** from the left menu.
2. Click the dataset it goes inside (for example `config`), then **Add Dataset**.
3. Enter the **Name** and set **Dataset Preset** to `Apps`. This lets apps save files there.
4. Click **Save**, and write down the path shown at the top of the dataset's details. With the layout above that's for example `/mnt/ssd/config/recipebank/config`, `/mnt/ssd/config/recipebank/cache` and `/mnt/tank/media/recipebank-photos`. You'll need them in Step 2.

Only **Config** is required. If you skip **Photos** or **Cache**, RecipeBank keeps those files in a folder inside Config instead. You can add them later: when a Photos folder appears, RecipeBank moves the photos into it by itself the next time it starts (each one is copied and checked before the old copy is removed).

Installed version 0.1.0 or 0.1.1 with a `/data` folder? Keep it: RecipeBank keeps using `/data` whenever it already holds a database.

## Step 2: Install the app

TrueNAS has two ways to install an app that isn't in its catalog. They give the same result, so pick whichever you prefer:
- **Option A: Install Custom App**, a fill-in-the-boxes form.
- **Option B: Install via YAML**, pasting one block of text.

Both use the image `ghcr.io/zachcurry13/recipebank` and TrueNAS's default pull policy, **only pull the image if it isn't on the NAS yet**, so RecipeBank starts from its own copy, even at boot before the internet is up. New versions arrive through TrueNAS's **Update** button (see [Updating RecipeBank](#updating-recipebank)).

### Option A: Install Custom App (form)

1. Open **Apps** from the left menu, click **Discover Apps** (top right), then **Custom App**.
2. Fill in these sections. Leave anything not listed here at its default.

| Section | Field | Enter |
|---|---|---|
| Application Name | Application Name | `recipebank` |
| Image Configuration | Repository | `ghcr.io/zachcurry13/recipebank` |
| | Tag | `latest` |
| | Pull Policy | **Only pull image if not present on host** (the default) |
| Container Configuration | Timezone | your time zone, for example `Europe/Berlin` |
| | Restart Policy | **Unless Stopped** |
| Security Context Configuration | Custom User | tick it, then **User ID** `568` and **Group ID** `568` |
| Network Configuration | Ports → **Add** | Container Port `8080`, Host Port `30090`, Protocol **TCP** |
| Storage Configuration | Storage → **Add** (1st) | Type **Host Path**, Mount Path `/config`, Host Path = your **Config** folder (for example `/mnt/ssd/config/recipebank/config`) |
| | Storage → **Add** (2nd) | Type **Host Path**, Mount Path `/cache`, Host Path = your **Cache** folder (for example `/mnt/ssd/config/recipebank/cache`) |
| | Storage → **Add** (3rd) | Type **Host Path**, Mount Path `/photos`, Host Path = your **Photos** folder (for example `/mnt/tank/media/recipebank-photos`) |

3. Click **Install**. TrueNAS downloads RecipeBank, which takes a minute. Wait until the app shows **Running**.

Field names can differ slightly between TrueNAS versions (for example "Ports" may be "Port Forwarding"). Look for the closest match.

### Option B: Install via YAML

1. Open **Apps** from the left menu, then click **Discover Apps** (top right).
2. Click the **⋮** (three dots) menu at the top right and choose **Install via YAML**.
3. **Name:** `recipebank`
4. Paste the text below into the big box, then change the **four lines marked `👈 CHANGE`**:

```yaml
services:
  recipebank:
    image: ghcr.io/zachcurry13/recipebank:latest
    pull_policy: missing
    container_name: recipebank
    restart: unless-stopped
    user: "568:568"
    ports:
      - "30090:8080"
    environment:
      TZ: Etc/UTC                            # 👈 CHANGE to your time zone, e.g. Europe/Berlin
    volumes:
      - /mnt/ssd/config/recipebank/config:/config    # 👈 CHANGE left side to your Config folder
      - /mnt/ssd/config/recipebank/cache:/cache      # 👈 CHANGE left side to your Cache folder
      - /mnt/tank/media/recipebank-photos:/photos    # 👈 CHANGE left side to your Photos folder
```

   - On the volume lines, only change the part **before** the `:`. Leave `:/config`, `:/cache` and `:/photos` exactly as they are.
   - Skipping Photos or Cache? Delete that line.
5. Click **Save**. Wait until the app shows **Running**.

## Step 3: Create your admin account

1. In your browser, go to `http://YOUR-TRUENAS-IP:30090`, for example `http://192.168.1.50:30090`.
2. The first time, RecipeBank shows **Welcome to RecipeBank**. Choose your admin username and password, then click **Create admin account**.
   - Do this right after installing. Until an admin account exists, anyone on your network who opens the page could create it.

## Step 4: First-time setup (inside RecipeBank)

1. **Family:** add each person who eats with you, with their allergies, diets, dislikes and how much heat they like. For a real allergy pick **Allergic** or **Severe (strict)**: the page explains the difference. Tick the **pets** you have, for the Home & Care warnings.
2. **Admin → AI (optional but recommended):** the AI reads photos of recipe cards and pages that don't use the standard recipe format. Allergy checks never depend on it.
   - **Ollama on your network (free):** Kind *OpenAI-compatible*, Address `http://YOUR-SERVER:11434/v1` (or the port your Ollama app uses), Model for example `qwen2.5:7b`, and for photos a vision model such as `qwen2.5vl:7b` or `llama3.2-vision`.
   - **A cloud AI:** Kind *OpenAI-compatible* with `https://api.openai.com/v1` and a model like `gpt-4o-mini`, or Kind *Anthropic* with a Claude model. Paste the API key.
   - Click **Save**, then **Test the AI**.
3. **Admin → Accounts:** add the other parent as **Parent** (adds and edits recipes and people) and kids as **Kid** (read and cook). Link each account to its person in the Family list.
4. **Add a recipe:** paste a link, take photos of a card, paste text, or type it in. Each one is checked for everyone before you save it.

## Step 5: Put it on your phone's home screen

- **iPhone (Safari):** open the RecipeBank address, tap **Share**, then **Add to Home Screen**.
- **Android (Chrome):** open the address, then ⋮ → **Add to Home screen** (or **Install app**).

Cook mode keeps the screen on only when RecipeBank is opened through a secure (`https://`) address; on a plain `http://` address the screen may dim while you cook.

---

## Updating RecipeBank

To update, go to **Apps** and click **recipebank**. When TrueNAS shows that an update is available, click **Update**. Your recipes, people and photos are kept. To add the Photos or Cache folder to an existing install, **Edit** the app, add the storage entry (or volume line) from Step 2, and **Save**. (TrueNAS checks for new images now and then, so it may take a while to notice a new version.)

If TrueNAS doesn't offer the update:
1. **Apps → Configuration → Manage Container Images → Pull Image**. Enter `ghcr.io/zachcurry13/recipebank` with the tag `latest`, and pull it.
2. Click **recipebank → Edit**, then **Save** without changing anything. The app restarts on the new version.

The version you are running shows at the bottom of every page.

## Backups

- **Config** (`recipebank.db`) and **Photos** are the ones that matter. Protect them with TrueNAS snapshots (**Data Protection → Periodic Snapshot Tasks**) or copy them somewhere safe. The database holds passwords (scrambled) and your AI key.
- **Cache** never needs a backup.

To check where RecipeBank keeps each kind of file, open **Admin → House settings → Where files are kept**. "Own dataset" means that folder is mounted separately.

## Troubleshooting

| Problem | Fix |
|---|---|
| App won't start, logs say **permission denied** on `/config` | The Config dataset must use the **Apps** preset. Or: **Datasets → recipebank → config → Permissions → Edit** and give the **apps** user (568) *Modify* access. |
| Logs say **can't write to /photos** (or `/cache`), or photos won't save | Same fix for that dataset: the **Apps** preset, or *Modify* access for the **apps** user (568). |
| Photos show inside the database folder in **Where files are kept** | The `/photos` mount is missing or misspelled. **Edit** the app, check the Mount Path is exactly `/photos`, and **Save**. Existing photos are moved over on the next start. |
| Can't open `http://…:30090` | Another app may already use port 30090. Edit the app and change `30090` to another number like `30091`. |
| A recipe link says the site **blocks apps** | Some publishers block apps on purpose. Open the recipe in your browser, select and copy it, then use **Paste text**. |
| Reading a photo says it **doesn't fit in the AI model's working memory** | RecipeBank already sends smaller copies when the AI says a photo is too big. If even the smallest doesn't fit, read one photo at a time, or give the model more room on the AI server: for Ollama, set its context length to 8192 or more (the `OLLAMA_CONTEXT_LENGTH` setting, or the model's `num_ctx`). |
| Photos or pasted text say **no AI is set up** | Set one up under **Admin → AI** (Step 4). Pasted text still works without it when it has a line saying *Ingredients* and a line saying *Directions*. |
| Forgot a password | Another admin can set a new one under **Admin → Accounts → 🔑 Password**. Keep two admin accounts so this is always possible. |
| See what's going on | **Apps → recipebank → Logs** (the icon on the container row). |
