// Product barcodes (EAN/UPC) from a photo, or live from the camera. Uses the
// browser's own BarcodeDetector where it exists (Chrome on Android) and the
// bundled ZXing library everywhere else (iPhone). From NovelCheck.
import { esc } from "./ui.js";
import { shrinkPhoto } from "./photo.js";

const FORMATS = ["ean_13", "ean_8", "upc_a", "upc_e"];
export const isBarcode = (v) => /^\d{8}$|^\d{12,14}$/.test(v || "");

let zxingLoad = null;
function loadZXing() {
  zxingLoad ??= new Promise((resolve, reject) => {
    if (window.ZXing) return resolve(window.ZXing);
    const s = document.createElement("script");
    s.src = "/vendor/zxing.min.js";
    s.onload = () => resolve(window.ZXing);
    s.onerror = () => {
      zxingLoad = null;
      reject(new Error("The barcode reader couldn't load."));
    };
    document.head.append(s);
  });
  return zxingLoad;
}

async function nativeDetector() {
  if (!("BarcodeDetector" in window)) return null;
  const supported = (await window.BarcodeDetector.getSupportedFormats?.()) || [];
  const formats = FORMATS.filter((f) => supported.includes(f));
  return formats.length ? new window.BarcodeDetector({ formats }) : null;
}

async function zxingReader() {
  const ZXing = await loadZXing();
  const F = ZXing.BarcodeFormat;
  const hints = new Map([[ZXing.DecodeHintType.POSSIBLE_FORMATS, [F.EAN_13, F.EAN_8, F.UPC_A, F.UPC_E]],
    [ZXing.DecodeHintType.TRY_HARDER, true]]);
  return new ZXing.BrowserMultiFormatReader(hints);
}

// fromPhoto reads the barcode in a photo ("" if none was found). Works on
// any address, unlike the live camera.
export async function fromPhoto(file) {
  const detector = await nativeDetector();
  if (detector) {
    const codes = await detector.detect(await createImageBitmap(file)).catch(() => []);
    const hit = codes.map((c) => c.rawValue).find(isBarcode);
    if (hit) return hit;
  }
  try {
    const reader = await zxingReader();
    const res = await reader.decodeFromImageUrl(await shrinkPhoto(file, 1600));
    return isBarcode(res?.getText()) ? res.getText() : "";
  } catch {
    return "";
  }
}

// liveBlocker says why live scanning can't start here ("" if it can).
export function liveBlocker() {
  if (!window.isSecureContext) return "Live scanning needs RecipeBank's secure https:// address. Take a photo of the barcode instead.";
  if (!navigator.mediaDevices?.getUserMedia) return "This browser can't use the camera for live scanning. Take a photo instead.";
  return "";
}

function cameraError(err) {
  switch (err?.name) {
    case "NotAllowedError":
    case "SecurityError":
      return "Camera access was blocked. Allow the camera for RecipeBank in the browser's settings, or take a photo instead.";
    case "NotFoundError":
    case "OverconstrainedError":
      return "No camera was found. Take a photo instead.";
    case "NotReadableError":
      return "The camera is busy in another app. Close that app and try again.";
    default:
      return `${esc(err?.message || "The camera couldn't start.")} Take a photo instead.`;
  }
}

// scanLive shows the camera in dlg and resolves with a barcode, or "" when closed.
export async function scanLive(dlg) {
  let stopped = false;
  let release = null;
  let done;
  const result = new Promise((r) => (done = r));
  const finish = (code) => {
    stopped = true;
    release?.();
    if (dlg.open) dlg.close();
    done(code);
  };
  dlg.innerHTML = `<div class="space-y-3 p-4">
    <div class="flex items-center justify-between gap-3"><h2 class="font-semibold">Point the camera at the barcode</h2>
      <button data-close class="btn-ghost px-2 text-xl" aria-label="Close">✕</button></div>
    <video class="w-full rounded-lg bg-black" playsinline muted></video>
    <p data-msg class="text-sm text-slate-400">Hold it steady, about a hand's width away.</p></div>`;
  dlg.onclick = (e) => { if (e.target === dlg || e.target.closest("[data-close]")) finish(""); };
  dlg.oncancel = () => finish("");
  if (!dlg.open) dlg.showModal();
  const video = dlg.querySelector("video");
  const back = { video: { facingMode: "environment" } };
  try {
    const detector = await nativeDetector();
    if (detector) {
      const stream = await navigator.mediaDevices.getUserMedia(back);
      release = () => stream.getTracks().forEach((t) => t.stop());
      if (stopped) return release();
      video.srcObject = stream;
      await video.play();
      const tick = async () => {
        if (stopped) return;
        const codes = await detector.detect(video).catch(() => []);
        const hit = codes.map((c) => c.rawValue).find(isBarcode);
        if (hit) return finish(hit);
        setTimeout(tick, 200);
      };
      tick();
    } else {
      const reader = await zxingReader();
      release = () => reader.reset();
      if (stopped) return release();
      await reader.decodeFromConstraints(back, video, (res) => {
        const text = res?.getText();
        if (isBarcode(text) && !stopped) finish(text);
      });
    }
  } catch (err) {
    release?.();
    const msg = dlg.querySelector("[data-msg]");
    msg.className = "text-sm text-rose-300";
    msg.innerHTML = cameraError(err);
  }
  return result;
}
