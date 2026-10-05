# Changelog

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
