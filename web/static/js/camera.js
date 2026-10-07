// Taking photos on phones. Opening the phone's camera app from a web page
// can make Android close the browser to free memory ("Unable to complete
// previous operation due to low memory"), and the photo is lost. So "Take a
// photo" uses the camera inside the page when the browser allows it (over
// https), and "Choose a photo" always works with a picture taken earlier.
import { esc, toast } from "./ui.js";

export const cameraInPage = () => Boolean(window.isSecureContext && navigator.mediaDevices?.getUserMedia);

// photoPicker is the Take and Choose buttons; name ties them to wirePhotoPicker.
export function photoPicker(name, { take = "📷 Take a photo", choose = "🖼️ Choose a photo", multiple = false, primary = false } = {}) {
  return `<button type="button" data-take="${name}" class="${primary ? "btn-primary" : "btn-secondary"}">${take}</button>
    <label class="btn-secondary cursor-pointer">${choose}<input type="file" accept="image/*" ${multiple ? "multiple" : ""} class="sr-only" data-choose="${name}"></label>
    <input type="file" accept="image/*" capture="environment" class="hidden" data-capture="${name}" tabindex="-1" aria-hidden="true">`;
}

// cameraTip helps when the camera app is the only way (a plain http:// address).
export const cameraTip = () => (cameraInPage() ? ""
  : `<p class="text-xs text-slate-500">If your phone closes RecipeBank while the camera is open, take the picture with the camera app first, then use Choose a photo.</p>`);

// wirePhotoPicker calls onFiles with the photos taken or chosen.
export function wirePhotoPicker(root, name, onFiles) {
  const take = root.querySelector(`[data-take="${name}"]`);
  const choose = root.querySelector(`[data-choose="${name}"]`);
  const capture = root.querySelector(`[data-capture="${name}"]`);
  const from = (inp) => () => {
    const files = [...inp.files];
    inp.value = "";
    if (files.length) onFiles(files);
  };
  if (choose) choose.onchange = from(choose);
  if (capture) capture.onchange = from(capture);
  if (take) take.onclick = async () => {
    if (!cameraInPage()) return capture.click(); // the camera app; Choose a photo is the fallback
    try {
      const file = await shoot();
      if (file) onFiles([file]);
    } catch (e) {
      toast(`${cameraProblem(e)} Use Choose a photo instead.`, true);
    }
  };
}

function cameraProblem(e) {
  switch (e?.name) {
    case "NotAllowedError":
    case "SecurityError":
      return "Camera access was blocked: allow the camera for RecipeBank in the browser's settings.";
    case "NotFoundError":
    case "OverconstrainedError":
      return "No camera was found.";
    case "NotReadableError":
      return "The camera is busy in another app.";
    default:
      return esc(e?.message || "The camera couldn't start.");
  }
}

// cameraDialog is its own window, so a sheet that's open (a receipt, a
// barcode) is still there after the photo.
function cameraDialog() {
  let d = document.getElementById("camera");
  if (!d) {
    d = document.createElement("dialog");
    d.id = "camera";
    d.className = "dialog";
    document.body.append(d);
  }
  return d;
}

// shoot shows the camera; it resolves with the photo, or nothing when closed.
async function shoot() {
  const stream = await navigator.mediaDevices.getUserMedia({ audio: false,
    video: { facingMode: "environment", width: { ideal: 1920 }, height: { ideal: 1440 } } });
  const d = cameraDialog();
  d.innerHTML = `<div class="space-y-3 p-4">
    <div class="flex items-center"><h2 class="text-lg font-semibold">📷 Take a photo</h2>
      <button type="button" data-close class="btn-ghost ml-auto" aria-label="Close">✕</button></div>
    <video class="max-h-[60dvh] w-full rounded-lg bg-black object-contain" playsinline muted autoplay></video>
    <button type="button" data-shoot class="btn-primary w-full py-4 text-lg">📸 Take it</button></div>`;
  const video = d.querySelector("video");
  video.srcObject = stream;
  return new Promise((resolve) => {
    let done = false;
    const finish = (file) => {
      if (done) return;
      done = true;
      stream.getTracks().forEach((t) => t.stop());
      if (d.open) d.close();
      resolve(file);
    };
    d.onclose = () => finish(undefined);
    d.onclick = (e) => { if (e.target === d || e.target.closest("[data-close]")) finish(undefined); };
    d.querySelector("[data-shoot]").onclick = () => {
      if (!video.videoWidth) return; // not started yet
      const c = document.createElement("canvas");
      c.width = video.videoWidth;
      c.height = video.videoHeight;
      c.getContext("2d").drawImage(video, 0, 0);
      c.toBlob((b) => finish(b ? new File([b], "photo.jpg", { type: "image/jpeg" }) : undefined), "image/jpeg", 0.9);
    };
    d.showModal();
  });
}
