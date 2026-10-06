-- RecipeBank schema. Every table is CREATE ... IF NOT EXISTS; new columns on
-- existing tables are added in migrate.go.

CREATE TABLE IF NOT EXISTS users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    username      TEXT NOT NULL UNIQUE COLLATE NOCASE,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'kid' CHECK (role IN ('admin', 'editor', 'kid')),
    person_id     INTEGER REFERENCES people(id) ON DELETE SET NULL, -- the eating profile that is "me"
    units         TEXT NOT NULL DEFAULT '' CHECK (units IN ('', 'us', 'metric')), -- '' = the house default
    theme         TEXT NOT NULL DEFAULT '',
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sessions (
    token_hash TEXT PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at DATETIME NOT NULL,
    remember   INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

-- Everyone who eats here (or uses the home recipes), with or without an
-- account: guests and small kids have no login but still have rules.
CREATE TABLE IF NOT EXISTS people (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT NOT NULL,
    is_kid     INTEGER NOT NULL DEFAULT 0,
    is_guest   INTEGER NOT NULL DEFAULT 0,
    heat_max   INTEGER NOT NULL DEFAULT -1, -- 0-5 peppers; -1 = no limit
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- A person's rules: an allergy (key = allergen, severity avoid/allergic/severe),
-- a diet (key = diet), a dislike or a skin sensitivity (key = the word).
CREATE TABLE IF NOT EXISTS person_rules (
    person_id INTEGER NOT NULL REFERENCES people(id) ON DELETE CASCADE,
    kind      TEXT NOT NULL CHECK (kind IN ('allergy', 'diet', 'dislike', 'sensitivity')),
    key       TEXT NOT NULL,
    severity  TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (person_id, kind, key)
);

-- Recipes for the kitchen (food) and for Home & Care (cleaners, toothpaste…).
-- Ingredients and steps are JSON lists (see store.Ingredient and store.Step).
CREATE TABLE IF NOT EXISTS recipes (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    area         TEXT NOT NULL DEFAULT 'kitchen' CHECK (area IN ('kitchen', 'home')),
    title        TEXT NOT NULL,
    summary      TEXT NOT NULL DEFAULT '',
    servings     REAL NOT NULL DEFAULT 0,       -- 0 = unknown
    yield_text   TEXT NOT NULL DEFAULT '',      -- "makes 2 cups", "one 16 oz bottle"
    prep_min     INTEGER NOT NULL DEFAULT 0,
    cook_min     INTEGER NOT NULL DEFAULT 0,
    total_min    INTEGER NOT NULL DEFAULT 0,
    heat         INTEGER NOT NULL DEFAULT -1,   -- 0-5 peppers; -1 = not set
    course       TEXT NOT NULL DEFAULT '',      -- kitchen: main, side, dessert…; home: the category
    cuisine      TEXT NOT NULL DEFAULT '',
    protein      TEXT NOT NULL DEFAULT '',
    ingredients  TEXT NOT NULL DEFAULT '[]',
    steps        TEXT NOT NULL DEFAULT '[]',
    notes        TEXT NOT NULL DEFAULT '',      -- the family's own notes
    storage      TEXT NOT NULL DEFAULT '',      -- home: how to store it, how long it keeps
    source_kind  TEXT NOT NULL DEFAULT 'manual' CHECK (source_kind IN ('web', 'photo', 'text', 'manual')),
    source_url   TEXT NOT NULL DEFAULT '',
    source_note  TEXT NOT NULL DEFAULT '',      -- "Grandma's card", "cookbook, page 42"
    photo        TEXT NOT NULL DEFAULT '',      -- file name under /data/photos
    source_photos TEXT NOT NULL DEFAULT '[]',   -- the card or page photos it was read from
    needs_review INTEGER NOT NULL DEFAULT 0,    -- the AI couldn't read every line
    rating       INTEGER NOT NULL DEFAULT 0,    -- 0-5 stars
    version_of   INTEGER REFERENCES recipes(id) ON DELETE SET NULL, -- "our version" of another recipe
    created_by   TEXT NOT NULL DEFAULT '',
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS recipes_area ON recipes(area, title);

CREATE TABLE IF NOT EXISTS token_usage (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    at                DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    model             TEXT NOT NULL,
    prompt_tokens     INTEGER NOT NULL DEFAULT 0,
    completion_tokens INTEGER NOT NULL DEFAULT 0,
    seconds           REAL NOT NULL DEFAULT 0
);

-- What's in the house: the pantry (area 'kitchen') and the supply closet
-- (area 'home'). Labels come from Open Food Facts or a parent.
CREATE TABLE IF NOT EXISTS stock (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    area             TEXT NOT NULL DEFAULT 'kitchen' CHECK (area IN ('kitchen', 'home')),
    name             TEXT NOT NULL,
    brand            TEXT NOT NULL DEFAULT '',
    barcode          TEXT NOT NULL DEFAULT '',
    qty              REAL NOT NULL DEFAULT 1,
    unit             TEXT NOT NULL DEFAULT '',
    location         TEXT NOT NULL DEFAULT '',      -- fridge, freezer, pantry, bathroom, laundry…
    use_by           TEXT NOT NULL DEFAULT '',      -- YYYY-MM-DD
    low_at           REAL NOT NULL DEFAULT 0,       -- running low at or below this; 0 = not watched
    allergens        TEXT NOT NULL DEFAULT '[]',    -- allergen keys the label lists
    traces           TEXT NOT NULL DEFAULT '[]',    -- "may contain"
    ingredients_text TEXT NOT NULL DEFAULT '',
    label_source     TEXT NOT NULL DEFAULT '',      -- 'off' (Open Food Facts), 'parent', '' = no label
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS stock_area ON stock(area, name);

-- The family's shopping list, shared by everyone.
CREATE TABLE IF NOT EXISTS shopping (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    key        TEXT NOT NULL DEFAULT '',          -- the food's identifying words, for combining amounts
    name       TEXT NOT NULL,
    qty        REAL NOT NULL DEFAULT 0,           -- 0 = no amount
    unit       TEXT NOT NULL DEFAULT '',
    section    TEXT NOT NULL DEFAULT 'Other',
    area       TEXT NOT NULL DEFAULT 'kitchen' CHECK (area IN ('kitchen', 'home')),
    note       TEXT NOT NULL DEFAULT '',          -- "for Soup, Pancakes"
    stock_id   INTEGER REFERENCES stock(id) ON DELETE SET NULL, -- a running-low item to restock when bought
    checked    INTEGER NOT NULL DEFAULT 0,
    added_by   TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- The meal plan: a recipe (or just a note, like "Pizza night out") per meal.
CREATE TABLE IF NOT EXISTS meal_plan (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    date       TEXT NOT NULL,                        -- YYYY-MM-DD
    meal       TEXT NOT NULL CHECK (meal IN ('breakfast', 'lunch', 'dinner', 'snack')),
    recipe_id  INTEGER REFERENCES recipes(id) ON DELETE CASCADE,
    title      TEXT NOT NULL DEFAULT '',             -- when there's no recipe
    servings   REAL NOT NULL DEFAULT 0,              -- 0 = the recipe's own
    note       TEXT NOT NULL DEFAULT '',
    created_by TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS meal_plan_date ON meal_plan(date);

-- Who's eating at home each day (people ids, guests included); no row = everyone but guests.
CREATE TABLE IF NOT EXISTS plan_days (
    date TEXT PRIMARY KEY,
    who  TEXT NOT NULL DEFAULT '[]'
);

-- Collections: themed shelves, filled by hand or with the AI's suggestions (a parent picks).
CREATE TABLE IF NOT EXISTS collections (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',   -- what belongs, in plain words (the AI reads it)
    icon        TEXT NOT NULL DEFAULT '📚',
    area        TEXT NOT NULL DEFAULT 'kitchen' CHECK (area IN ('kitchen', 'home')),
    created_by  TEXT NOT NULL DEFAULT '',
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS collection_recipes (
    collection_id INTEGER NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    recipe_id     INTEGER NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    added_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (collection_id, recipe_id)
);

-- Phones and browsers that turned on notifications, and what each wants
-- (comma-separated: timers, useby, low, tonight).
CREATE TABLE IF NOT EXISTS push_subscriptions (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    endpoint   TEXT NOT NULL UNIQUE,
    p256dh     TEXT NOT NULL,
    auth       TEXT NOT NULL,
    wants      TEXT NOT NULL DEFAULT 'timers,useby,low,tonight',
    device     TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- What the AI misread on this family's cards ("bell pepper flakes" was
-- "red pepper flakes"), learned from corrections and sent with new cards.
CREATE TABLE IF NOT EXISTS reading_hints (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    wrong      TEXT NOT NULL,
    right_text TEXT NOT NULL,
    times      INTEGER NOT NULL DEFAULT 1,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (wrong, right_text)
);

-- Each time a recipe was cooked, and how each person liked it (1 or -1).
CREATE TABLE IF NOT EXISTS cooks (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    recipe_id  INTEGER NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    cooked_on  TEXT NOT NULL,
    note       TEXT NOT NULL DEFAULT '',
    added_by   TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS cooks_recipe ON cooks (recipe_id);
CREATE TABLE IF NOT EXISTS cook_thumbs (
    cook_id   INTEGER NOT NULL REFERENCES cooks(id) ON DELETE CASCADE,
    person_id INTEGER NOT NULL REFERENCES people(id) ON DELETE CASCADE,
    thumb     INTEGER NOT NULL CHECK (thumb IN (-1, 1)),
    PRIMARY KEY (cook_id, person_id)
);
