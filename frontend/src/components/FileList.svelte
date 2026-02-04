<script>
    /**
     * @typedef {Object} TreeNode
     * @property {string} type - "dir" | "file" | "zip"
     * @property {string} name
     * @property {string} [relPath]
     * @property {string} [source]
     * @property {string} [url]
     * @property {number} [size]
     * @property {string} [mediaType]
     */

    /** @type {{
     *   items: TreeNode[],
     *   onEnterDir: (node: TreeNode) => void,
     *   onPlayFile: (node: TreeNode) => void,
     *   onExtractZip: (node: TreeNode) => void
     * }} */
    let { items, onEnterDir, onPlayFile, onExtractZip } = $props();

    import { makeMediaId, progressStore } from "../progress.js";

    // 关键：订阅 store，并把值落到本地 state，确保进度更新会触发重渲染
    let allProgress = $state({});
    $effect(() => {
        const unsub = progressStore.subscribe((v) => {
            allProgress = v || {};
        });
        return unsub;
    });

    function formatSize(bytes) {
        if (!Number.isFinite(bytes)) return "";
        const units = ["B", "KB", "MB", "GB", "TB"];
        let v = bytes;
        let i = 0;
        while (v >= 1024 && i < units.length - 1) {
            v /= 1024;
            i++;
        }
        const n = i === 0 ? String(Math.round(v)) : v.toFixed(v >= 10 ? 1 : 2);
        return `${n} ${units[i]}`;
    }

    function formatTime(sec) {
        const s = Math.max(0, Math.floor(Number(sec) || 0));
        const hh = Math.floor(s / 3600);
        const mm = Math.floor((s % 3600) / 60);
        const ss = s % 60;
        const pad2 = (n) => String(n).padStart(2, "0");
        return hh > 0 ? `${hh}:${pad2(mm)}:${pad2(ss)}` : `${mm}:${pad2(ss)}`;
    }

    function nodeKey(item) {
        if (!item) return "";
        // 对 file：沿用 makeMediaId，确保与播放页写入的进度 key 一致
        if (item.type === "file") return makeMediaId(item);
        // 对 dir/zip：即使 relPath 为空也要保证 key 唯一
        const type = item?.type ? String(item.type) : "node";
        const source = item?.source ? String(item.source) : "";
        const relPath = item?.relPath ? String(item.relPath) : "";
        const name = item?.name ? String(item.name) : "";
        const url = item?.url ? String(item.url) : "";
        return `${type}::${source}::${relPath}::${name || url}`;
    }

    function getItemProgress(item) {
        if (!item || item.type !== "file") return null;
        const id = makeMediaId(item);
        const p = (allProgress && id && allProgress[id]) || null;
        if (!p || !Number.isFinite(p.t) || !Number.isFinite(p.d) || p.d <= 0) return null;
        const pct = Math.max(0, Math.min(100, (p.t / p.d) * 100));
        if (pct < 0.1) return null;
        return { t: p.t, d: p.d, pct };
    }
</script>

<div class="list">
    {#each items as it (nodeKey(it))}
        {#if it.type === "dir"}
            <button type="button" class="row" onclick={() => onEnterDir(it)}>
                <div class="icon folder">📁</div>
                <div class="row-main">
                    <div class="row-title">{it.name}</div>
                    <div class="row-sub">{it.relPath || it.source || ""}</div>
                </div>
                <div class="chev">›</div>
            </button>
        {:else if it.type === "zip"}
            <div class="row zip">
                <div class="icon folder">📦</div>
                <div class="row-main">
                    <div class="row-title">{it.name}</div>
                    <div class="row-sub">{it.relPath}</div>
                </div>
                <button
                    type="button"
                    class="zip-btn"
                    onclick={(e) => {
                        e.stopPropagation();
                        onExtractZip(it);
                    }}
                >
                    解压
                </button>
            </div>
        {:else}
            {@const p = getItemProgress(it)}
            <button type="button" class="row" onclick={() => onPlayFile(it)}>
                <div class="icon file">
                    {it.mediaType === "audio" ? "🎵" : "▶"}
                </div>
                <div class="row-main">
                    <div class="row-title">{it.name}</div>
                    <div class="row-sub">
                        {it.relPath}
                        {#if it.size}
                            <span class="dot">·</span>
                            {formatSize(it.size)}
                        {/if}
                    </div>
                    {#if p}
                        <div class="progress">
                            <div class="progress-bar" style={`width:${p.pct}%`}></div>
                        </div>
                        <div class="progress-text">
                            已看 {formatTime(p.t)} / {formatTime(p.d)}
                        </div>
                    {/if}
                </div>
            </button>
        {/if}
    {/each}
</div>

<style>
    .list {
        display: grid;
        gap: var(--space-sm);
    }
    .row {
        display: grid;
        grid-template-columns: auto 1fr auto;
        align-items: center;
        gap: var(--space-sm);
        width: 100%;
        text-align: left;
        border: 1px solid var(--color-border);
        background: var(--color-bg-card);
        border-radius: var(--radius-lg);
        padding: var(--space-sm);
        box-shadow: var(--shadow-sm);
        cursor: pointer;
        color: inherit;
    }
    .row.zip {
        cursor: default;
    }
    .zip-btn {
        border: 1px solid var(--color-border-strong);
        background: var(--color-bg-overlay);
        border-radius: var(--radius-md);
        padding: var(--space-sm) var(--space-sm);
        font-weight: 700;
        cursor: pointer;
        box-shadow: var(--shadow-md);
    }
    .zip-btn:hover {
        border-color: var(--color-border-hover);
    }
    .row:hover {
        border-color: var(--color-border-hover);
    }
    .icon {
        width: 34px;
        height: 34px;
        border-radius: var(--radius-md);
        display: grid;
        place-items: center;
        font-size: var(--font-size-lg);
        border: 1px solid var(--color-border);
        background: var(--color-bg-subtle);
    }
    .icon.file {
        font-weight: 900;
        color: var(--color-primary-dark);
    }
    .row-main {
        min-width: 0;
        display: grid;
        gap: 2px;
    }
    .progress {
        height: 6px;
        border-radius: 999px;
        background: var(--color-bg-subtle);
        overflow: hidden;
        border: 1px solid var(--color-border);
    }
    .progress-bar {
        height: 100%;
        background: var(--color-primary);
        border-radius: inherit;
    }
    .progress-text {
        font-size: var(--font-size-xs);
        color: var(--color-text-muted);
    }
    .row-title {
        font-weight: 700;
        line-height: 1.2;
        overflow: hidden;
        white-space: nowrap;
        text-overflow: ellipsis;
    }
    .row-sub {
        font-size: var(--font-size-sm);
        color: var(--color-text-muted);
        overflow: hidden;
        white-space: nowrap;
        text-overflow: ellipsis;
    }
    .chev {
        color: var(--color-text-muted);
        font-size: var(--font-size-xl);
        padding: 0 var(--space-xs);
    }
    .dot {
        margin: 0 6px;
    }
</style>
