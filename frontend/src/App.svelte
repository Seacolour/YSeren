<script>
  let q = $state('');
  let loading = $state(false);
  let error = $state('');

  function loadRecent() {
    try {
      const raw = localStorage.getItem('lvlink:recent');
      const arr = raw ? JSON.parse(raw) : [];
      return Array.isArray(arr) ? arr : [];
    } catch {
      return [];
    }
  }

  function saveRecent(arr) {
    try {
      localStorage.setItem('lvlink:recent', JSON.stringify(arr.slice(0, 30)));
    } catch {
      // ignore
    }
  }

  function formatSize(bytes) {
    if (!Number.isFinite(bytes)) return '';
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    let v = bytes;
    let i = 0;
    while (v >= 1024 && i < units.length - 1) {
      v /= 1024;
      i++;
    }
    const n = i === 0 ? String(Math.round(v)) : v.toFixed(v >= 10 ? 1 : 2);
    return `${n} ${units[i]}`;
  }

  // Tree API data
  let treeRoot = $state(null); // TreeNode
  let nav = $state([]); // TreeNode[] breadcrumb path, last = current dir
  let nodeByURL = $state({}); // url -> TreeNode(file)

  let selected = $state(null); // TreeNode(file)
  let recent = $state(loadRecent());
  let notice = $state('');

  /** @type {AbortController | null} */
  let ac = null;
  let firstQueryRun = true;

  function indexTree(root) {
    const m = Object.create(null);
    const byURL = Object.create(null);
    const walk = (node) => {
      if (!node) return;
      const key = node.source ? `${node.source}:${node.relPath || ''}` : `:${node.relPath || ''}`;
      m[key] = node;
      if (node.type === 'file' && node.url) byURL[node.url] = node;
      if (Array.isArray(node.children)) {
        for (const c of node.children) walk(c);
      }
    };
    walk(root);
    nodeByURL = byURL;
    return m;
  }

  const currentDir = $derived(nav.length ? nav[nav.length - 1] : null);
  const currentChildren = $derived(currentDir ? listChildren(currentDir) : []);

  function resetNav() {
    if (!treeRoot) {
      nav = [];
      return;
    }
    nav = [treeRoot];
  }

  function enterDir(node) {
    if (!node || node.type !== 'dir') return;
    // 防止重复进入同一目录导致 breadcrumb 叠加
    if (currentDir && node.source === currentDir.source && (node.relPath || '') === (currentDir.relPath || '')) return;
    nav = [...nav, node];
  }

  function goToCrumb(idx) {
    if (idx < 0 || idx >= nav.length) return;
    nav = nav.slice(0, idx + 1);
  }

  function listChildren(node) {
    const arr = Array.isArray(node?.children) ? node.children : [];
    // dir -> zip -> file, then name
    return [...arr].sort((a, b) => {
      if (a.type !== b.type) {
        const rank = (t) => (t === 'dir' ? 0 : t === 'zip' ? 1 : 2);
        return rank(a.type) - rank(b.type);
      }
      return (a.name || '').localeCompare(b.name || '');
    });
  }

  async function fetchTree(query) {
    loading = true;
    error = '';
    notice = '';

    if (ac) ac.abort();
    ac = new AbortController();

    const params = new URLSearchParams();
    if (query && query.trim()) params.set('q', query.trim());

    try {
      const res = await fetch(`/api/tree?${params.toString()}`, { signal: ac.signal });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      treeRoot = data.root || null;
      const idx = indexTree(treeRoot);

      // 尝试恢复“当前位置”（仅在非搜索模式下）
      const prev = currentDir;
      if (!query?.trim() && prev?.source) {
        const key = `${prev.source}:${prev.relPath || ''}`;
        const found = idx[key];
        if (found) {
          // 重新构建 nav：从 root 逐级找回（靠 relPath 分割）
          const parts = (found.relPath || '').split('/').filter(Boolean);
          let cur = treeRoot;
          const nextNav = [treeRoot];
          if (found.source) {
            // 先定位 source root
            const srcRoot = (treeRoot?.children || []).find((c) => c.type === 'dir' && c.source === found.source);
            if (srcRoot) {
              cur = srcRoot;
              nextNav.push(srcRoot);
            }
          }
          for (const seg of parts) {
            const child = (cur?.children || []).find((c) => c.type === 'dir' && c.name === seg);
            if (!child) break;
            nextNav.push(child);
            cur = child;
          }
          nav = nextNav;
        } else {
          resetNav();
        }
      } else {
        resetNav();
      }
    } catch (e) {
      if (e?.name === 'AbortError') return;
      error = `加载失败：${e?.message || e}`;
    } finally {
      loading = false;
    }
  }

  async function extractZip(node) {
    notice = '';
    error = '';
    try {
      const res = await fetch('/api/zip/extract', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ source: node.source, relPath: node.relPath })
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) throw new Error(data?.message || `HTTP ${res.status}`);

      if (data.status === 'exists') {
        notice = `解压目录已存在：${data.extractedRelDir || ''}`;
      } else if (data.status === 'ok') {
        notice = `解压完成：${data.extractedRelDir || ''}`;
      } else {
        throw new Error(data?.message || '解压失败');
      }

      // 强制刷新树（后端有 5s cache）
      const params = new URLSearchParams();
      if (q && q.trim()) params.set('q', q.trim());
      params.set('refresh', '1');
      const treeRes = await fetch(`/api/tree?${params.toString()}`);
      if (treeRes.ok) {
        const treeData = await treeRes.json();
        treeRoot = treeData.root || null;
        indexTree(treeRoot);
        // 保持现有 nav（不重置），用户自己选择进入新目录
      }
    } catch (e) {
      error = `解压失败：${e?.message || e}`;
    }
  }

  function openPlayer(item) {
    selected = item;
    const entry = {
      source: item.source,
      name: item.name,
      relPath: item.relPath,
      url: item.url,
      at: Date.now()
    };
    const next = [entry, ...recent.filter((r) => r.url !== item.url)];
    recent = next.slice(0, 30);
    saveRecent(recent);
    history.pushState({}, '', `/#/play?u=${encodeURIComponent(item.url)}`);
  }

  function closePlayer() {
    selected = null;
    history.pushState({}, '', '/');
  }

  function syncFromLocation() {
    try {
      const hash = location.hash || '';
      if (!hash.startsWith('#/play')) return;
      const qs = hash.split('?')[1] || '';
      const u = new URLSearchParams(qs).get('u');
      if (!u) return;
      const url = decodeURIComponent(u);
      // 如果 tree 里有就用 tree 节点（带 name/source/relPath）
      const found = nodeByURL[url];
      selected = found || { name: url.split('/').pop(), url };
    } catch {
      // ignore
    }
  }

  // 初始化：只跑一次（恢复最近播放 + hash 播放 + popstate）
  $effect(() => {
    syncFromLocation();
    window.addEventListener('popstate', syncFromLocation);
    return () => window.removeEventListener('popstate', syncFromLocation);
  });

  // 搜索防抖：首屏立即加载，其余 q 变化走 250ms 防抖
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
  <header class="topbar">
    <div class="brand">
      <div class="logo">lv</div>
      <div class="title">
        <div class="name">lv-link</div>
        <div class="sub">局域网视频</div>
      </div>
    </div>

    <div class="search">
      <input placeholder="搜索：文件名 / 路径" bind:value={q} />
    </div>
  </header>

  {#if selected}
    <main class="player">
      <div class="player-head">
        <button type="button" class="btn" onclick={closePlayer}>返回</button>
        <div class="player-title">{selected.name}</div>
      </div>
      <video class="video" src={selected.url} controls playsinline preload="metadata">
        <track kind="captions" />
      </video>
      <div class="hint">
        <a href={selected.url} target="_blank" rel="noreferrer">在新窗口打开</a>
      </div>
    </main>
  {:else}
    <main class="content">
      {#if recent.length}
        <section class="section">
          <div class="section-title">最近播放</div>
          <div class="grid">
            {#each recent as r (r.url)}
              <button type="button" class="card" onclick={() => openPlayer(r)}>
                <div class="thumb">
                  <div class="badge">{r.source || 'recent'}</div>
                </div>
                <div class="meta">
                  <div class="name">{r.name}</div>
                  <div class="sub">{r.relPath || ''}</div>
                </div>
              </button>
            {/each}
          </div>
        </section>
      {/if}

      <section class="section">
        <div class="section-title">
          文件浏览
          <span class="muted">（{loading ? '加载中…' : q?.trim() ? '搜索结果' : '目录树' }）</span>
        </div>

        {#if notice}
          <div class="notice">{notice}</div>
        {/if}

        {#if error}
          <div class="error">{error}</div>
        {/if}

        {#if treeRoot}
          <div class="crumbs">
            {#each nav as c, i (c.source ? `${c.source}:${c.relPath}` : c.relPath)}
              <button
                type="button"
                class="crumb"
                onclick={() => goToCrumb(i)}
                disabled={i === nav.length - 1}
              >
                {c.name}
              </button>
              {#if i < nav.length - 1}
                <span class="sep">/</span>
              {/if}
            {/each}
          </div>

          {#if currentDir}
            <div class="path">
              当前目录：{nav.map((x) => x.name).join(' / ')}
            </div>
            <div class="list">
              {#each currentChildren as it (it.type === 'file' ? it.url : `${it.source}:${it.relPath}`)}
                {#if it.type === 'dir'}
                  <button type="button" class="row" onclick={() => enterDir(it)}>
                    <div class="icon folder">📁</div>
                    <div class="row-main">
                      <div class="row-title">{it.name}</div>
                      <div class="row-sub">{it.relPath || it.source || ''}</div>
                    </div>
                    <div class="chev">›</div>
                  </button>
                {:else if it.type === 'zip'}
                  <div class="row zip">
                    <div class="icon folder">📦</div>
                    <div class="row-main">
                      <div class="row-title">{it.name}</div>
                      <div class="row-sub">{it.relPath}</div>
                    </div>
                    <button
                      type="button"
                      class="zip-btn"
                      onclick={(e) => {
                        e.stopPropagation();
                        extractZip(it);
                      }}
                    >
                      解压
                    </button>
                  </div>
                {:else}
                  <button type="button" class="row" onclick={() => openPlayer(it)}>
                    <div class="icon file">▶</div>
                    <div class="row-main">
                      <div class="row-title">{it.name}</div>
                      <div class="row-sub">
                        {it.relPath}
                        {#if it.size}
                          <span class="dot">·</span>
                          {formatSize(it.size)}
                        {/if}
                      </div>
                    </div>
                  </button>
                {/if}
              {/each}
            </div>
          {/if}
        {:else}
          <div class="empty">未加载到目录树数据</div>
        {/if}
      </section>
    </main>
  {/if}
</div>

<style>
  :global(body) {
    margin: 0;
    background: #f6f7fb;
    color: #111827;
    font-family: ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, Arial, "PingFang SC", "Microsoft YaHei",
      sans-serif;
  }
  :global(*) {
    box-sizing: border-box;
  }

  .app {
    min-height: 100svh;
  }

  .topbar {
    position: sticky;
    top: 0;
    z-index: 10;
    display: grid;
    grid-template-columns: auto 1fr;
    gap: 12px;
    align-items: center;
    padding: 12px 12px;
    background: rgba(255, 255, 255, 0.86);
    backdrop-filter: blur(14px);
    border-bottom: 1px solid rgba(17, 24, 39, 0.08);
  }

  .brand {
    display: flex;
    gap: 10px;
    align-items: center;
  }
  .logo {
    width: 36px;
    height: 36px;
    border-radius: 12px;
    display: grid;
    place-items: center;
    background: linear-gradient(135deg, #5b8cff, #9b5bff);
    font-weight: 800;
    letter-spacing: -0.5px;
    color: white;
  }
  .title .name {
    font-weight: 800;
    line-height: 1.05;
  }
  .title .sub {
    font-size: 12px;
    opacity: 0.7;
    margin-top: 2px;
  }

  .search input {
    width: 100%;
    padding: 10px 12px;
    border-radius: 12px;
    border: 1px solid rgba(17, 24, 39, 0.12);
    background: rgba(255, 255, 255, 0.9);
    color: inherit;
    outline: none;
  }
  .search input::placeholder {
    color: rgba(17, 24, 39, 0.45);
  }

  .content {
    padding: 12px 12px 28px;
    display: grid;
    gap: 18px;
  }

  .section-title {
    font-weight: 700;
    margin: 6px 0 10px;
  }
  .muted {
    opacity: 0.6;
    font-weight: 500;
    margin-left: 6px;
    font-size: 12px;
  }

  .grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 10px;
  }
  @media (min-width: 540px) {
    .grid {
      grid-template-columns: repeat(3, minmax(0, 1fr));
    }
  }

  .card {
    text-align: left;
    border: 1px solid rgba(17, 24, 39, 0.08);
    border-radius: 14px;
    background: white;
    box-shadow: 0 8px 24px rgba(17, 24, 39, 0.06);
    overflow: hidden;
    padding: 0;
    color: inherit;
    cursor: pointer;
  }
  .thumb {
    aspect-ratio: 16 / 9;
    background: radial-gradient(circle at 20% 30%, rgba(91, 140, 255, 0.25), rgba(155, 91, 255, 0.12) 45%, rgba(255, 255, 255, 0) 70%),
      linear-gradient(135deg, rgba(17, 24, 39, 0.04), rgba(17, 24, 39, 0.02));
    position: relative;
  }
  .badge {
    position: absolute;
    left: 10px;
    top: 10px;
    font-size: 11px;
    padding: 4px 8px;
    border-radius: 999px;
    background: rgba(255, 255, 255, 0.86);
    border: 1px solid rgba(17, 24, 39, 0.12);
  }
  .meta {
    padding: 10px 10px 12px;
    display: grid;
    gap: 4px;
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
    font-size: 12px;
    opacity: 0.7;
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
  }
  .dot {
    margin: 0 6px;
    opacity: 0.6;
  }

  .crumbs {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 6px;
    padding: 10px 10px;
    border: 1px solid rgba(17, 24, 39, 0.08);
    background: white;
    border-radius: 14px;
    box-shadow: 0 8px 24px rgba(17, 24, 39, 0.05);
    margin-bottom: 10px;
  }
  .crumb {
    border: 0;
    background: transparent;
    padding: 6px 8px;
    border-radius: 10px;
    cursor: pointer;
    font-weight: 650;
    color: inherit;
  }
  .crumb:disabled {
    cursor: default;
    opacity: 0.8;
  }
  .crumb:not(:disabled):hover {
    background: rgba(91, 140, 255, 0.12);
  }
  .sep {
    opacity: 0.45;
  }

  .list {
    display: grid;
    gap: 8px;
  }
  .path {
    margin: 0 0 10px;
    font-size: 12px;
    color: rgba(17, 24, 39, 0.6);
  }
  .row {
    display: grid;
    grid-template-columns: auto 1fr auto;
    align-items: center;
    gap: 10px;
    width: 100%;
    text-align: left;
    border: 1px solid rgba(17, 24, 39, 0.08);
    background: white;
    border-radius: 14px;
    padding: 10px 10px;
    box-shadow: 0 8px 24px rgba(17, 24, 39, 0.05);
    cursor: pointer;
    color: inherit;
  }
  .row.zip {
    cursor: default;
  }
  .zip-btn {
    border: 1px solid rgba(17, 24, 39, 0.12);
    background: rgba(255, 255, 255, 0.9);
    border-radius: 12px;
    padding: 8px 10px;
    font-weight: 700;
    cursor: pointer;
    box-shadow: 0 8px 24px rgba(17, 24, 39, 0.06);
  }
  .zip-btn:hover {
    border-color: rgba(91, 140, 255, 0.35);
  }
  .row:hover {
    border-color: rgba(91, 140, 255, 0.28);
  }
  .icon {
    width: 34px;
    height: 34px;
    border-radius: 12px;
    display: grid;
    place-items: center;
    font-size: 16px;
    border: 1px solid rgba(17, 24, 39, 0.08);
    background: rgba(17, 24, 39, 0.03);
  }
  .icon.file {
    font-weight: 900;
    color: #4f46e5;
  }
  .row-main {
    min-width: 0;
    display: grid;
    gap: 2px;
  }
  .row-title {
    font-weight: 700;
    line-height: 1.2;
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
  }
  .row-sub {
    font-size: 12px;
    opacity: 0.65;
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
  }
  .chev {
    opacity: 0.35;
    font-size: 18px;
    padding: 0 4px;
  }
  .empty {
    padding: 12px 10px;
    opacity: 0.7;
  }

  .player {
    padding: 12px;
    display: grid;
    gap: 10px;
  }
  .player-head {
    display: grid;
    grid-template-columns: auto 1fr;
    gap: 10px;
    align-items: center;
  }
  .btn {
    padding: 10px 12px;
    border-radius: 12px;
    border: 1px solid rgba(17, 24, 39, 0.12);
    background: white;
    color: inherit;
    cursor: pointer;
    box-shadow: 0 8px 24px rgba(17, 24, 39, 0.06);
  }
  .player-title {
    font-weight: 700;
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
  }
  .video {
    width: 100%;
    border-radius: 14px;
    background: #000;
    border: 1px solid rgba(17, 24, 39, 0.12);
  }
  .hint {
    opacity: 0.7;
    font-size: 12px;
  }
  .hint a {
    color: #2563eb;
    text-decoration: none;
  }
  .error {
    color: #7f1d1d;
    background: rgba(239, 68, 68, 0.12);
    border: 1px solid rgba(239, 68, 68, 0.25);
    padding: 10px 12px;
    border-radius: 12px;
    margin-bottom: 8px;
  }
  .notice {
    color: #0f3d1f;
    background: rgba(34, 197, 94, 0.12);
    border: 1px solid rgba(34, 197, 94, 0.25);
    padding: 10px 12px;
    border-radius: 12px;
    margin-bottom: 8px;
  }
</style>


