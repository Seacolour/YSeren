<script>
    import { playlist, removeFromPlaylist, clearPlaylist, reorderPlaylist } from "../playlist.svelte.js";

    /** @type {{ onPlay: (item: any) => void, onClose: () => void }} */
    let { onPlay, onClose } = $props();

    let dragIndex = $state(-1);
    let overIndex = $state(-1);

    function onDragStart(e, index) {
        dragIndex = index;
        e.dataTransfer.effectAllowed = "move";
    }

    function onDragOver(e, index) {
        e.preventDefault();
        e.dataTransfer.dropEffect = "move";
        overIndex = index;
    }

    function onDrop(e, index) {
        e.preventDefault();
        if (dragIndex >= 0 && dragIndex !== index) {
            reorderPlaylist(dragIndex, index);
        }
        dragIndex = -1;
        overIndex = -1;
    }

    function onDragEnd() {
        dragIndex = -1;
        overIndex = -1;
    }

    function playItem(item, index) {
        onPlay(item);
    }

    function remove(e, index) {
        e.stopPropagation();
        removeFromPlaylist(index);
    }
</script>

<div class="playlist-backdrop" onclick={onClose} role="presentation"></div>
<aside class="playlist-panel">
    <div class="playlist-header">
        <h3 class="playlist-title">播放列表 ({playlist.items.length})</h3>
        <div class="playlist-actions">
            {#if playlist.items.length > 0}
                <button type="button" class="btn-text" onclick={clearPlaylist}>清空</button>
            {/if}
            <button type="button" class="btn-close" onclick={onClose}>✕</button>
        </div>
    </div>

    {#if playlist.items.length === 0}
        <div class="playlist-empty">
            <p>播放列表为空</p>
            <p class="hint">在文件列表中点击 + 按钮添加</p>
        </div>
    {:else}
        <div class="playlist-list">
            {#each playlist.items as item, i (item.url)}
                <!-- svelte-ignore a11y_no_static_element_interactions, a11y_no_noninteractive_tabindex, a11y_no_noninteractive_element_interactions -->
                <div
                    class="playlist-item"
                    class:dragging={dragIndex === i}
                    class:drag-over={overIndex === i && dragIndex !== i}
                    draggable="true"
                    ondragstart={(e) => onDragStart(e, i)}
                    ondragover={(e) => onDragOver(e, i)}
                    ondrop={(e) => onDrop(e, i)}
                    ondragend={onDragEnd}
                    onclick={() => playItem(item, i)}
                    onkeydown={(e) => { if (e.key === 'Enter') playItem(item, i); }}
                    tabindex="0"
                    role="listitem"
                >
                    <span class="drag-handle" title="拖拽排序">⠿</span>
                    <span class="item-index">{i + 1}</span>
                    <div class="item-info">
                        <div class="item-name">{item.name}</div>
                        {#if item.source}
                            <div class="item-source">{item.source}</div>
                        {/if}
                    </div>
                    <span class="item-type">{item.mediaType === "audio" ? "🎵" : "🎬"}</span>
                    <button
                        type="button"
                        class="btn-remove"
                        title="移除"
                        onclick={(e) => remove(e, i)}
                    >✕</button>
                </div>
            {/each}
        </div>
    {/if}
</aside>

<style>
    .playlist-backdrop {
        position: fixed;
        inset: 0;
        background: rgba(0, 0, 0, 0.4);
        z-index: 90;
    }

    .playlist-panel {
        position: fixed;
        top: 0;
        right: 0;
        bottom: 0;
        width: min(380px, 85vw);
        background: var(--color-bg, #1a1a2e);
        z-index: 100;
        display: flex;
        flex-direction: column;
        box-shadow: -4px 0 24px rgba(0, 0, 0, 0.3);
        animation: slide-in 0.2s ease-out;
    }

    @keyframes slide-in {
        from { transform: translateX(100%); }
        to { transform: translateX(0); }
    }

    .playlist-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: var(--space-md);
        border-bottom: 1px solid var(--color-border);
        flex-shrink: 0;
    }

    .playlist-title {
        margin: 0;
        font-size: var(--font-size-base);
        font-weight: 700;
    }

    .playlist-actions {
        display: flex;
        align-items: center;
        gap: var(--space-sm);
    }

    .btn-text {
        background: none;
        border: none;
        color: var(--color-text-muted);
        cursor: pointer;
        font-size: var(--font-size-sm);
        padding: var(--space-xs) var(--space-sm);
        border-radius: var(--radius-md);
    }
    .btn-text:hover {
        background: var(--color-border);
        color: var(--color-text);
    }

    .btn-close {
        background: none;
        border: none;
        color: var(--color-text-muted);
        cursor: pointer;
        font-size: 1.1rem;
        padding: var(--space-xs);
        line-height: 1;
    }

    .playlist-empty {
        flex: 1;
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        color: var(--color-text-muted);
        gap: var(--space-xs);
    }
    .playlist-empty p {
        margin: 0;
    }
    .hint {
        font-size: var(--font-size-sm);
        opacity: 0.7;
    }

    .playlist-list {
        list-style: none;
        margin: 0;
        padding: 0;
        overflow-y: auto;
        flex: 1;
    }

    .playlist-item {
        display: flex;
        align-items: center;
        gap: var(--space-sm);
        padding: var(--space-sm) var(--space-md);
        border-bottom: 1px solid var(--color-border);
        cursor: pointer;
        transition: background 0.12s;
        user-select: none;
    }
    .playlist-item:hover {
        background: var(--color-border);
    }
    .playlist-item.dragging {
        opacity: 0.4;
    }
    .playlist-item.drag-over {
        border-top: 2px solid var(--color-accent, #6c63ff);
    }

    .drag-handle {
        cursor: grab;
        color: var(--color-text-muted);
        font-size: 1.1rem;
        flex-shrink: 0;
        opacity: 0.5;
    }
    .drag-handle:active {
        cursor: grabbing;
    }

    .item-index {
        font-size: var(--font-size-sm);
        color: var(--color-text-muted);
        min-width: 1.5em;
        text-align: center;
        flex-shrink: 0;
    }

    .item-info {
        flex: 1;
        min-width: 0;
    }

    .item-name {
        font-weight: 500;
        overflow: hidden;
        white-space: nowrap;
        text-overflow: ellipsis;
    }

    .item-source {
        font-size: var(--font-size-xs);
        color: var(--color-text-muted);
        overflow: hidden;
        white-space: nowrap;
        text-overflow: ellipsis;
    }

    .item-type {
        flex-shrink: 0;
        font-size: var(--font-size-sm);
    }

    .btn-remove {
        background: none;
        border: none;
        color: var(--color-text-muted);
        cursor: pointer;
        font-size: var(--font-size-sm);
        padding: var(--space-xs);
        border-radius: var(--radius-md);
        opacity: 0;
        transition: opacity 0.12s;
        flex-shrink: 0;
    }
    .playlist-item:hover .btn-remove {
        opacity: 1;
    }
    .btn-remove:hover {
        background: var(--color-error-bg, rgba(255,0,0,0.1));
        color: var(--color-error, #f44);
    }
</style>
