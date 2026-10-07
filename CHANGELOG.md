# Changelog

## [0.6.0] - 2026-10-07

Show only what your family uses, allergies with exceptions, and better tools for your own AI.

### 🌿 Show only what you use
- **Admin → Features:** turn off what the family doesn't use (the meal plan, shopping list, Home & Care, pantry, collections, bookshelf, events, prices, nutrition, email, share links, kids can help). It's hidden for everyone; nothing is deleted, and turning it back on brings it all back.
- **Me → Keep it simple:** each person can choose which meals their plan shows (for example, only dinner) and leave pages out of their own menu, without changing anything for anyone else.

### 🥜 Allergies with exceptions
- Each allergy on the Family page has **Can have anyway**, for foods made from the allergen that the person's doctor says are fine: highly refined soybean oil or soy lecithin for a soy allergy, highly refined peanut oil for a peanut allergy, or one you type ("almond" for someone allergic only to some tree nuts).
- Only that exact food is let through. Everything else made from the allergen on the line still counts ("soybean oil and soy sauce" is still soy), and cold-pressed, roasted or gourmet oils keep counting.

### 🤖 Your AI, your way
- **A second AI just for photos:** handwritten cards, receipts, the fridge and cookbook indexes go to it first, for example a bigger model on a computer with a stronger graphics card, or an online service. When it can't be reached, the main AI reads the photos. Text stays with the main AI.
- **🦙 Ollama, without a terminal:** find Ollama on your network, see which text and photo models fit your graphics card, download with a progress bar, use a model for text or photos, and delete models you don't use. Brought over from NovelCheck.
- **🩺 AI check-up:** whether each AI is answering (it asks for its models, which costs nothing), what online AI cost this month, newer versions of your Ollama models with an **Update** button, and a **speed test** that times each model on a made-up recipe card and says whether it read it right.
- **Quick setup** fills in the settings for Ollama, LM Studio, Google Gemini, OpenAI or Claude, with prices for the monthly cost.

### Fixes
- On a computer, the top menu no longer scrolls sideways: the main pages are in the bar (more of them on wider screens) and the rest are under **☰ More**.

Updating: nothing to change in TrueNAS. RecipeBank adds what it needs to its database by itself.

## [0.5.1] - 2026-10-06

A fix for taking photos on Android phones.

