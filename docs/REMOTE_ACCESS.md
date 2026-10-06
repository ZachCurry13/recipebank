# Using RecipeBank away from home (Cloudflare Tunnel)

A Cloudflare Tunnel gives RecipeBank a secure web address, such as `https://recipes.example.com`, that works from anywhere: the shopping list in the store on cellular, the meal plan at work. You don't open any ports on your router, and your home IP address stays hidden. Cloudflare's tunnel service is free.

The secure `https://` address also unlocks what phones only allow on secure pages: **live barcode scanning** with the camera, **keeping the screen on** and **voice** in cook mode, and **phone notifications**.

RecipeBank has the tunnel connector built in, so there's nothing extra to install on TrueNAS. (If you already did this for NovelCheck, it's the same steps with a second tunnel or a second address.)

## What you need

- A free **Cloudflare account** (dash.cloudflare.com).
- A **domain name** whose DNS is managed by Cloudflare. If you don't have one, you can buy one inside Cloudflare (**Domain Registration → Register Domains**), usually about $10 a year.
- About 10 minutes.

## Step 1: Create the tunnel in Cloudflare

1. Sign in at **dash.cloudflare.com** and open **Zero Trust**. The first time, pick a team name and the **Free** plan.
2. Go to **Networks → Tunnels** and click **Create a tunnel**.
3. Choose **Cloudflared**, name it `recipebank`, and click **Save tunnel**.
4. Cloudflare shows install commands. **Don't run them.** Copy one: they all contain your tunnel token, the long text starting with `eyJ`. You can paste the whole command into RecipeBank and it picks out the token.
5. Click **Next**.

## Step 2: Choose your web address

Add a route (called **Public Hostnames**, or **Published application routes** in newer dashboards):

1. **Subdomain:** `recipes`, or anything you like.
2. **Domain:** pick your domain.
3. **Service type:** `HTTP`.
4. **URL:** `localhost:8080`
5. Click **Save**.

`localhost:8080` is right even though you normally open RecipeBank on another port: the connector runs inside RecipeBank itself.

## Step 3: Turn it on in RecipeBank

1. Open **Admin → Remote access**.
2. Paste the token (or the whole command) into **Tunnel token**, and type your address into **Public address**.
3. Tick **Turn on remote access** and click **Save & connect**.
4. Within about 10 seconds the status says **Connected**. Check by opening the address on your phone with Wi-Fi off.

## Step 4: Use it on your phones

Open the new `https://` address on each phone, sign in, and add it to the home screen (iPhone: **Share → Add to Home Screen**; Android: **Install app**). It works at home and away, so everyone can use this one address. Each address keeps its own sign-in, so everyone signs in once more.

## Optional: an extra lock with Cloudflare Access

RecipeBank already needs a password. For extra peace of mind, Cloudflare can ask for a one-time code sent by email before anyone sees the page: in Zero Trust, **Access → Applications → Add an application → Self-hosted**, enter your address, and add a policy that allows your family's email addresses.

## Alternative: TrueNAS's own Cloudflared app

If you already run the tunnel as its own TrueNAS app: add a route with the URL `YOUR-TRUENAS-IP:30090` (RecipeBank's port) instead of `localhost:8080`, and leave **Remote access** in RecipeBank turned off.

## Troubleshooting

- **Stays "Connecting…" or shows "Trying again":** copy the token again from **Zero Trust → Networks → Tunnels → recipebank → Configure**, paste it and click **Save & connect**. The connector log on the Admin page shows Cloudflare's messages.
- **"Cloudflare rejected the tunnel token":** the token was cut short or the tunnel was deleted. Copy a fresh one.
- **Connected, but the address shows a Cloudflare error (502 or 1033):** the route's URL is wrong. It must be `localhost:8080` with type `HTTP`.
- **The address doesn't load on one device:** that network may remember an old "not found" answer for up to 30 minutes. Try the phone on mobile data, or wait a bit. Ad-blocking DNS (like Pi-hole) can block new addresses too.
- **"QUIC connection failed … suggested_protocol=http2" in the log:** harmless; cloudflared switches to HTTP/2.
- **"Not available in this install":** you're running an image from before 0.2.0. Update RecipeBank.
