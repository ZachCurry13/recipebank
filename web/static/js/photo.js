// Photos from the phone camera or gallery.

// shrinkPhoto turns a phone photo into a JPEG data: URL no wider or taller
// than max pixels (small and quick to send; still sharp enough to read a card).
export async function shrinkPhoto(file, max = 2000) {
  const bmp = await createImageBitmap(file);
  const scale = Math.min(1, max / Math.max(bmp.width, bmp.height));
  const canvas = document.createElement("canvas");
  canvas.width = Math.round(bmp.width * scale);
  canvas.height = Math.round(bmp.height * scale);
  canvas.getContext("2d").drawImage(bmp, 0, 0, canvas.width, canvas.height);
  return canvas.toDataURL("image/jpeg", 0.85);
}
