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
</script>

<div class="list">
    {#each items as it (it.type === "file" ? it.url : `${it.source}:${it.relPath}`)}
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