### Fixes
- Taking a photo of the fridge (or a recipe card, a receipt, a dish or a barcode) could fail with "Unable to complete previous operation due to low memory": the phone closed the browser to make room for its camera app, and the photo was lost. **Take a photo** now uses the camera inside RecipeBank where the browser allows it (over a secure https:// address), so the app never leaves the page.
- Every photo button now has **Choose a photo** next to it, so a picture taken earlier with the phone's camera app always works, even on a plain http:// address.

Updating: nothing to change in TrueNAS.

## [0.5.0] - 2026-10-06

Plan the week in one tap, scan receipts and cookbooks, bring recipes over from other apps, and let the kids help.

### ✨ Plan my week
- On the Plan page, **Plan my week** fills the empty dinners with recipes everyone home that day can eat: ones people liked first, food near its date used up, quicker ones on weeknights, nothing from the last week, and within the weekly budget when prices are known.
- A recipe that makes plenty covers the next night as leftovers. It's a draft: drop any day, **Try again**, or **Add to the plan**.

### 🧾 Scan a receipt
- **Pantry → Scan a receipt:** photograph a store receipt (up to 3 photos for a long one). The AI reads each line; each is matched to a pantry or supply item, or added as new.
- You tick what's right and fix names, amounts and prices before anything is saved. Prices are kept for the budget.

### 🧒 Kids can help
- Every step is marked by what it needs a grown-up for: a knife, a sharp tool, the stove, the oven or hot things, from word lists (never the AI). They lean careful.
- In cook mode, **Kid helper** shows on each step whether a kid can do it. Recipes with steps for kids say **Kids can help**, and the Kitchen can be filtered by it.

### 📖 Bookshelf
- **More → Bookshelf:** add your cookbooks by scanning the barcode on the back (title, author and cover from Open Library) or by hand.
- List each book's recipes with their pages, typed or read by the AI from a photo of the index. Kitchen search shows matches in your cookbooks too ("Lasagna, page 112").
- **Add it** next to a listed recipe photographs the page and links the saved recipe to the book. Recipes show which cookbook and page they came from, and you can set it on any recipe.
- **Still to photograph:** a list of the recipe cards and clippings waiting to be scanned, ticked off when a recipe is saved from one.

### 📦 From another app
- **Add → From another app:** bring all your recipes over from Paprika (.paprikarecipes), Mealie (a backup .zip or recipe .json files) or Tandoor (its export .zip), with their photos. Each recipe is checked for everyone like any other; ones already here are skipped.

### 🔗 Share a recipe with a link
- **Share link** on a recipe makes a read-only page for someone without an account. It shows no names or allergies, runs out after 7 days, and parents can stop sharing at any time.

### 💾 Back up and move recipes
- **Admin → Back up:** download every recipe (Kitchen and Home & Care) with its photos, the collections and the bookshelf as one file. Load it into any RecipeBank, yours after a reinstall or a relative's, to add what it doesn't have yet.

### ✉️ Who may send email
- **Admin → Email → Who can send email:** parents only (the new default) or everyone.

### Fixes
- Saving a recipe drafted from a photo of a dish (Add → From a dish) failed with an error. It saves now.
- Email works right after it's set up, without reloading the page.
- A recipe's cost says how many ingredients had a price in words (phones show no tooltips), and says "No prices yet" instead of "about $0.00".
- The Events list ignores people taken off the Family page, and an event with nobody listed as coming makes the recipe "as written".
- Package sizes like "500g", "1.5 L" and "1 dozen" are understood.

Updating: nothing to change in TrueNAS. The first start of 0.5.0 updates the database by itself (it takes a moment with many recipes).

## [0.4.0] - 2026-10-06

Events, a budget, planned leftovers, email, and how hard a recipe is, plus fixes from going over 0.3.

### 🎉 Events
- **More → Events** for holiday dinners and potlucks: who's coming (family and guests, plus a number of others), the menu, and who brings each dish.
- Every recipe on the menu is checked for everyone coming, and each person sees how many dishes they can eat, with a warning when it's none.
- **Our dishes to the list** buys your own dishes for everyone coming. Print the menu; Tonight mentions an event happening today.

### 💲 Budget
- Give pantry and supply items a price and package size ("$4.50 for 1 lb"); a scanned product's size is filled in from its label.
- The shopping list shows what each priced line costs and how much of your weekly budget is left (**Admin → House settings**: currency and weekly budget).
- Recipes show "about $X to make" from the share of each package they use, and the meal plan shows the week. Both say how many ingredients had a price.

### 🍱 Planned leftovers
- When planning a meal, **Leftovers from…** picks a recipe planned in the last few days. The earlier meal is cooked and bought for both, and the leftovers meal buys nothing. Tonight says "make extra" and "just reheat".

### ✉️ Email
- Email a recipe (with a message) or the shopping list to one address, through your own email account (**Admin → Email**, with a test button and Gmail help).

### 🟢 How hard it is
- Recipes are easy, medium or hard, worked out from active time, steps, ingredients and techniques like kneading or tempering, or set in the editor. Filter the Kitchen by it, or ask smart search for something "easy".

### Fixes
- Metric shows grams for flour, sugar, butter and other things metric cooks weigh (liquids stay in ml, spoons stay spoons).
- Ingredient lines read better: "2 x 400g tins tomatoes", "1 tablespoon + 1 teaspoon sugar", "pinch of salt", "a dash of hot sauce", "can of beans"; "to taste" and "for serving" are notes, not part of the food.

Updating: nothing to change in TrueNAS. RecipeBank adds what it needs to its database by itself.

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
