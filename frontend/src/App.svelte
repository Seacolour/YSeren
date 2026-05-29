<script>
  import { onMount } from "svelte";
  import Topbar from "./components/Topbar.svelte";
  import Player from "./components/Player.svelte";
  import Playlist from "./components/Playlist.svelte";
  import Breadcrumb from "./components/Breadcrumb.svelte";
  import FileList from "./components/FileList.svelte";
  import Skeleton from "./components/Skeleton.svelte";
  import { getNextItem } from "./playlist.svelte.js";
  import { checkForUpdate } from "./update.svelte.js";

  let q = $state("");
  let loading = $state(false);
  let refreshing = $state(false);
  let error = $state("");
  let showPlaylist = $state(false);

  let treeRoot = $state(null);
  let nav = $state([]);
  let fileByKey = $state({});
  let dirByKey = $state({});
  let parentByDirKey = $state({});

  let selected = $state(null);
  /** @type {AbortController | null} */
  let ac = null;
  let firstQueryRun = true;
  /** @type {(() => void) | null} */
  let focusSearch = null;

  const QS = Object.freeze({
    dirSource: "ds",
    dirRelPath: "dp",
    playSource: "ps",
    playRelPath: "pp",
  });

  function makeKey(source, relPath) {
    return `${source || ""}::${relPath || ""}`;
  }

  function readURLState() {
    const u = new URL(window.location.href);
    const ds = u.searchParams.get(QS.dirSource) || "";
    const dp = u.searchParams.get(QS.dirRelPath) || "";
    const ps = u.searchParams.get(QS.playSource) || "";
    const pp = u.searchParams.get(QS.playRelPath) || "";
    return { ds, dp, ps, pp };
  }

  function writeURLState(next, { replace = false } = {}) {
    const u = new URL(window.location.href);

    if (next?.ds) u.searchParams.set(QS.dirSource, next.ds);
    else u.searchParams.delete(QS.dirSource);
    if (next?.dp) u.searchParams.set(QS.dirRelPath, next.dp);
    else u.searchParams.delete(QS.dirRelPath);

    if (next?.ps) u.searchParams.set(QS.playSource, next.ps);
    else u.searchParams.delete(QS.playSource);
    if (next?.pp) u.searchParams.set(QS.playRelPath, next.pp);
    else u.searchParams.delete(QS.playRelPath);

    const url = u.pathname + (u.searchParams.toString() ? `?${u.searchParams.toString()}` : "") + u.hash;
    if (replace) history.replaceState(null, "", url);
    else history.pushState(null, "", url);
  }

  function indexTree(root) {
    const files = {};
    const dirs = {};
    const parents = {};

    function walk(node, parent) {
      if (node?.type === "file" && node?.source && node?.relPath) {
        files[makeKey(node.source, node.relPath)] = node;
      }
      if (node?.type === "dir") {
        const k = makeKey(node?.source || "", node?.relPath || "");
        dirs[k] = node;
        if (parent?.type === "dir") {
          const pk = makeKey(parent?.source || "", parent?.relPath || "");
          parents[k] = pk;
        }
      }
      if (node?.children) {
        for (const c of node.children) walk(c, node);
      }
    }

    walk(root, null);
    return { files, dirs, parents };
  }

  function buildNavToDir(targetKey) {
    const chain = [];
    let k = targetKey;
    const seen = new Set();
    while (k && !seen.has(k)) {
      seen.add(k);
      const n = dirByKey[k];
      if (n) chain.push(n);
      k = parentByDirKey[k];
    }
    chain.reverse();
    return chain;
  }

  function syncNavForTree(query, previousDirKey = "") {
    if (!treeRoot) return;

    if (query?.trim()) {
      const chain = previousDirKey ? buildNavToDir(previousDirKey) : [];
      nav = chain.length ? chain : [treeRoot];
      return;
    }

    syncStateFromURL({ replace: true });
    if (!nav?.length) nav = [treeRoot];
  }

  function syncStateFromURL({ replace = true } = {}) {
    if (!treeRoot) return;
    const { ds, dp, ps, pp } = readURLState();

    if (!q?.trim()) {
      if (ds) {
        const targetKey = makeKey(ds, dp);
        const chain = buildNavToDir(targetKey);
        if (chain.length) {
          nav = chain;
        } else {
          nav = [treeRoot];
          writeURLState({ ds: "", dp: "", ps, pp }, { replace });
        }
      } else if (treeRoot && (!nav?.length || nav[0] !== treeRoot)) {
        nav = [treeRoot];
      }
    }

    if (ps && pp) {
      const node = fileByKey[makeKey(ps, pp)];
      if (node) selected = node;
      else if (selected) selected = null;
    } else if (selected) {
      selected = null;
    }
  }

  function openPlayer(node) {
    selected = node;
    showPlaylist = false;
    if (node?.source && node?.relPath) {
      const { ds, dp } = readURLState();
      writeURLState({ ds, dp, ps: node.source, pp: node.relPath });
    }
  }

  function closePlayer() {
    selected = null;
    const { ds, dp } = readURLState();
    writeURLState({ ds, dp, ps: "", pp: "" });
  }

  function playNextInPlaylist() {
    if (!selected?.url) return;
    const next = getNextItem(selected.url);
    if (next) openPlayer(next);
  }

  function enterDir(node) {
    nav = [...nav, node];
    if (!q?.trim()) {
      const { ps, pp } = readURLState();
      writeURLState(
        { ds: node?.source || "", dp: node?.relPath || "", ps, pp },
        { replace: true },
      );
    }
  }

  function goToCrumb(index) {
    nav = nav.slice(0, index + 1);
    if (!q?.trim()) {
      const node = nav[index];
      const { ps, pp } = readURLState();
      writeURLState(
        { ds: node?.source || "", dp: node?.relPath || "", ps, pp },
        { replace: true },
      );
    }
  }

  let currentDir = $derived(nav.length ? nav[nav.length - 1] : null);
  let currentChildren = $derived(
    currentDir?.children?.slice().sort((a, b) => {
      const ta = a.type === "dir" ? 0 : 1;
      const tb = b.type === "dir" ? 0 : 1;
      return ta - tb || a.name.localeCompare(b.name, "zh-Hans-CN");
    }) || [],
  );

  let sectionHint = $derived(
    loading ? "加载中…" : q?.trim() ? "搜索结果" : "目录树，自动忽略压缩包",
  );

  async function fetchTree(query, { refresh = false } = {}) {
    if (ac) ac.abort();
    ac = new AbortController();
    loading = true;
    if (refresh) refreshing = true;
    error = "";
    const previousDirKey = currentDir
      ? makeKey(currentDir?.source || "", currentDir?.relPath || "")
      : "";
    try {
      const params = new URLSearchParams();
      if (query?.trim()) params.set("q", query.trim());
      if (refresh) params.set("refresh", "1");
      const qs = params.toString();
      const url = "/api/tree" + (qs ? `?${qs}` : "");
      const res = await fetch(url, { signal: ac.signal });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      treeRoot = data.root;
      const idx = indexTree(treeRoot);
      fileByKey = idx.files;
      dirByKey = idx.dirs;
      parentByDirKey = idx.parents;
      syncNavForTree(query, previousDirKey);
    } catch (e) {
      if (e.name !== "AbortError") {
        error = String(e?.message || e);
      }
    } finally {
      loading = false;
      refreshing = false;
    }
  }

  function handleRefresh() {
    fetchTree(q, { refresh: true });
  }

  function togglePlaylist() {
    showPlaylist = !showPlaylist;
  }

  function handleKeydown(e) {
    const tag = e.target?.tagName?.toLowerCase();
    const inInput = tag === "input" || tag === "textarea" || e.target?.isContentEditable;

    if (e.key === "Escape") {
      if (selected) {
        e.preventDefault();
        closePlayer();
      } else if (showPlaylist) {
        e.preventDefault();
        showPlaylist = false;
      }
      return;
    }

    if (e.key === "F5" && !inInput) {
      e.preventDefault();
      handleRefresh();
      return;
    }

    if (e.key === "/" && !inInput) {
      e.preventDefault();
      focusSearch?.();
    }
  }

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

  onMount(() => {
    checkForUpdate();
  });

  $effect(() => {
    const onPop = () => syncStateFromURL({ replace: true });
    window.addEventListener("popstate", onPop);
    return () => window.removeEventListener("popstate", onPop);
  });

  $effect(() => {
    window.addEventListener("keydown", handleKeydown);
    return () => window.removeEventListener("keydown", handleKeydown);
  });
