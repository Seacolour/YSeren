<script>
  import { onMount } from 'svelte';
  import * as Backend from '../wailsjs/go/main/App.js';
  import { EventsOff, EventsOn } from '../wailsjs/runtime/runtime.js';

  const emptyState = {
    status: { state: 'stopped', urls: [] },
    sources: [],
    port: 1479,
    logLevel: 'info',
    preferences: {
      minimizeToTray: true,
      startSharingOnLaunch: true,
      launchAtStartup: false
    },
    config: { path: '', mode: 'user', exists: false },
    version: 'dev',
    firstRun: true,
    canStart: false,
    loadError: ''
  };

  let state = $state(emptyState);
  let page = $state('share');
  let loading = $state(true);
  let busy = $state(false);
  let errorMessage = $state('');
  let toast = $state('');
  let toastTimer;
  let portDraft = $state(1479);
  let editingIndex = $state(-1);
  let editName = $state('');
  let editPath = $state('');
  let removingIndex = $state(-1);

  let statusName = $derived.by(() => {
    switch (state.status?.state) {
      case 'running': return '共享运行中';
      case 'starting': return '正在启动';
      case 'stopping': return '正在停止';
      case 'failed': return '启动失败';
      default: return '共享已停止';
    }
  });
  let running = $derived(state.status?.state === 'running');
  let urls = $derived(state.status?.urls ?? []);
  let localURL = $derived(urls[0] ?? '');
  let lanURLs = $derived(urls.slice(1));

  onMount(() => {
    void refreshState();
    EventsOn('desktop:state', (nextState) => applyState(nextState));
    return () => {
      EventsOff('desktop:state');
      if (toastTimer) clearTimeout(toastTimer);
    };
  });

  function applyState(nextState) {
    if (!nextState) return;
    state = nextState;
    portDraft = nextState.port;
  }

  function backendMethod(name) {
    const method = Backend[name];
    if (typeof method !== 'function') {
      throw new Error(`Desktop 绑定尚未生成：${name}`);
    }
    return method;
  }

  function readableError(error) {
    if (!error) return '操作失败';
    if (typeof error === 'string') return error;
    return error.message || String(error);
  }

  function showToast(message) {
    toast = message;
    if (toastTimer) clearTimeout(toastTimer);
    toastTimer = setTimeout(() => { toast = ''; }, 2400);
  }

  async function refreshState() {
    try {
      applyState(await backendMethod('GetState')());
    } catch (error) {
      errorMessage = readableError(error);
    } finally {
      loading = false;
    }
  }

  async function run(action, successMessage = '') {
    if (busy) return null;
    busy = true;
    errorMessage = '';
    try {
      const result = await action();
      if (result?.status) applyState(result);
      if (successMessage) showToast(successMessage);
      return result;
    } catch (error) {
      errorMessage = readableError(error);
      return null;
    } finally {
      busy = false;
    }
  }

  async function chooseAndAdd(openAfterStart = false) {
    const path = await run(() => backendMethod('ChooseDirectory')());
    if (!path) return;
    const nextState = await run(() => backendMethod('AddSource')(path), '媒体目录已添加');
    if (openAfterStart && nextState?.status?.state === 'running') {
      await run(() => backendMethod('OpenBrowser')(''));
    }
  }

  function openEditor(index) {
    const source = state.sources[index];
    editingIndex = index;
    editName = source.name;
    editPath = source.path;
  }

  async function chooseEditPath() {
    const path = await run(() => backendMethod('ChooseDirectory')());
    if (path) editPath = path;
  }

  async function saveSource() {
    const result = await run(
      () => backendMethod('UpdateSource')(editingIndex, editName, editPath),
      '媒体源已更新'
    );
    if (result) editingIndex = -1;
  }

  async function removeSource() {
    const result = await run(
      () => backendMethod('RemoveSource')(removingIndex),
      '媒体源已移除'
    );
    if (result) removingIndex = -1;
  }

  async function savePort(event) {
    event.preventDefault();
    await run(() => backendMethod('SetPort')(Number(portDraft)), '端口设置已保存');
  }

  async function savePreferences(minimizeToTray, startSharingOnLaunch) {
    await run(
      () => backendMethod('UpdatePreferences')(minimizeToTray, startSharingOnLaunch),
      '偏好设置已保存'
    );
  }

  async function setLaunchAtStartup(enabled) {
    await run(() => backendMethod('SetLaunchAtStartup')(enabled), enabled ? '已启用开机启动' : '已关闭开机启动');
  }

  async function copyAddress(address) {
    await run(() => backendMethod('CopyAddress')(address), '地址已复制');
  }

  async function openAddress(address) {
    await run(() => backendMethod('OpenBrowser')(address));
  }
