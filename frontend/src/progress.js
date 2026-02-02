const LS_KEY = "yseren:progress:v1";
const MAX_ITEMS = 500;

function safeParse(raw) {
  try {
    return raw ? JSON.parse(raw) : null;
  } catch {
    return null;
  }
}

function loadAll() {
  const data = safeParse(localStorage.getItem(LS_KEY));
  if (!data || typeof data !== "object") return { v: 1, items: {} };
  if (data.v !== 1 || !data.items || typeof data.items !== "object") {
    return { v: 1, items: {} };
  }
  return data;
}

function saveAll(data) {
  try {
    localStorage.setItem(LS_KEY, JSON.stringify(data));
  } catch {
    // ignore (storage full / private mode)
  }
}

function nowSec() {
  return Math.floor(Date.now() / 1000);
}

function prune(data) {
  const keys = Object.keys(data.items);
  if (keys.length <= MAX_ITEMS) return;
  keys.sort((a, b) => (data.items[b]?.updatedAt || 0) - (data.items[a]?.updatedAt || 0));
  for (let i = MAX_ITEMS; i < keys.length; i++) {
    delete data.items[keys[i]];
  }
}

export function makeMediaId(item) {
  // 优先用 source+relPath（更稳定），否则退化到 url
  const source = item?.source ? String(item.source) : "";
  const relPath = item?.relPath ? String(item.relPath) : "";
  if (source && relPath) return `${source}::${relPath}`;
  const url = item?.url ? String(item.url) : "";
  return url ? `url::${url}` : "";
}

export function getProgress(id) {
  if (!id) return null;
  const data = loadAll();
  return data.items[id] || null;
}

export function setProgress(id, t, d, ended = false) {
  if (!id) return;
  if (!Number.isFinite(t) || t < 0) return;
  const data = loadAll();
  data.items[id] = {
    t,
    d: Number.isFinite(d) ? d : 0,
    ended: !!ended,
    updatedAt: nowSec(),
  };
  prune(data);
  saveAll(data);
}

export function clearProgress(id) {
  if (!id) return;
  const data = loadAll();
  if (data.items[id]) {
    delete data.items[id];
    saveAll(data);
  }
}
