# Changelog

## [0.3.0] - 2026-10-06

Cooking from what you have, better reading of recipe cards, and a guide for new people.

### 🥕 What can I make?
- Tick what's in the kitchen (the pantry is ticked already), type more, or photograph the fridge and untick anything the AI got wrong.
- Recipes that need least come first, then ones that use up food near its date. Each shows what's missing, swaps already in the kitchen, and **Add what's missing to the list**.

### ↔ Substitute
- On a recipe, tap **Substitute**, then the ingredient you don't have (cumin, buttermilk…): stand-ins with how much to use. Only ideas that suit everyone eating are shown, and what's in the pantry comes first.
- The AI can suggest more ideas; they're checked by the same allergy and diet rules.

### 📷 Reading recipe cards
- Photos stay the right way up, and ⟳ turns any that aren't.
- The AI copies the card word for word first and then organizes it, so amounts aren't lost. Two-column cards are read left column first.
- After every scan RecipeBank checks the recipe against itself: foods the steps use that no line has, lines without an amount, "11/2" that probably means 1 1/2. Until someone taps **Checked against the card**, the recipe is "not sure" for anyone with an allergy.
- **Read the card again** (turned, or with another AI) and keep the new reading only if it's better.
- Lines you fix while checking teach the reader for the next cards (**Admin → AI** lists what it learned).
- Steps that name an allergen no ingredient line has ("stir in the butter") also make a recipe "not sure".

### 🍽️ From a dish
- **Add → From a dish:** photograph a meal to find your recipes like it, or let the AI draft its best guess, checked like any other draft.

### 🍳 We cooked it
- Log a cook with each person's 👍 or 👎 and a note. The recipe shows its history, and Tonight's ideas put what people liked first.

### 🥗 Nutrition estimates
- Calories, protein, carbohydrates, fat, fiber and sodium per serving, from USDA FoodData Central (a free key in **Admin → House settings**). It lists what it couldn't count. Only food names are sent.

### 🖨 Share, print and the family cookbook
- On Android, share a recipe link to RecipeBank from any app.
- Print a recipe, or the whole **family cookbook** (or one collection) with a cover and contents. Save it as a PDF from the print menu.

### 🥫 Pantry
- A scanned barcode shows the package photo: "Is this what you scanned?" Scanning something already in the house asks before counting one more.

### 👋 Getting started and updates
- A short welcome the first time someone opens RecipeBank, and a **Getting started** checklist for whoever set it up.
- When a new version is out, parents see a note and admins can get a phone notification. After an update, everyone sees what's new once; all the notes are under **Me → What's new**.

### Also
- Ingredients in a cleaner layout, with the amounts in their own column.
- Typing "cheddar" counts as cheddar cheese, and bell peppers and garlic salt are no longer counted as always in the kitchen.
- Card abbreviations like "1 T." and "1 t." are read as tablespoon and teaspoon.

Updating: nothing to change in TrueNAS. RecipeBank adds what it needs to its database by itself.

## [0.2.0] - 2026-10-06

A big one: the pantry, shopping, planning the week, and using RecipeBank away from home.

### 🌙 Tonight and the meal plan
- **Tonight** is the new first page: what's planned today, who's eating and anything that doesn't suit them, Start cooking at the planned servings, food in the pantry to use soon, and ideas everyone can eat when dinner isn't planned.
- **Meal plan:** a week of breakfasts, lunches and dinners (a recipe, or a note like "Leftovers"). Pick who's home each day and add guests; every meal is checked for the people actually eating.
- One tap puts the whole week on the shopping list, scaled to the servings you planned.

### 🥫 Pantry and 🧽 Supplies
- What's in the house: food in the **Pantry**, cleaning and bathroom things in **Supplies**. Amounts with quick +/−, where things are kept, use-by dates and a "running low" level.
- **Scan a barcode** (a photo of it works on any address; the live camera on the secure https address) and the open Open Food Facts databases fill in the name and what the label lists. Only the barcode is sent.
- When a recipe line is "not sure" for an allergy and the same product is in your pantry, the recipe shows what its label says; a parent can use that label in one tap.

### 🛒 Shopping list
- One list for the whole family, grouped by store section, ticked as you shop.
- Add a recipe at the servings you picked: amounts for the same food are combined (2 cups + 4 tablespoons of flour = 2 ¼ cups), and what's already in the house is listed under "you probably have".
- Use up the toothpaste and it goes on the list by itself; tick it and clear the list, and it's back in Supplies.
- **Works in the store without signal:** the app opens with the list saved on your phone, and ticks are sent once you're back online.

