<script>
    /**
     * @typedef {Object} NavItem
     * @property {string} name
     * @property {string} [source]
     * @property {string} [relPath]
     */

    /** @type {{ items: NavItem[], onNavigate: (index: number) => void }} */
    let { items, onNavigate } = $props();
</script>

<div class="crumbs">
    {#each items as c, i (c.source ? `${c.source}:${c.relPath}` : c.relPath)}
        <button
            type="button"
            class="crumb"
            onclick={() => onNavigate(i)}
            disabled={i === items.length - 1}
        >
            {c.name}
        </button>
        {#if i < items.length - 1}
            <span class="sep">/</span>
        {/if}
    {/each}
</div>

<style>
    .crumbs {
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        gap: 6px;
        padding: var(--space-sm);
        border: 1px solid var(--color-border);
        background: var(--color-bg-card);
        border-radius: var(--radius-lg);
        box-shadow: var(--shadow-sm);
        margin-bottom: var(--space-sm);
    }
    .crumb {
        border: 0;
        background: transparent;
        padding: 6px var(--space-sm);
        border-radius: var(--radius-sm);
        cursor: pointer;
        font-weight: 650;
        color: inherit;
    }
    .crumb:disabled {
        cursor: default;
        opacity: 0.8;
    }
    .crumb:not(:disabled):hover {
        background: var(--color-notice-bg);
    }
    .sep {
        color: var(--color-text-muted);
    }
</style>
