<script>
    /**
     * @typedef {Object} MediaItem
     * @property {string} name
     * @property {string} url
     * @property {string} [mediaType]
     * @property {string} [source]
     * @property {string} [relPath]
     */

    /** @type {{ item: MediaItem, onClose: () => void }} */
    let { item, onClose } = $props();

    import { clearProgress, getProgress, makeMediaId, setProgress } from "../progress.js";

    /** @type {HTMLVideoElement | HTMLAudioElement | null} */
    let mediaEl = $state(null);
    let mediaId = $derived(makeMediaId(item));
    // 这些变量不参与视图渲染，只作为运行期标记即可（保持轻量）
    let trackedId = "";
    let restoredOnce = false;
    let lastWriteAt = 0;

    let resumeToast = $state("");

    function syncForMediaId() {
        if (trackedId === mediaId) return;
        trackedId = mediaId;
        restoredOnce = false;
        lastWriteAt = 0;
        resumeToast = "";
    }

    /** @param {HTMLVideoElement | HTMLAudioElement} node */
    function mediaRef(node) {
        mediaEl = node;
        return {
            destroy() {
                if (mediaEl === node) mediaEl = null;
            },
        };
    }

    function tryRestore() {
        syncForMediaId();
        if (!mediaEl || !mediaId || restoredOnce) return;
        const p = getProgress(mediaId);
        if (!p || p.ended) {
            restoredOnce = true;
            return;
        }
        const d = Number.isFinite(mediaEl.duration) ? mediaEl.duration : Number(p.d) || 0;
        // 接近片尾就不恢复（避免一打开就结束）
        if (d > 0 && d - p.t < 10) {
            restoredOnce = true;
            return;
        }
        const safeT = Math.max(0, Math.min(Number(p.t) || 0, d > 1 ? d - 1 : Number(p.t) || 0));
        if (safeT > 1) {
            mediaEl.currentTime = safeT;
            resumeToast = `已从 ${formatTime(safeT)} 继续播放`;
            setTimeout(() => {
                // 轻量：不做复杂的计时取消；内容变了也无所谓
                resumeToast = "";
            }, 2000);
        }
        restoredOnce = true;
    }

    function onTimeUpdate() {
        syncForMediaId();
        if (!mediaEl || !mediaId) return;
        const now = Date.now();
        if (now - lastWriteAt < 2000) return; // 2s 节流
        lastWriteAt = now;
        const t = mediaEl.currentTime || 0;
        const d = Number.isFinite(mediaEl.duration) ? mediaEl.duration : 0;
        if (d > 0 && d - t < 10) {
            clearProgress(mediaId);
            return;
        }
        if (t > 1) setProgress(mediaId, t, d, false);
    }

    function onEnded() {
        syncForMediaId();
        if (!mediaId) return;
        clearProgress(mediaId);
    }

    function formatTime(sec) {
        const s = Math.max(0, Math.floor(Number(sec) || 0));
        const hh = Math.floor(s / 3600);
        const mm = Math.floor((s % 3600) / 60);
        const ss = s % 60;
        const pad2 = (n) => String(n).padStart(2, "0");
        return hh > 0 ? `${hh}:${pad2(mm)}:${pad2(ss)}` : `${mm}:${pad2(ss)}`;
    }
</script>

<main class="player">
    <div class="player-head">
        <button type="button" class="btn" onclick={onClose}>返回</button>
        <div class="player-title">{item.name}</div>
    </div>
    {#if item.mediaType === "audio"}
        <div class="audio-container">
            <div class="audio-icon">🎵</div>
            <audio
                class="audio"
                use:mediaRef
                src={item.url}
                controls
                preload="metadata"
                onloadedmetadata={tryRestore}
                ontimeupdate={onTimeUpdate}
                onended={onEnded}
            ></audio>
        </div>
    {:else}
        <div class="video-wrap">
            <video
                class="video"
                use:mediaRef
                src={item.url}
                controls
                playsinline
                preload="metadata"
                onloadedmetadata={tryRestore}
                ontimeupdate={onTimeUpdate}
                onended={onEnded}
            >
                <track kind="captions" />
            </video>
            {#if resumeToast}
                <div class="toast">{resumeToast}</div>
            {/if}
        </div>
    {/if}
    <div class="hint">
        <a href={item.url} target="_blank" rel="noreferrer">在新窗口打开</a>
    </div>
</main>

<style>
    .player {
        padding: var(--space-md);
        display: grid;
        gap: var(--space-sm);
    }
    .player-head {
        display: grid;
        grid-template-columns: auto 1fr;
        gap: var(--space-sm);
        align-items: center;
    }
    .btn {
        padding: var(--space-sm) var(--space-md);
        border-radius: var(--radius-md);
        border: 1px solid var(--color-border-strong);
        background: var(--color-bg-card);
        color: inherit;
        cursor: pointer;
        box-shadow: var(--shadow-md);
    }
    .player-title {
        font-weight: 700;
        overflow: hidden;
        white-space: nowrap;
        text-overflow: ellipsis;
    }
    .video {
        width: 100%;
        border-radius: var(--radius-lg);
        background: #000;
        border: 1px solid var(--color-border-strong);
    }
    .video-wrap {
        position: relative;
        display: grid;
    }
    .toast {
        position: absolute;
        left: 12px;
        top: 12px;
        padding: 6px 10px;
        border-radius: 999px;
        background: rgba(0, 0, 0, 0.55);
        color: #fff;
        font-size: var(--font-size-sm);
        border: 1px solid rgba(255, 255, 255, 0.15);
        backdrop-filter: blur(8px);
        pointer-events: none;
        max-width: calc(100% - 24px);
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }
    .audio-container {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        padding: 40px 20px;
        background: var(--gradient-audio);
        border-radius: var(--radius-lg);
        border: 1px solid var(--color-border);
    }
    .audio-icon {
        font-size: var(--font-size-icon);
        margin-bottom: 20px;
    }
    .audio {
        width: 100%;
        max-width: 400px;
    }
    .hint {
        color: var(--color-text-muted);
        font-size: var(--font-size-sm);
    }
    .hint a {
        color: var(--color-primary);
        text-decoration: none;
    }
</style>
