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
  let fileByKey = $state({});
  let dirByKey = $state({});
  let parentByDirKey = $state({});

  let selected = $state(null);
  /** @type {AbortController | null} */
  let ac = null;
  let firstQueryRun = true;

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

    // directory
    if (next?.ds) u.searchParams.set(QS.dirSource, next.ds);
    else u.searchParams.delete(QS.dirSource);
    if (next?.dp) u.searchParams.set(QS.dirRelPath, next.dp);
    else u.searchParams.delete(QS.dirRelPath);

    // player
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
        // root/virtual-root 也会是 dir；这里允许 source 为空
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
    // 用 parentByDirKey 回溯到根，再反转
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

    // 1) 恢复目录（仅在非搜索状态下）
    if (!q?.trim()) {
      if (ds) {
        const targetKey = makeKey(ds, dp);
        const chain = buildNavToDir(targetKey);
        if (chain.length) {
          nav = chain;
        } else {
          nav = [treeRoot];
          // URL 指向不存在的目录时，清理掉，避免一直“恢复失败”
          writeURLState({ ds: "", dp: "", ps, pp }, { replace });
        }
      } else if (treeRoot && (!nav?.length || nav[0] !== treeRoot)) {
        nav = [treeRoot];
      }
    }

    // 2) 恢复播放器
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
    // 把播放状态写进 URL，刷新/返回时可恢复
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

  async function fetchTree(query) {
    if (ac) ac.abort();
    ac = new AbortController();
    loading = true;
    error = "";
    const previousDirKey = currentDir
      ? makeKey(currentDir?.source || "", currentDir?.relPath || "")
      : "";
    try {
      const url =
        "/api/tree" + (query ? `?q=${encodeURIComponent(query)}` : "");
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

  // 监听浏览器前进/后退：按 URL 恢复目录/播放器
  $effect(() => {
    const onPop = () => syncStateFromURL({ replace: true });
    window.addEventListener("popstate", onPop);
    return () => window.removeEventListener("popstate", onPop);
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
            （{loading ? "加载中…" : q?.trim() ? "搜索结果" : "目录树，自动忽略压缩包"}）
          </span>
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
