// Photos from the phone camera or gallery, the right way up.

// readFile reads a file as a data: URL (the page's security rules allow
// data: images, not blob: ones).
function readFile(file) {
  return new Promise((resolve, reject) => {
    const r = new FileReader();
    r.onload = () => resolve(r.result);
    r.onerror = () => reject(new Error("That photo couldn't be read."));
    r.readAsDataURL(file);
  });
}

// loadImage decodes a file or data: URL through an <img>, which honours the
// rotation a phone stores in the photo (so a portrait card stays portrait).
async function loadImage(src) {
  const url = typeof src === "string" ? src : await readFile(src);
  return new Promise((resolve, reject) => {
    const img = new Image();
    img.onload = () => resolve(img);
    img.onerror = () => reject(new Error("That photo couldn't be opened."));
    img.src = url;
  });
}

// draw puts img on a canvas no bigger than max pixels, turned clockwise by
// quarter turns, and returns it as a JPEG data: URL.
function draw(img, max, turns) {
  const scale = Math.min(1, max / Math.max(img.naturalWidth, img.naturalHeight));
  const w = Math.round(img.naturalWidth * scale);
  const h = Math.round(img.naturalHeight * scale);
  const t = ((turns % 4) + 4) % 4;
  const canvas = document.createElement("canvas");
  canvas.width = t % 2 ? h : w;
  canvas.height = t % 2 ? w : h;
  const ctx = canvas.getContext("2d");
  ctx.translate(canvas.width / 2, canvas.height / 2);
  ctx.rotate((t * Math.PI) / 2);
  ctx.drawImage(img, -w / 2, -h / 2, w, h);
  return canvas.toDataURL("image/jpeg", 0.88);
}

// shrinkPhoto turns a phone photo into a JPEG data: URL no wider or taller
// than max pixels (small and quick to send; still sharp enough to read a card).
export async function shrinkPhoto(file, max = 2000) {
  return draw(await loadImage(file), max, 0);
}

// rotatePhoto turns a photo (a data: URL) a quarter turn clockwise.
export async function rotatePhoto(dataURL) {
  const img = await loadImage(dataURL);
  return draw(img, Math.max(img.naturalWidth, img.naturalHeight), 1);
}
