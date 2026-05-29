const LS_KEY = "yseren:playback-rate:v1";

/** @type {readonly number[]} */
export const PLAYBACK_RATES = Object.freeze([0.5, 0.75, 1, 1.25, 1.5, 1.75, 2]);

function loadRate() {
  try {
    const raw = localStorage.getItem(LS_KEY);
    const rate = Number(raw);
    if (PLAYBACK_RATES.includes(rate)) return rate;
  } catch {
    // ignore
  }
  return 1;
}

export const playbackState = $state({ rate: loadRate() });

/** @param {number} rate */
export function setPlaybackRate(rate) {
  if (!PLAYBACK_RATES.includes(rate)) return;
  playbackState.rate = rate;
  try {
    localStorage.setItem(LS_KEY, String(rate));
  } catch {
    // ignore
  }
}

/** @param {number} rate */
export function formatPlaybackRate(rate) {
  if (rate === 1) return "1x";
  return `${rate}x`;
}
