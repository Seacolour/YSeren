<script>
    /**
     * @typedef {Object} MediaItem
     * @property {string} name
     * @property {string} url
     * @property {string} [mediaType]
     * @property {string} [source]
     * @property {string} [relPath]
     */

    /** @type {{ item: MediaItem, onClose: () => void, onEnded?: () => void }} */
    let { item, onClose, onEnded: onEndedCallback } = $props();

    import { clearProgress, getProgress, makeMediaId, setProgress } from "../progress.svelte.js";
    import {
        PLAYBACK_RATES,
        formatPlaybackRate,
        playbackState,
        setPlaybackRate,
    } from "../playback.svelte.js";
    import { formatTime } from "../utils.js";

    /** @type {HTMLVideoElement | HTMLAudioElement | null} */
    let mediaEl = $state(null);
    let mediaId = $derived(makeMediaId(item));
    let trackedId = "";
    let restoredOnce = false;
    let lastWriteAt = 0;

    let resumeToast = $state("");

    function applyPlaybackRate() {
        if (mediaEl) mediaEl.playbackRate = playbackState.rate;
    }

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
        applyPlaybackRate();
        return {
            destroy() {
                if (mediaEl === node) mediaEl = null;
            },
        };
    }

    function tryRestore() {
        syncForMediaId();
        applyPlaybackRate();
        if (!mediaEl || !mediaId || restoredOnce) return;
        const p = getProgress(mediaId);
        if (!p || p.ended) {
            restoredOnce = true;
            return;
        }
        const d = Number.isFinite(mediaEl.duration) ? mediaEl.duration : Number(p.d) || 0;
        if (d > 0 && d - p.t < 10) {
            restoredOnce = true;
            return;
        }
        const safeT = Math.max(0, Math.min(Number(p.t) || 0, d > 1 ? d - 1 : Number(p.t) || 0));
        if (safeT > 1) {
            mediaEl.currentTime = safeT;
            resumeToast = `已从 ${formatTime(safeT)} 继续播放`;
            setTimeout(() => {
                resumeToast = "";
            }, 2000);
        }
        restoredOnce = true;
    }

    function onTimeUpdate() {
        syncForMediaId();
        if (!mediaEl || !mediaId) return;
        const now = Date.now();
        if (now - lastWriteAt < 2000) return;
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
        onEndedCallback?.();
    }

    /** @param {number} rate */
    function selectRate(rate) {
        setPlaybackRate(rate);
        applyPlaybackRate();
    }

    $effect(() => {
        const rate = playbackState.rate;
        if (mediaEl) mediaEl.playbackRate = rate;
    });
</script>

<main class="player">
    <div class="player-head">
        <button type="button" class="btn" onclick={onClose}>返回 <span class="kbd-hint">Esc</span></button>
        <div class="player-title">{item.name}</div>
    </div>

    <div class="speed-bar" role="group" aria-label="播放倍速">
        <span class="speed-label">倍速</span>
        <div class="speed-options">
            {#each PLAYBACK_RATES as rate (rate)}
                <button
                    type="button"
                    class="speed-btn"
                    class:active={playbackState.rate === rate}
                    aria-pressed={playbackState.rate === rate}
                    onclick={() => selectRate(rate)}
                >
                    {formatPlaybackRate(rate)}
                </button>
            {/each}
        </div>
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
    @media (min-width: 768px) {
        .player {
            padding: var(--space-lg) var(--space-xl);
            gap: var(--space-md);
        }
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

    .speed-bar {
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        gap: var(--space-sm);
        padding: var(--space-sm);
        border: 1px solid var(--color-border);
        border-radius: var(--radius-lg);
        background: var(--color-bg-card);
        box-shadow: var(--shadow-sm);
    }
    .speed-label {
        font-size: var(--font-size-sm);
        font-weight: 700;
        color: var(--color-text-muted);
        flex-shrink: 0;
    }
    .speed-options {
        display: flex;
        flex-wrap: wrap;
        gap: 6px;
    }
    .speed-btn {
        padding: 5px 10px;
        border-radius: 999px;
        border: 1px solid var(--color-border-strong);
        background: var(--color-bg-subtle);
        color: inherit;
        font-size: var(--font-size-sm);
        font-weight: 600;
        cursor: pointer;
        transition: border-color var(--transition-fast), background var(--transition-fast), color var(--transition-fast);
    }
    .speed-btn:hover {
        border-color: var(--color-border-hover);
        background: var(--color-notice-bg);
    }
    .speed-btn.active {
        border-color: var(--color-primary);
        background: var(--color-primary);
        color: #fff;
    }

    .video {
        width: 100%;
        max-width: var(--player-max-width);
        margin: 0 auto;
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
        max-width: var(--player-max-width);
        margin: 0 auto;
        width: 100%;
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
    .kbd-hint {
        display: none;
        margin-left: 6px;
        padding: 1px 6px;
        border-radius: 4px;
        border: 1px solid var(--color-border-strong);
        background: var(--color-bg-subtle);
        font-size: 10px;
        font-weight: 600;
        color: var(--color-text-muted);
    }
    @media (min-width: 768px) {
        .kbd-hint {
            display: inline;
        }
    }
</style>
