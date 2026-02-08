<script>
  import Topbar from "./components/Topbar.svelte";
  import Player from "./components/Player.svelte";
  import Breadcrumb from "./components/Breadcrumb.svelte";
  import FileList from "./components/FileList.svelte";
  import Skeleton from "./components/Skeleton.svelte";
  import Playlist from "./components/Playlist.svelte";
  import { playlist, getNextItem } from "./playlist.svelte.js";

  let q = $state("");
  let loading = $state(false);
  let error = $state("");

  // Tree API data
  let treeRoot = $state(null);
  let nav = $state([]);
  let nodeByURL = $state({});

  let selected = $state(null);
  let notice = $state("");
  let showPlaylist = $state(false);

  /** @type {AbortController | null} */
  let ac = null;
  let firstQueryRun = true;

  function indexTree(root) {
    const map = {};
    function walk(node) {
      if (node.type === "file" && node.url) {
        map[node.url] = node;
      }
      if (node.children) {
        for (const c of node.children) walk(c);
      }
    }
    walk(root);
    return map;
  }

  function openPlayer(node) {
    selected = node;
    showPlaylist = false;
  }

  function closePlayer() {
    selected = null;
  }

  function onPlayerEnded() {
    if (!selected) return;
    const next = getNextItem(selected.url);
    if (next) {
      selected = next;
    } else {
      selected = null;
    }
  }

  function enterDir(node) {
    nav = [...nav, node];
  }

  function goToCrumb(index) {
    nav = nav.slice(0, index + 1);
  }

  // "name" | "modTime"
  let sortBy = $state("name");

  let currentDir = $derived(nav.length ? nav[nav.length - 1] : null);
  let currentChildren = $derived(
    currentDir?.children?.slice().sort((a, b) => {
      const ta = a.type === "dir" ? 0 : 1;
      const tb = b.type === "dir" ? 0 : 1;
      if (ta !== tb) return ta - tb;
      if (sortBy === "modTime") {
        return (b.modTime || 0) - (a.modTime || 0);
      }
      return a.name.localeCompare(b.name, "zh-Hans-CN");
    }) || [],
  );

  async function fetchTree(query) {
    if (ac) ac.abort();
    ac = new AbortController();
    loading = true;
    error = "";
    try {
      const url =
        "/api/tree" + (query ? `?q=${encodeURIComponent(query)}` : "");
      const res = await fetch(url, { signal: ac.signal });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      treeRoot = data.root;
      nodeByURL = indexTree(treeRoot);

      // 每次拿到新树都重置 nav 到根节点（搜索结果也是一棵新树）
      if (treeRoot) {
        nav = [treeRoot];
      }
    } catch (e) {
      if (e.name !== "AbortError") {
        error = String(e?.message || e);
      }
    } finally {
      loading = false;
    }
  }

  async function extractZip(node) {
    notice = `正在解压 ${node.name}…`;
    try {
      const payload = { source: node.source, relPath: node.relPath };
      const res = await fetch("/api/zip/extract", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      if (!res.ok) {
        const j = await res.json().catch(() => ({ error: res.statusText }));
        throw new Error(j?.error || res.statusText);
      }
      notice = "解压成功，正在刷新…";
      await fetchTree(q);
      notice = "";
    } catch (e) {
      notice = `解压失败: ${e?.message || e}`;
      setTimeout(() => (notice = ""), 3000);
    }
  }

  // 搜索防抖
  $effect(() => {
    const query = q;
    if (firstQueryRun) {
      firstQueryRun = false;
      fetchTree(query);
      return;
    }
    const t = setTimeout(() => fetchTree(query), 250);
    return () => clearTimeout(t);
  });
</script>

<div class="app">
  <Topbar bind:searchQuery={q} />

  {#if selected}
    <Player item={selected} onClose={closePlayer} onEnded={onPlayerEnded} />
  {:else}
    <main class="content">
      <section class="section">
        <div class="section-header">
          <div class="section-title">
            文件浏览
            <span class="muted">
              （{loading ? "加载中…" : q?.trim() ? "搜索结果" : "目录树"}）
            </span>
          </div>
          <div class="sort-toggle">
            <button
              type="button"
              class="sort-btn"
              class:active={sortBy === "name"}
              onclick={() => (sortBy = "name")}
            >名称</button>
            <button
              type="button"
              class="sort-btn"
              class:active={sortBy === "modTime"}
              onclick={() => (sortBy = "modTime")}
            >最近修改</button>
          </div>
        </div>

        {#if notice}
          <div class="notice">{notice}</div>
        {/if}

        {#if error}
          <div class="error">{error}</div>
        {/if}

        {#if loading}
          <Skeleton count={5} />
        {:else if treeRoot}
          <Breadcrumb items={nav} onNavigate={goToCrumb} />

          {#if currentDir}
            <div class="path">
              当前目录：{nav.map((x) => x.name).join(" / ")}
            </div>
            <FileList
              items={currentChildren}
              onEnterDir={enterDir}
              onPlayFile={openPlayer}
              onExtractZip={extractZip}
            />
          {/if}
        {:else}
          <div class="empty">未加载到目录树数据</div>
        {/if}
      </section>
    </main>
  {/if}

  {#if playlist.items.length > 0 && !showPlaylist}
    <button
      type="button"
      class="playlist-fab"
      onclick={() => (showPlaylist = true)}
    >
      ▶ {playlist.items.length}
    </button>
  {/if}

  {#if showPlaylist}
    <Playlist onPlay={openPlayer} onClose={() => (showPlaylist = false)} />
  {/if}
</div>

<style>
  .app {
    min-height: 100svh;
  }

  .content {
    padding: var(--space-md);
    display: grid;
    gap: var(--space-lg);
  }

  .section {
    display: grid;
    gap: var(--space-sm);
  }
  .section-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    flex-wrap: wrap;
    gap: var(--space-sm);
  }
  .section-title {
    font-weight: 700;
    margin: 6px 0 0;
  }
  .muted {
    font-weight: 400;
    color: var(--color-text-muted);
  }
  .sort-toggle {
    display: flex;
    gap: 2px;
    background: var(--color-border);
    border-radius: var(--radius-md);
    padding: 2px;
  }
  .sort-btn {
    padding: var(--space-xs) var(--space-sm);
    border: none;
    border-radius: calc(var(--radius-md) - 2px);
    background: transparent;
    color: var(--color-text-muted);
    font-size: var(--font-size-sm);
    cursor: pointer;
    transition: background 0.15s, color 0.15s;
  }
  .sort-btn.active {
    background: var(--color-bg-card, #fff);
    color: var(--color-text);
    font-weight: 600;
    box-shadow: 0 1px 2px rgba(0,0,0,.08);
  }

  .path {
    font-size: var(--font-size-sm);
    color: var(--color-text-muted);
    margin-bottom: var(--space-xs);
  }

  .notice {
    padding: var(--space-sm) var(--space-md);
    background: var(--color-notice-bg);
    border-radius: var(--radius-md);
    font-size: var(--font-size-base);
  }
  .error {
    padding: var(--space-sm) var(--space-md);
    background: var(--color-error-bg);
    border-radius: var(--radius-md);
    color: var(--color-error);
  }

  .empty {
    padding: var(--space-md) var(--space-sm);
    color: var(--color-text-muted);
  }

  .playlist-fab {
    position: fixed;
    bottom: var(--space-lg, 24px);
    right: var(--space-lg, 24px);
    z-index: 80;
    display: flex;
    align-items: center;
    gap: var(--space-xs);
    padding: var(--space-sm) var(--space-md);
    border: 1px solid var(--color-border-strong, #555);
    border-radius: 999px;
    background: var(--color-bg-card, #222);
    color: var(--color-text, #fff);
    font-weight: 700;
    font-size: var(--font-size-base);
    cursor: pointer;
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.3);
    transition: transform 0.15s, box-shadow 0.15s;
  }
  .playlist-fab:hover {
    transform: scale(1.05);
    box-shadow: 0 6px 20px rgba(0, 0, 0, 0.4);
  }
</style>