### 📚 Collections, seasons and feasts
- Collections you fill by hand, or by describing them ("quick dairy-free weeknight dinners"): the AI suggests recipes, RecipeBank's own rules drop anything that breaks a diet, a time limit or "everyone can eat" (and says why), and you tick what to add.
- Seasonal shelves that come round each year: Advent and Christmas baking, St. Nicholas, St. Lucy, Epiphany, Candlemas crêpes, Valentine's, Mardi Gras, **Lent Fridays (meatless)**, St. Patrick, St. Joseph, Easter, summer grilling, back-to-school lunches, fall harvest, Halloween, Thanksgiving, winter comfort food, and **spring cleaning** for Home & Care. Easter, Advent and Thanksgiving dates are worked out each year.

### ✨ Smart search
- Ask the Kitchen a question in plain words: "something quick with chicken everyone can eat". Diets, time limits and "everyone" are checked by RecipeBank itself; the AI picks matching recipes by meaning (without an AI, shared words and the rules decide).

### 🔊 Cook mode voice
- **Read aloud:** each step is read as you go, and finished timers are announced.
- **Listen** (on the secure address): say "next", "back", "repeat" or "timer".

### 🌍 Away from home, and 📱 notifications
- **Remote access** through a free Cloudflare Tunnel, built in: Admin → Remote access, with a step-by-step guide (docs/REMOTE_ACCESS.md). The secure https address also enables live barcode scanning, keeping the screen on while cooking, voice, and notifications.
- **Phone notifications**, chosen per device under Me: cook-mode timers (even with the screen off), food to use soon each morning, something running low, and tonight's dinner each afternoon (times set in Admin).

### ✓ Fixes
- Chicken broth gets broth swaps (not chickpeas); "Broth with Herbs" and "Tuna in Water" are recognized as broth and tuna.
- "1 large egg" instead of "1 large eggs" when amounts are scaled.

## [0.1.2] - 2026-10-05

### 📷 Reading photos with a home AI
- Photos of recipe cards now fit AI models with little working memory (a home AI server often has room for about 4,000 tokens): RecipeBank sends the AI a smaller copy of each photo, and an even smaller one if the AI says it doesn't fit. The full photo is still kept with the recipe.
- If a photo still doesn't fit, the message says what to do (one photo at a time, or more room for the model on the AI server).

### 🗂️ A config folder, like your other apps
- The database can now live in a `/config` folder, the way many TrueNAS apps keep their settings, with `/cache` and `/photos` beside it. The install guide shows the layout (form and YAML).
- Installs from 0.1.0 or 0.1.1 that use `/data` keep working as they are.

## [0.1.1] - 2026-10-05

### 🗂️ Separate folders for data, photos and cache
- RecipeBank can now keep its database, its photos and a cache in three separate folders, so on TrueNAS each can be its own dataset: the database on a fast pool with snapshots, the photos next to your other photos, and the cache somewhere that needs no backup. The install guide shows how (form and YAML).
- The photos and cache folders are optional. Without them everything stays in the data folder as before; add a photos folder later and RecipeBank moves the photos into it by itself (each one is copied and checked first).
- Library cards and recipe pages load smaller copies of photos from the cache, so pages open faster on phones.
- **Admin → House settings → Where files are kept** shows which folder holds what.

## [0.1.0] - 2026-09-29

The first version, for testing.

### 🍲 Recipes
- Add recipes from a link, from photos of a card or cookbook page (the AI reads it), from pasted text, or by typing them in. Every recipe is checked for everyone before it's saved.
- Recipe pages with a servings scaler, US or metric units, steps with timers, your own notes, star ratings and "Make our version" (a copy to change that keeps the original).
- Cook mode: one big step at a time, the ingredients for that step, and timers that ring.
- Library with search and filters: who's eating, only what they can all eat, time, diet, heat, course, cuisine and protein.

### 👪 Family and allergies
- Add everyone who eats with you, with or without an account: allergies (Avoid, Allergic or Severe), diets, dislikes, skin sensitivities and how much heat they like.
- Each recipe shows who it's OK for, who it's not for and why, and who it's not sure for (check the label). Parents can mark a label as checked.
- Swaps are offered only when they work for everyone eating.

### 🧴 Home & Care
- A separate area for home-made cleaners, laundry, toothpaste, mouthwash and more.
- Warnings before you make it: dangerous mixes, what kids mustn't swallow, what's toxic to your pets, surfaces to avoid, and how to store it.

### ⚙️ Setup
- Install on TrueNAS with the Custom App form or with YAML (see docs/TRUENAS.md).
- Admin page for the AI (Ollama on your network or a cloud AI), the allergen list (US or EU), units and accounts.
