<script>
    import logoUrl from "../assets/yseren-logo.svg";

    /** @type {string} */
    let { searchQuery = $bindable("") } = $props();
</script>

<header class="topbar">
    <div class="brand">
        <img class="logo" src={logoUrl} alt="" aria-hidden="true" />
        <div class="title">
            <div class="name">YSeren</div>
            <div class="sub">局域网媒体</div>
        </div>
    </div>

    <div class="search">
        <input placeholder="搜索：文件名 / 路径" bind:value={searchQuery} />
    </div>
</header>

<style>
    .topbar {
        position: sticky;
        top: 0;
        z-index: 10;
        display: grid;
        grid-template-columns: auto 1fr;
        gap: var(--space-md);
        align-items: center;
        padding: var(--space-md);
        background: var(--color-bg-overlay);
        backdrop-filter: blur(14px);
        border-bottom: 1px solid var(--color-border);
    }

    .brand {
        display: flex;
        gap: var(--space-sm);
        align-items: center;
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
    }
    .search input::placeholder {
        color: var(--color-text-placeholder);
    }
</style>
