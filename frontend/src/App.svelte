<script>
  import Topbar from "./components/Topbar.svelte";
  import Player from "./components/Player.svelte";
  import Breadcrumb from "./components/Breadcrumb.svelte";
  import FileList from "./components/FileList.svelte";
  import Skeleton from "./components/Skeleton.svelte";

  let q = $state("");
  let loading = $state(false);
  let error = $state("");

  // Tree API data
  let treeRoot = $state(null);
  let nav = $state([]);
  let nodeByURL = $state({});

  let selected = $state(null);
  let notice = $state("");

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
  }

  function closePlayer() {
    selected = null;
  }

  function enterDir(node) {
    nav = [...nav, node];
  }

  function goToCrumb(index) {
    nav = nav.slice(0, index + 1);
  }

  let currentDir = $derived(nav.length ? nav[nav.length - 1] : null);
  let currentChildren = $derived(
    currentDir?.children?.slice().sort((a, b) => {
      const ta = a.type === "dir" ? 0 : 1;
      const tb = b.type === "dir" ? 0 : 1;
      return ta - tb || a.name.localeCompare(b.name, "zh-Hans-CN");
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

      // 搜索时不改 nav；首屏时若 treeRoot 是虚拟根目录，则展开到那里
      if (!query && treeRoot) {
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
    <Player item={selected} onClose={closePlayer} />
  {:else}
    <main class="content">
      <section class="section">
        <div class="section-title">
          文件浏览
          <span class="muted">
            （{loading ? "加载中…" : q?.trim() ? "搜索结果" : "目录树"}）
          </span>
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
  .section-title {
    font-weight: 700;
    margin: 6px 0 0;
  }
  .muted {
    font-weight: 400;
    color: var(--color-text-muted);
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
</style>
