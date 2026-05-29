const LS_DISMISS_PREFIX = "yseren:dismiss-update:";
const LS_CHECK_AT = "yseren:update-check-at:v1";
const CHECK_INTERVAL_MS = 24 * 60 * 60 * 1000;

/** @typedef {{ version: string, tag: string, url: string, name?: string }} UpdateInfo */

export const updateState = $state({
  loading: false,
  /** @type {UpdateInfo | null} */
  info: null,
});

function isDismissed(version) {
  try {
    return localStorage.getItem(`${LS_DISMISS_PREFIX}${version}`) === "1";
  } catch {
    return false;
  }
}

function shouldCheckNow(force) {
  if (force) return true;
  try {
    const last = Number(localStorage.getItem(LS_CHECK_AT) || 0);
    return !last || Date.now() - last >= CHECK_INTERVAL_MS;
  } catch {
    return true;
  }
}

function markChecked() {
  try {
    localStorage.setItem(LS_CHECK_AT, String(Date.now()));
  } catch {
    // ignore
  }
}

/** @param {{ force?: boolean }} [options] */
export async function checkForUpdate(options = {}) {
  const { force = false } = options;
  if (!force && !shouldCheckNow(false)) return;
  if (updateState.loading) return;

  updateState.loading = true;
  markChecked();
  try {
    const res = await fetch("/api/version");
    if (!res.ok) return;
    const contentType = res.headers.get("content-type") || "";
    if (!contentType.includes("application/json")) return;
    const data = await res.json();

    const update = data?.update;
    if (update?.version && update?.url && !isDismissed(update.version)) {
      updateState.info = {
        version: String(update.version),
        tag: String(update.tag || `v${update.version}`),
        url: String(update.url),
        name: update.name ? String(update.name) : "",
      };
    } else {
      updateState.info = null;
    }
  } catch {
    // 离线或后端不可达时不打扰用户
  } finally {
    updateState.loading = false;
  }
}

/** @param {string} version */
export function dismissUpdate(version) {
  if (!version) return;
  try {
    localStorage.setItem(`${LS_DISMISS_PREFIX}${version}`, "1");
  } catch {
    // ignore
  }
  updateState.info = null;
}
