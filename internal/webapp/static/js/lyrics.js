// Lyrics follow the same authoritative room position as the player's progress.
import { state, on, playbackPosition, lyricsAPI } from './core.js?v=31';

export function normalizeLyrics(data) {
    if (data.unsupported) return { status: 'unsupported', lines: [] };
    let lines = [];
    if (data.synced === true) {
        lines = (Array.isArray(data.lines) ? data.lines : []).flatMap(line => {
            const time = Number(line.startTimeMs);
            return line.startTimeMs != null && line.startTimeMs !== '' && Number.isFinite(time) && time >= 0 && typeof line.words === 'string' ? [{ time: time / 1000, text: line.words }] : [];
        });
        if (!lines.some(line => line.text.trim()) && typeof data.lrc === 'string') {
            const offset = Number(data.lrc.match(/\[offset:([+-]?\d+)\]/i)?.[1] || 0) / 1000;
            lines = data.lrc.split(/\r?\n/).flatMap(row => {
                const stamps = [...row.matchAll(/\[(\d+):(\d{2})(?:[.:](\d{1,3}))?\]/g)];
                const text = row.replace(/\[[^\]]*\]/g, '').trim();
                return stamps.filter(match => Number(match[2]) < 60).map(match => ({ time: Math.max(0, Number(match[1]) * 60 + Number(match[2]) + Number(`0.${match[3] || 0}`) + offset), text }));
            });
        }
    }
    if (lines.some(line => line.text.trim())) return { status: 'synced', lines: lines.sort((a, b) => a.time - b.time) };
    const plain = typeof data.text === 'string' ? data.text.trim() : '';
    return { status: plain ? 'plain' : 'empty', lines: plain ? plain.split(/\r?\n/).map(text => ({ text })) : [] };
}

export function createLyrics({ player, toggle, stage, panel, status, lines: container, follow, preview, previewLine, headerTitle, headerSubtitle }) {
    let open = false, key = '', result = { status: 'idle', lines: [] }, active = -1, following = true;
    let controller = null, disposed = false;
    const cache = new Map();
    const visible = () => open && state.expandedPlayer && !disposed;
    const expanded = () => state.expandedPlayer && !disposed;
    const trackKey = () => state.room?.track ? `${state.room.track.id}|${state.room.track.url}` : '';
    function render() {
        player.classList.toggle('has-lyrics', visible());
        panel.hidden = !visible();
        stage.hidden = !visible();
        panel.dataset.state = result.status;
        toggle.disabled = !state.room?.track && !open;
        toggle.setAttribute('aria-pressed', String(open));
        toggle.setAttribute('aria-label', open ? 'Show artwork' : 'Show lyrics');
        toggle.title = open ? 'Show artwork' : 'Show lyrics';
        const hasPreview = expanded() && !open && ['synced', 'plain'].includes(result.status);
        preview.hidden = !hasPreview;
        player.classList.toggle('has-lyric-preview', hasPreview);
        preview.disabled = !state.room?.track;
        headerTitle.textContent = visible() ? state.room?.track?.title || 'Track lyrics' : 'Playing together';
        headerSubtitle.textContent = visible() ? state.room?.track?.artist || 'Music' : ({ spotify: 'Spotify', youtube: 'YouTube', soundcloud: 'SoundCloud' }[state.room?.track?.platform] || 'Shared listening');
        const messages = { idle: 'Choose a track to see lyrics.', loading: 'Loading lyrics…', empty: 'No lyrics available for this track.', unsupported: 'No lyrics available for this track.', plain: 'Timing isn’t available for these lyrics.', synced: 'Synced lyrics' };
        status.textContent = messages[result.status];
        status.hidden = result.status === 'synced';
        container.replaceChildren(...result.lines.map(line => {
            const node = document.createElement('p');
            node.className = 'lyric-line'; node.textContent = line.text || '…';
            return node;
        }));
        container.style.paddingBlock = result.status === 'synced' ? `${Math.max(24, panel.clientHeight / 2 - 32)}px` : '24px';
        active = -1; following = true; follow.hidden = true; panel.scrollTop = 0;
        update(playbackPosition(), true);
    }
    async function load() {
        if (!expanded() || result.status !== 'idle' || !key) return;
        if (cache.has(key)) { result = cache.get(key); render(); return; }
        result = { status: 'loading', lines: [] }; render();
        controller = new AbortController();
        const request = controller, requestedKey = key;
        try {
            const data = await lyricsAPI(state.room.track.url, request.signal);
            if (request.signal.aborted || requestedKey !== key || disposed) return;
            result = normalizeLyrics(data);
            cache.set(key, result);
            if (cache.size > 20) cache.delete(cache.keys().next().value);
        } catch (error) {
            if (request.signal.aborted || requestedKey !== key || disposed) return;
            result = { status: 'empty', lines: [] };
        }
        render();
    }
    function sync() {
        const next = trackKey();
        if (next !== key) {
            controller?.abort(); key = next; result = { status: 'idle', lines: [] }; render();
        }
        toggle.disabled = !state.room?.track && !open;
        load();
    }
    function update(position, force = false) {
        if (!expanded()) return;
        if (result.status !== 'synced') {
            previewLine.textContent = result.status === 'plain' ? result.lines.find(line => line.text.trim())?.text || '' : '';
            return;
        }
        let next = -1;
        for (let i = 0; i < result.lines.length && result.lines[i].time <= position; i++) next = i;
        previewLine.textContent = result.lines[Math.max(0, next)]?.text || 'Instrumental';
        if (next === active && !force) return;
        container.children[active]?.classList.remove('is-current');
        container.children[active]?.removeAttribute('aria-current');
        active = next;
        const node = container.children[active];
        node?.classList.add('is-current'); node?.setAttribute('aria-current', 'true');
        if (visible() && following && node) {
            const top = panel.scrollTop + node.getBoundingClientRect().top - panel.getBoundingClientRect().top - panel.clientHeight / 2 + node.clientHeight / 2;
            panel.scrollTo({ top: Math.max(0, top), behavior: force || matchMedia('(prefers-reduced-motion: reduce)').matches ? 'auto' : 'smooth' });
        }
    }
    function stopFollowing() {
        if (result.status === 'synced') { following = false; follow.hidden = !visible(); }
    }
    panel.addEventListener('wheel', stopFollowing, { passive: true });
    panel.addEventListener('touchstart', stopFollowing, { passive: true });
    panel.addEventListener('pointerdown', stopFollowing);
    panel.addEventListener('keydown', event => { if (['ArrowUp', 'ArrowDown', 'PageUp', 'PageDown', 'Home', 'End', ' '].includes(event.key)) stopFollowing(); });
    function setOpen(next) {
        open = next; render(); sync();
        if (open) panel.focus({ preventScroll: true }); else toggle.focus({ preventScroll: true });
    }
    toggle.addEventListener('click', () => setOpen(!open));
    preview.addEventListener('click', () => setOpen(true));
    follow.addEventListener('click', () => { panel.focus({ preventScroll: true }); following = true; follow.hidden = true; update(playbackPosition(), true); });
    const resize = new ResizeObserver(() => {
        container.style.paddingBlock = result.status === 'synced' ? `${Math.max(24, panel.clientHeight / 2 - 32)}px` : '24px';
        if (following) update(playbackPosition(), true);
    });
    resize.observe(panel);
    on('room', sync); on('progress', update);
    on('dispose', () => { disposed = true; controller?.abort(); resize.disconnect(); cache.clear(); });
    return { viewChanged() { render(); sync(); } };
}