</script>

<div class="app">
  <Topbar
    bind:searchQuery={q}
    onRefresh={handleRefresh}
    onTogglePlaylist={togglePlaylist}
    {refreshing}
    registerSearchFocus={(fn) => { focusSearch = fn; }}
  />

  {#if selected}
    <div class="player-shell">
      <Player item={selected} onClose={closePlayer} onEnded={playNextInPlaylist} />
    </div>
  {:else}
    <main class="content">
      <section class="section">
        <div class="section-head">
          <div class="section-title">
            文件浏览
            <span class="muted">（{sectionHint}）</span>
          </div>
          {#if !loading && currentChildren.length > 0}
            <span class="item-count">{currentChildren.length} 项</span>
          {/if}
        </div>

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
            />
          {/if}
        {:else}
          <div class="empty">未加载到目录树数据</div>
        {/if}
      </section>
    </main>
  {/if}

  {#if showPlaylist}
    <Playlist
      onPlay={openPlayer}
      onClose={() => { showPlaylist = false; }}
    />
  {/if}
</div>

<style>
  .app {
    min-height: 100svh;
  }

  .content,
  .player-shell {
    max-width: var(--content-max-width);
    margin: 0 auto;
    width: 100%;
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

  .section-head {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: var(--space-md);
    flex-wrap: wrap;
  }

  .section-title {
    font-weight: 700;
    margin: 6px 0 0;
  }
  .muted {
    font-weight: 400;
    color: var(--color-text-muted);
  }

  .item-count {
    font-size: var(--font-size-sm);
    color: var(--color-text-muted);
    padding: 4px 10px;
    border-radius: 999px;
    background: var(--color-bg-subtle);
    border: 1px solid var(--color-border);
  }

  .path {
    font-size: var(--font-size-sm);
    color: var(--color-text-muted);
    margin-bottom: var(--space-xs);
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

  @media (min-width: 768px) {
    .content {
      padding: var(--space-lg) var(--space-xl);
    }
  }
</style>
