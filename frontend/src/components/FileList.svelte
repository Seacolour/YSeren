<script>
    /**
     * @typedef {Object} TreeNode
     * @property {string} type - "dir" | "file"
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
     *   onPlayFile: (node: TreeNode) => void
     * }} */
    let { items, onEnterDir, onPlayFile } = $props();

    import { makeMediaId, progressState } from "../progress.svelte.js";
    import { formatSize, formatTime } from "../utils.js";
    import { addToPlaylist, isInPlaylist } from "../playlist.svelte.js";

    function handleAddToPlaylist(e, item) {
        e.stopPropagation();
        addToPlaylist(item);
    }

    function nodeKey(item) {
        if (!item) return "";
        // 对 file：沿用 makeMediaId，确保与播放页写入的进度 key 一致
        if (item.type === "file") return makeMediaId(item);
        // 对 dir：即使 relPath 为空也要保证 key 唯一
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
        const p = (id && progressState.items[id]) || null;
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
        {:else}
            {@const p = getItemProgress(it)}
            <div class="row file-row">
                <button type="button" class="file-play-area" onclick={() => onPlayFile(it)}>
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
                <button
                    type="button"
                    class="btn-add-playlist"
                    class:added={isInPlaylist(it.url)}
                    title={isInPlaylist(it.url) ? "已在播放列表" : "加入播放列表"}
                    onclick={(e) => handleAddToPlaylist(e, it)}
                >{isInPlaylist(it.url) ? "✓" : "+"}</button>
            </div>
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
    .file-row {
        grid-template-columns: 1fr auto;
        padding: 0;
        gap: 0;
        cursor: default;
    }
    .file-play-area {
        display: grid;
        grid-template-columns: auto 1fr;
        align-items: center;
        gap: var(--space-sm);
        padding: var(--space-sm);
        background: none;
        border: none;
        color: inherit;
        cursor: pointer;
        text-align: left;
        min-width: 0;
    }
    .btn-add-playlist {
        width: 36px;
        height: 36px;
        display: grid;
        place-items: center;
        border: none;
        background: none;
        color: var(--color-text-muted);
        cursor: pointer;
        font-size: 1.2rem;
        font-weight: 700;
        border-radius: var(--radius-md);
        flex-shrink: 0;
        align-self: center;
        margin-right: var(--space-xs);
        transition: background 0.12s, color 0.12s;
    }
    .btn-add-playlist:hover {
        background: var(--color-border);
        color: var(--color-text);
    }
    .btn-add-playlist.added {
        color: var(--color-primary, #6c63ff);
    }
</style>
