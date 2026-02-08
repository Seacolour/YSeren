const LS_KEY = "yseren:playlist:v1";

function safeParse(raw) {
  try {
    return raw ? JSON.parse(raw) : null;
  } catch {
    return null;
  }
}

function load() {
  const data = safeParse(localStorage.getItem(LS_KEY));
  if (!Array.isArray(data)) return [];
  return data;
}

function save(list) {
  try {
    localStorage.setItem(LS_KEY, JSON.stringify(list));
  } catch {
    // ignore
  }
}

export const playlist = $state({ items: load() });

/** 添加到播放列表末尾（去重：同 url 不重复添加） */
export function addToPlaylist(item) {
  if (!item?.url) return;
  if (playlist.items.some((i) => i.url === item.url)) return;
  playlist.items = [
    ...playlist.items,
    {
      name: item.name,
      url: item.url,
      source: item.source || "",
      relPath: item.relPath || "",
      mediaType: item.mediaType || "video",
    },
  ];
  save(playlist.items);
}

/** 从播放列表移除指定索引 */
export function removeFromPlaylist(index) {
  if (index < 0 || index >= playlist.items.length) return;
  playlist.items = playlist.items.filter((_, i) => i !== index);
  save(playlist.items);
}

/** 清空播放列表 */
export function clearPlaylist() {
  playlist.items = [];
  save(playlist.items);
}

/** 拖拽排序：将 fromIndex 移动到 toIndex */
export function reorderPlaylist(fromIndex, toIndex) {
  if (fromIndex === toIndex) return;
  const arr = [...playlist.items];
  const [moved] = arr.splice(fromIndex, 1);
  arr.splice(toIndex, 0, moved);
  playlist.items = arr;
  save(playlist.items);
}

/** 获取指定 url 之后的下一个项目（用于连续播放） */
export function getNextItem(currentUrl) {
  const idx = playlist.items.findIndex((i) => i.url === currentUrl);
  if (idx === -1 || idx >= playlist.items.length - 1) return null;
  return playlist.items[idx + 1];
}

/** 检查某个 url 是否已在播放列表中 */
export function isInPlaylist(url) {
  return playlist.items.some((i) => i.url === url);
}
