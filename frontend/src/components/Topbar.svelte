<script>
    import logoUrl from "../assets/yseren-logo.svg";
    import { playlist } from "../playlist.svelte.js";
    import { dismissUpdate, updateState } from "../update.svelte.js";

    /** @type {{
     *   searchQuery?: string,
     *   onRefresh?: () => void,
     *   onTogglePlaylist?: () => void,
     *   refreshing?: boolean,
     *   registerSearchFocus?: (fn: () => void) => void
     * }} */
    let {
        searchQuery = $bindable(""),
        onRefresh,
        onTogglePlaylist,
        refreshing = false,
        registerSearchFocus,
    } = $props();

    /** @type {HTMLInputElement | null} */
    let searchInput = $state(null);

    $effect(() => {
        registerSearchFocus?.(() => searchInput?.focus());
    });

    function closeUpdateBanner() {
        if (updateState.info?.version) {
            dismissUpdate(updateState.info.version);
        }
    }
</script>

<header class="topbar">
    {#if updateState.info}
        <div class="update-banner" role="status">
            <span class="update-text">
                发现新版本 <strong>{updateState.info.tag}</strong>
            </span>
            <div class="update-actions">
                <a
                    class="update-link"
                    href={updateState.info.url}
                    target="_blank"
                    rel="noreferrer"
                >查看更新</a>
                <button type="button" class="update-dismiss" onclick={closeUpdateBanner} aria-label="关闭更新提示">
                    ✕
                </button>
            </div>
        </div>
    {/if}
    <div class="topbar-inner">
        <div class="brand">
            <img class="logo" src={logoUrl} alt="" aria-hidden="true" />
            <div class="title">
                <div class="name">YSeren</div>
                <div class="sub">局域网媒体</div>
            </div>
        </div>

        <div class="search">
            <input
                bind:this={searchInput}
                placeholder="搜索文件名或路径（按 / 聚焦）"
                bind:value={searchQuery}
            />
        </div>

        <div class="actions">
            {#if onRefresh}
                <button
                    type="button"
                    class="action-btn"
                    title="刷新目录（F5）"
                    aria-label="刷新目录"
                    disabled={refreshing}
                    onclick={onRefresh}
                >
                    <span class="action-icon" class:spinning={refreshing}>↻</span>
                    <span class="action-label">刷新</span>
                </button>
            {/if}
            {#if onTogglePlaylist}
                <button
                    type="button"
                    class="action-btn"
                    title="播放列表"
                    aria-label="播放列表"
                    onclick={onTogglePlaylist}
                >
                    <span class="action-icon">☰</span>
                    <span class="action-label">列表</span>
                    {#if playlist.items.length > 0}
                        <span class="badge">{playlist.items.length}</span>
                    {/if}
                </button>
            {/if}
        </div>
    </div>
</header>

<style>
    .topbar {
        position: sticky;
        top: 0;
        z-index: 10;
        background: var(--color-bg-overlay);
        backdrop-filter: blur(14px);
        border-bottom: 1px solid var(--color-border);
    }

    .update-banner {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: var(--space-md);
        padding: var(--space-sm) var(--space-md);
        background: var(--color-notice-bg);
        border-bottom: 1px solid rgba(91, 140, 255, 0.2);
        max-width: var(--content-max-width);
        margin: 0 auto;
    }
    .update-text {
        font-size: var(--font-size-sm);
        color: var(--color-text);
    }
    .update-actions {
        display: flex;
        align-items: center;
        gap: var(--space-sm);
        flex-shrink: 0;
    }
    .update-link {
        font-size: var(--font-size-sm);
        font-weight: 700;
        color: var(--color-primary);
        text-decoration: none;
        padding: 4px 10px;
        border-radius: 999px;
        border: 1px solid rgba(91, 140, 255, 0.35);
        background: rgba(255, 255, 255, 0.65);
    }
    .update-link:hover {
        background: #fff;
    }
    .update-dismiss {
        border: 0;
        background: transparent;
        color: var(--color-text-muted);
        cursor: pointer;
        font-size: var(--font-size-base);
        line-height: 1;
        padding: 4px;
    }

    .topbar-inner {
        display: grid;
        grid-template-columns: auto minmax(0, 1fr) auto;
        gap: var(--space-sm);
        align-items: center;
        padding: var(--space-md);
        max-width: var(--content-max-width);
        margin: 0 auto;
    }

    .brand {
        display: flex;
        gap: var(--space-sm);
        align-items: center;
        min-width: 0;
    }
    .logo {
        width: 36px;
        height: 36px;
        border-radius: var(--radius-md);
        display: block;
        flex: 0 0 auto;
    }
    .title .name {
        font-weight: 800;
        line-height: 1.05;
    }
    .title .sub {
        font-size: var(--font-size-sm);
        color: var(--color-text-muted);
        margin-top: 2px;
    }

    .search input {
        width: 100%;
        padding: var(--space-sm) var(--space-md);
        border-radius: var(--radius-md);
        border: 1px solid var(--color-border-strong);
        background: var(--color-bg-overlay);
        color: inherit;
        outline: none;
        transition: border-color var(--transition-fast), box-shadow var(--transition-fast);
    }
    .search input:focus {
        border-color: var(--color-primary);
        box-shadow: 0 0 0 3px rgba(91, 140, 255, 0.15);
    }
    .search input::placeholder {
        color: var(--color-text-placeholder);
    }
    .search {
        min-width: 0;
    }

    .actions {
        display: flex;
        gap: var(--space-xs);
        align-items: center;
    }

    .action-btn {
        position: relative;
        display: inline-flex;
        justify-content: center;
        align-items: center;
        gap: 0;
        width: 38px;
        min-width: 38px;
        height: 38px;
        padding: 0;
        border-radius: var(--radius-md);
        border: 1px solid var(--color-border-strong);
        background: var(--color-bg-card);
        color: inherit;
        cursor: pointer;
        font-size: var(--font-size-sm);
        font-weight: 600;
        box-shadow: var(--shadow-sm);
        transition: border-color var(--transition-fast), background var(--transition-fast);
    }
    .action-btn:hover:not(:disabled) {
        border-color: var(--color-border-hover);
        background: var(--color-notice-bg);
    }
    .action-btn:disabled {
        opacity: 0.6;
        cursor: wait;
    }

    .action-icon {
        font-size: var(--font-size-lg);
        line-height: 1;
    }
    .action-icon.spinning {
        animation: spin 0.8s linear infinite;
    }
    .action-label {
        display: none;
    }

    .badge {
        position: absolute;
        top: -6px;
        right: -6px;
        min-width: 18px;
        height: 18px;
        padding: 0 5px;
        border-radius: 999px;
        background: var(--color-primary);
        color: #fff;
        font-size: 10px;
        font-weight: 700;
        display: grid;
        place-items: center;
    }

    @keyframes spin {
        to { transform: rotate(360deg); }
    }

    @media (min-width: 768px) {
        .topbar-inner {
            grid-template-columns: auto 1fr auto;
            gap: var(--space-md);
        }
        .actions {
            gap: var(--space-sm);
        }
        .action-btn {
            width: auto;
            min-width: 0;
            height: auto;
            gap: 6px;
            padding: var(--space-sm) var(--space-md);
        }
        .action-label {
            display: inline;
        }
    }
</style>
