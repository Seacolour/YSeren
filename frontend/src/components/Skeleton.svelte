<script>
    /** @type {{ count?: number }} */
    let { count = 3 } = $props();
</script>

<div class="skeleton-list">
    {#each Array(count) as _, i (i)}
        <div class="skeleton-row">
            <div class="skeleton-icon"></div>
            <div class="skeleton-content">
                <div class="skeleton-title"></div>
                <div class="skeleton-sub"></div>
            </div>
        </div>
    {/each}
</div>

<style>
    .skeleton-list {
        display: grid;
        gap: var(--space-sm);
    }

    .skeleton-row {
        display: grid;
        grid-template-columns: auto 1fr;
        gap: var(--space-sm);
        align-items: center;
        padding: var(--space-sm);
        border: 1px solid var(--color-border);
        background: var(--color-bg-card);
        border-radius: var(--radius-lg);
    }

    .skeleton-icon {
        width: 34px;
        height: 34px;
        border-radius: var(--radius-md);
        background: linear-gradient(
            90deg,
            var(--color-bg-subtle) 25%,
            var(--color-border) 50%,
            var(--color-bg-subtle) 75%
        );
        background-size: 200% 100%;
        animation: shimmer 1.5s infinite;
    }

    .skeleton-content {
        display: grid;
        gap: var(--space-xs);
    }

    .skeleton-title {
        height: 16px;
        width: 60%;
        border-radius: var(--radius-sm);
        background: linear-gradient(
            90deg,
            var(--color-bg-subtle) 25%,
            var(--color-border) 50%,
            var(--color-bg-subtle) 75%
        );
        background-size: 200% 100%;
        animation: shimmer 1.5s infinite;
    }

    .skeleton-sub {
        height: 12px;
        width: 40%;
        border-radius: var(--radius-sm);
        background: linear-gradient(
            90deg,
            var(--color-bg-subtle) 25%,
            var(--color-border) 50%,
            var(--color-bg-subtle) 75%
        );
        background-size: 200% 100%;
        animation: shimmer 1.5s infinite;
        animation-delay: 0.2s;
    }

    @keyframes shimmer {
        0% {
            background-position: 200% 0;
        }
        100% {
            background-position: -200% 0;
        }
    }
</style>
