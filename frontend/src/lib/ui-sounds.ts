import { clickSoftSound } from "@/lib/click-soft";
import { errorBuzzSound } from "@/lib/error-buzz";
import { playSound } from "@/lib/sound-engine";
import { successChimeSound } from "@/lib/success-chime";
import { switchOffSound } from "@/lib/switch-off";
import { switchOnSound } from "@/lib/switch-on";
import type { SoundAsset } from "@/lib/sound-types";

const sounds = {
  click: clickSoftSound,
  success: successChimeSound,
  error: errorBuzzSound,
  "switch-on": switchOnSound,
  "switch-off": switchOffSound,
} satisfies Record<string, SoundAsset>;

export type UiSound = keyof typeof sounds;

const volume: Record<UiSound, number> = {
  click: 0.45,
  success: 0.4,
  error: 0.4,
  "switch-on": 0.4,
  "switch-off": 0.4,
};

export function playUiSound(name: UiSound) {
  const sound = sounds[name];
  void playSound(sound.dataUri, { volume: volume[name] }).catch(() => {
    // Autoplay can reject before the first gesture. The next interaction retries.
  });
}

const interactiveSelector = [
  "button",
  "a[href]",
  "[role='tab']",
  "[role='switch']",
  "[role='option']",
  "[role='menuitem']",
].join(",");

export function playInteractionSound(target: Element) {
  const control = target.closest(interactiveSelector);
  if (!control || control.getAttribute("data-sound") === "none") {
    return;
  }
  if (control.matches(":disabled") || control.getAttribute("aria-disabled") === "true") {
    return;
  }
  const toggle = control.closest("[role='switch']");
  if (toggle) {
    playUiSound(toggle.getAttribute("aria-checked") === "true" ? "switch-on" : "switch-off");
    return;
  }
  playUiSound("click");
}
