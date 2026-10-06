// The shared-recipe page follows the device's light or dark setting.
if (!matchMedia("(prefers-color-scheme: dark)").matches) document.documentElement.classList.add("light");
