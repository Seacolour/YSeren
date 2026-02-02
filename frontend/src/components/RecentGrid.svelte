<script>
    /**
     * @typedef {Object} RecentItem
     * @property {string} name
     * @property {string} url
     * @property {string} [source]
     * @property {string} [relPath]
     */

    /** @type {{ items: RecentItem[], onSelect: (item: RecentItem) => void }} */
    let { items, onSelect } = $props();
</script>

{#if items.length}
    <section class="section">
        <div class="section-title">最近播放</div>
        <div class="grid">
            {#each items as r (r.url)}
                <button type="button" class="card" onclick={() => onSelect(r)}>
                    <div class="thumb">
                        <div class="badge">{r.source || "recent"}</div>
                    </div>
                    <div class="meta">
                        <div class="name">{r.name}</div>
                        <div class="sub">{r.relPath || ""}</div>
                    </div>
                </button>
            {/each}
        </div>
    </section>
{/if}

<style>
    .section-title {
        font-weight: 700;
        margin: 6px 0 var(--space-sm);
    }

    .grid {
        display: grid;
        grid-template-columns: repeat(2, minmax(0, 1fr));
        gap: var(--space-sm);
    }
    @media (min-width: 540px) {
        .grid {
            grid-template-columns: repeat(3, minmax(0, 1fr));
        }
    }

    .card {
        text-align: left;
        border: 1px solid var(--color-border);
        border-radius: var(--radius-lg);
        background: var(--color-bg-card);
        box-shadow: var(--shadow-md);
        overflow: hidden;
        padding: 0;
        color: inherit;
        cursor: pointer;
    }
    .thumb {
        aspect-ratio: 16 / 9;
        background: var(--gradient-thumb);
        position: relative;
    }
    .badge {
        position: absolute;
        left: var(--space-sm);
        top: var(--space-sm);
        font-size: var(--font-size-xs);
        padding: var(--space-xs) var(--space-sm);
        border-radius: 999px;
        background: var(--color-bg-overlay);
        border: 1px solid var(--color-border-strong);
    }
    .meta {
        padding: var(--space-sm) var(--space-sm) var(--space-md);
        display: grid;
        gap: var(--space-xs);
    }
    .meta .name {
        font-weight: 650;
        line-height: 1.2;
        overflow: hidden;
        display: -webkit-box;
        -webkit-line-clamp: 2;
        -webkit-box-orient: vertical;
    }
    .meta .sub {
        font-size: var(--font-size-sm);
        color: var(--color-text-muted);
        overflow: hidden;
        white-space: nowrap;
        text-overflow: ellipsis;
    }
</style>