</script>

<div class="app-shell">
  <aside class="sidebar">
    <div class="brand">
      <img src="/yseren-logo.svg" alt="" />
      <div><strong>YSeren</strong><span>局域网媒体</span></div>
    </div>

    <nav aria-label="主导航">
      <button class:active={page === 'share'} onclick={() => { page = 'share'; }}>
        <span class="nav-icon">⌁</span><span>共享</span>
      </button>
      <button class:active={page === 'sources'} onclick={() => { page = 'sources'; }}>
        <span class="nav-icon">▱</span><span>媒体源</span>
        {#if state.sources.length}<span class="nav-count">{state.sources.length}</span>{/if}
      </button>
      <button class:active={page === 'settings'} onclick={() => { page = 'settings'; }}>
        <span class="nav-icon">⚙</span><span>设置</span>
      </button>
    </nav>

    <div class="sidebar-note"><span class="shield">✓</span><p>仅在你信任的局域网中共享媒体。</p></div>
    <div class="version">{state.version && state.version !== 'dev' ? `v${state.version}` : '开发版本'}</div>
  </aside>

  <main class="main-content">
    {#if page === 'share'}
      <header class="page-header">
        <div>
          <p class="eyebrow">共享</p>
          <h1>让另一台设备直接访问这里的媒体</h1>
          <p>文件保留在当前电脑，不上传，也不额外复制。</p>
        </div>
      </header>

      {#if state.firstRun}
        <section class="onboarding-card">
          <div class="sparkle">✦</div>
          <div class="onboarding-copy">
            <span class="badge">首次使用</span>
            <h2>选择一个媒体目录，就可以开始</h2>
            <p>YSeren 只会公开目录中可识别的音视频文件，不会显示普通文档或原始目录列表。</p>
            <div class="steps" aria-label="首次运行步骤">
              <span><b>1</b> 选择目录</span><i></i><span><b>2</b> 开始共享</span><i></i><span><b>3</b> 浏览器打开</span>
            </div>
            <button class="primary large" disabled={busy} onclick={() => chooseAndAdd(true)}>
              <span>＋</span> 选择媒体目录并开始共享
            </button>
          </div>
        </section>
      {:else}
        <section class="status-card" class:running>
          <div class="status-top">
            <div class="status-identity">
              <span class="status-dot"></span>
              <div><p>当前状态</p><h2>{statusName}</h2></div>
            </div>
            {#if running}
              <button class="secondary danger-text" disabled={busy} onclick={() => run(() => backendMethod('StopSharing')())}>■ 停止共享</button>
            {:else}
              <button class="primary" disabled={busy || !state.canStart} onclick={() => run(() => backendMethod('StartSharing')())}>▶ 开始共享</button>
            {/if}
          </div>

          {#if state.status?.lastError}<div class="inline-alert error">{state.status.lastError}</div>{/if}

          {#if running}
            <div class="address-grid">
              <article class="address-card local">
                <span class="address-type">本机地址</span><strong>{localURL}</strong>
                <div class="address-actions">
                  <button aria-label="复制本机地址" title="复制地址" onclick={() => copyAddress(localURL)}>⧉</button>
                  <button aria-label="在浏览器打开本机地址" title="打开浏览器" onclick={() => openAddress(localURL)}>↗</button>
                </div>
              </article>
              {#each lanURLs as address (address)}
                <article class="address-card lan">
                  <span class="address-type">局域网地址</span><strong>{address}</strong>
                  <div class="address-actions">
                    <button aria-label="复制局域网地址" title="复制地址" onclick={() => copyAddress(address)}>⧉</button>
                    <button aria-label="在浏览器打开局域网地址" title="打开浏览器" onclick={() => openAddress(address)}>↗</button>
                  </div>
                </article>
              {:else}
                <article class="address-card unavailable">
                  <span class="address-type">局域网地址</span><strong>暂未检测到可用的内网 IPv4</strong>
                  <small>检查网络连接后重启共享即可刷新。</small>
                </article>
              {/each}
            </div>
            <button class="open-browser" onclick={() => openAddress('')} disabled={busy}>在默认浏览器中打开 <span>↗</span></button>
          {:else}
            <div class="stopped-hint"><span>▶</span><p>共享停止时，其他设备无法访问媒体。你的文件不会被移动或修改。</p></div>
          {/if}
        </section>

        <section class="summary-card">
          <div><span class="summary-icon">▱</span><div><strong>{state.sources.length} 个媒体源</strong><p>{state.sources.map((source) => source.name).join('、')}</p></div></div>
          <button class="text-button" onclick={() => { page = 'sources'; }}>管理媒体源 →</button>
        </section>
      {/if}

    {:else if page === 'sources'}
      <header class="page-header with-action">
        <div><p class="eyebrow">媒体源</p><h1>选择要共享的目录</h1><p>目录名会显示在浏览器中，你可以随时修改名称或路径。</p></div>
        <button class="primary" disabled={busy} onclick={() => chooseAndAdd(false)}>＋ 添加目录</button>
      </header>

      <section class="source-list">
        {#each state.sources as source, index (`${source.name}-${source.path}`)}
          <article class="source-card" class:unavailable={!source.available}>
            <div class="folder-icon">▰</div>
            <div class="source-info">
              <div class="source-title"><strong>{source.name}</strong>{#if source.available}<span class="health good">可访问</span>{:else}<span class="health bad">不可用</span>{/if}</div>
              <p title={source.path}>{source.path}</p>
              {#if source.error}<small>{source.error}</small>{/if}
            </div>
            <div class="row-actions">
              <button aria-label={`编辑 ${source.name}`} title="编辑" onclick={() => openEditor(index)}>✎</button>
              <button class="delete" aria-label={`移除 ${source.name}`} title="移除" onclick={() => { removingIndex = index; }}>×</button>
            </div>
          </article>
        {:else}
          <div class="empty-state">
            <div>▱</div><h2>还没有媒体源</h2><p>添加视频或音乐所在的目录，YSeren 就能把它变成局域网页面。</p>
            <button class="primary" onclick={() => chooseAndAdd(false)}>＋ 选择媒体目录</button>
          </div>
        {/each}
      </section>
      <div class="privacy-note"><span>✓</span><p><strong>共享边界</strong><br />只允许访问配置目录内已识别的媒体文件；非媒体文件、目录列表与路径穿越请求会被拒绝。</p></div>

    {:else}
      <header class="page-header">
        <div><p class="eyebrow">设置</p><h1>Desktop 偏好与网络</h1><p>常用选项保持简单，YAML 仍可用于高级配置与 Headless 版本。</p></div>
      </header>

      <div class="settings-stack">
        <section class="settings-card network-settings">
          <div class="settings-heading"><div><h2>网络</h2><p>修改端口后，运行中的共享会自动重启。</p></div></div>
          <form class="setting-row" onsubmit={savePort}>
            <label for="port">服务端口</label>
            <div class="port-control"><input id="port" type="number" min="1" max="65535" bind:value={portDraft} /><button class="secondary" type="submit" disabled={busy || Number(portDraft) === state.port}>保存</button></div>
          </form>
        </section>

        <section class="settings-card behavior-settings">
          <div class="settings-heading"><div><h2>应用行为</h2><p>托盘会保留共享服务，退出则会停止服务。</p></div></div>
          <label class="toggle-row">
            <span><strong>关闭窗口时最小化到托盘</strong><small>仍可通过托盘打开、启停共享或退出。</small></span>
            <input type="checkbox" checked={state.preferences.minimizeToTray} onchange={(event) => savePreferences(event.currentTarget.checked, state.preferences.startSharingOnLaunch)} />
          </label>
          <label class="toggle-row">
            <span><strong>启动应用时自动共享</strong><small>仅当至少有一个可访问的媒体源时启动。</small></span>
            <input type="checkbox" checked={state.preferences.startSharingOnLaunch} onchange={(event) => savePreferences(state.preferences.minimizeToTray, event.currentTarget.checked)} />
          </label>
          <label class="toggle-row">
            <span><strong>登录 Windows 后启动 YSeren</strong><small>后台启动并隐藏主窗口，可从托盘唤回。</small></span>
            <input type="checkbox" checked={state.preferences.launchAtStartup} onchange={(event) => setLaunchAtStartup(event.currentTarget.checked)} />
          </label>
        </section>

        <section class="settings-card config-settings">
          <div class="settings-heading"><div><h2>配置文件</h2><p>{state.config.mode === 'portable' ? 'Portable 模式' : state.config.mode === 'explicit' ? '显式配置' : '用户配置'}</p></div></div>
          <div class="config-path" title={state.config.path}>{state.config.path || '尚未确定配置路径'}</div>
          <div class="config-actions">
            <button class="secondary" disabled={busy} onclick={() => run(() => backendMethod('ImportConfig')(), '配置已导入')}>导入 YAML</button>
            <button class="secondary" disabled={busy} onclick={() => run(() => backendMethod('ExportConfig')(), '配置已导出')}>导出 YAML</button>
          </div>
        </section>
      </div>
    {/if}

    {#if state.loadError || errorMessage}
      <div class="global-alert" role="alert"><span>!</span><p>{errorMessage || state.loadError}</p><button aria-label="关闭错误提示" onclick={() => { errorMessage = ''; }}>×</button></div>
    {/if}
    {#if toast}<div class="toast" role="status">✓ {toast}</div>{/if}
    {#if loading}<div class="loading"><span></span><p>正在读取 YSeren 状态…</p></div>{/if}
  </main>
</div>

{#if editingIndex >= 0}
  <div class="modal-backdrop" role="presentation">
    <div class="modal" role="dialog" aria-modal="true" aria-labelledby="edit-title">
      <div class="modal-header"><div><p class="eyebrow">媒体源</p><h2 id="edit-title">编辑目录</h2></div><button aria-label="关闭" onclick={() => { editingIndex = -1; }}>×</button></div>
      <label>显示名称<input maxlength="80" bind:value={editName} /></label>
      <label>目录路径<div class="path-input"><input bind:value={editPath} /><button class="secondary" onclick={chooseEditPath}>选择…</button></div></label>
      <div class="modal-actions"><button class="text-button" onclick={() => { editingIndex = -1; }}>取消</button><button class="primary" disabled={busy || !editName.trim() || !editPath.trim()} onclick={saveSource}>保存并应用</button></div>
    </div>
  </div>
{/if}

{#if removingIndex >= 0}
  <div class="modal-backdrop" role="presentation">
    <div class="modal compact" role="alertdialog" aria-modal="true" aria-labelledby="remove-title">
      <div class="remove-mark">×</div><h2 id="remove-title">移除“{state.sources[removingIndex]?.name}”？</h2>
      <p>这不会删除磁盘上的文件。运行中的共享会立即更新，目录将不再可访问。</p>
      <div class="modal-actions"><button class="text-button" onclick={() => { removingIndex = -1; }}>取消</button><button class="danger" disabled={busy} onclick={removeSource}>移除媒体源</button></div>
    </div>
  </div>
{/if}
