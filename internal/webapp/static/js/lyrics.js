// Lyrics follow the same authoritative room position as the player's progress.
import { state, on, playbackPosition, lyricsAPI, canControl } from './core.js';

export function normalizeLyrics(data) {
    if (data.unsupported) return { status: 'unsupported', lines: [] };
    const displayText = text => text.trim().replace(/^\[\s*संगीत\s*\]$/u, '♪');
    const lrc = typeof data.lrc === 'string' ? data.lrc : '';
    const timestamp = /\[(\d+):(\d{2})(?:[.:](\d{1,3}))?\]/g;
    const parsedOffset = Number(lrc.match(/\[offset:([+-]?\d+)\]/i)?.[1] || 0) / 1000;
    const offset = Number.isFinite(parsedOffset) ? parsedOffset : 0;
    const lines = lrc.split(/\r?\n/).flatMap(row => {
        const text = displayText(row.replace(timestamp, ''));
        return [...row.matchAll(timestamp)].filter(match => Number(match[2]) < 60).map(match => ({
            time: Math.max(0, Number(match[1]) * 60 + Number(match[2]) + Number(`0.${match[3] || 0}`) + offset), text
        })).filter(line => Number.isFinite(line.time));
    });
    if (lines.some(line => line.text.trim())) return { status: 'synced', lines: lines.sort((a, b) => a.time - b.time) };
    const plain = typeof data.text === 'string' ? data.text.trim() : '';
    return { status: plain ? 'plain' : 'empty', lines: plain ? plain.split(/\r?\n/).map(text => ({ text: displayText(text) })) : [] };
}

export function createLyrics({ player, stage, panel, status, lines: container, headerTitle, headerSubtitle, triggers, seek }) {
    let open = false, key = '', result = { status: 'idle', lines: [] }, active = -1, following = true;
    let controller = null, resumeTimer = null, disposed = false;
    const cache = new Map();
    const visible = () => open && state.expandedPlayer && !disposed;
    const expanded = () => state.expandedPlayer && !disposed;
    const trackKey = () => state.room?.track ? `${state.room.track.id}|${state.room.track.url}` : '';
    function render() {
        player.classList.toggle('has-lyrics', visible());
        panel.hidden = !visible();
        stage.hidden = !visible();
        panel.dataset.state = result.status;
        headerTitle.parentElement.hidden = !visible();
        headerTitle.textContent = visible() ? state.room?.track?.title || 'Track lyrics' : '';
        headerSubtitle.textContent = visible() ? state.room?.track?.artist || '' : '';
        const messages = { idle: 'Choose a track to see lyrics.', loading: 'Loading lyrics…', empty: 'No lyrics available for this track.', unsupported: 'No lyrics available for this track.', error: 'Lyrics could not load. Try again.', plain: 'Timing isn’t available for these lyrics.', synced: 'Synced lyrics' };
        status.textContent = messages[result.status];
        status.hidden = result.status === 'synced';
        for (const trigger of triggers) {
            trigger.disabled = !state.room?.track || ['empty', 'unsupported'].includes(result.status);
            trigger.title = trigger.disabled ? 'No lyrics available' : 'Open lyrics';
        }
        container.replaceChildren(...result.lines.map(line => {
            const node = document.createElement(result.status === 'synced' ? 'button' : 'p');
            node.className = 'lyric-line';
            const copy = document.createElement('span'); copy.textContent = line.text || '…'; node.append(copy);
            if (result.status === 'synced') { node.type = 'button'; node.dataset.time = String(line.time); node.disabled = !canControl() || (state.room?.track?.duration > 0 && line.time > state.room.track.duration); node.setAttribute('aria-label', `Seek to ${line.text === '♪' ? 'instrumental break' : line.text || 'instrumental'} at ${Math.floor(line.time / 60)}:${String(Math.floor(line.time % 60)).padStart(2, '0')}`); }
            return node;
        }));
        container.style.paddingBlock = result.status === 'synced' ? `${Math.max(24, panel.clientHeight / 2 - 32)}px` : '24px';
        clearTimeout(resumeTimer);
        active = -1; following = true; panel.scrollTop = 0;
        update(playbackPosition(), true, true);
    }
    async function load() {
        if (result.status !== 'idle' || !key || state.room.track.ready === false) return;
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
            result = { status: 'error', lines: [] };
        }
        render();
    }
    function sync() {
        const next = trackKey();
        if (next !== key) {
            controller?.abort(); key = next; result = { status: 'idle', lines: [] }; render();
        }
        load();
    }
    function update(position, force = false, instant = false) {
        if (!expanded()) return;
        if (result.status !== 'synced') {
            return;
        }
        let next = -1;
        for (let i = 0; i < result.lines.length && result.lines[i].time <= position; i++) next = i;
        if (next === active && !force) return;
        if (next !== active) {
            for (const index of [active - 1, active, active + 1]) container.children[index]?.classList.remove('is-near');
            container.children[active]?.classList.remove('is-current');
            container.children[active]?.removeAttribute('aria-current');
            active = next;
            for (const index of [active - 1, active + 1]) container.children[index]?.classList.add('is-near');
        }
        const node = container.children[active];
        node?.classList.add('is-current'); node?.setAttribute('aria-current', 'true');
        if (visible() && following && node) {
            const top = panel.scrollTop + node.getBoundingClientRect().top - panel.getBoundingClientRect().top - panel.clientHeight / 2 + node.clientHeight / 2;
            panel.scrollTo({ top: Math.max(0, top), behavior: instant || matchMedia('(prefers-reduced-motion: reduce)').matches ? 'auto' : 'smooth' });
        }
    }
    function pauseFollowing() {
        if (!visible() || result.status !== 'synced') return;
        if (following) panel.scrollTo({ top: panel.scrollTop, behavior: 'auto' });
        following = false;
        clearTimeout(resumeTimer);
        resumeTimer = setTimeout(() => {
            following = true;
            update(playbackPosition(), true);
        }, 3000);
    }
    panel.addEventListener('wheel', pauseFollowing, { passive: true });
    panel.addEventListener('touchstart', pauseFollowing, { passive: true });
    panel.addEventListener('touchend', pauseFollowing, { passive: true });
    panel.addEventListener('pointerdown', pauseFollowing);
    panel.addEventListener('scroll', () => { if (!following) pauseFollowing(); }, { passive: true });
    panel.addEventListener('keydown', event => { if (['ArrowUp', 'ArrowDown', 'PageUp', 'PageDown', 'Home', 'End', ' '].includes(event.key)) pauseFollowing(); });
    container.addEventListener('click', event => {
        const line = event.target.closest('button[data-time]');
        if (!line || line.disabled || !canControl()) return;
        const time = Number(line.dataset.time);
        if (!Number.isFinite(time)) return;
        clearTimeout(resumeTimer); following = true;
        seek(time);
    });
    function setOpen(next) {
        if (next && result.status === 'error') result = { status: 'idle', lines: [] };
        open = next; render(); sync();
        if (open) panel.focus({ preventScroll: true }); else player.querySelector('.player-artwork').focus({ preventScroll: true });
    }
    const resize = new ResizeObserver(() => {
        container.style.paddingBlock = result.status === 'synced' ? `${Math.max(24, panel.clientHeight / 2 - 32)}px` : '24px';
        if (following) update(playbackPosition(), true);
    });
    resize.observe(panel);
    on('room', sync); on('progress', update);
    const seekability = () => { for (const line of container.querySelectorAll('button[data-time]')) line.disabled = !canControl() || (state.room?.track?.duration > 0 && Number(line.dataset.time) > state.room.track.duration); };
    on('permissions', seekability); on('connection', seekability);
    on('dispose', () => { disposed = true; controller?.abort(); clearTimeout(resumeTimer); resize.disconnect(); cache.clear(); });
    return { open() { if (expanded() && state.room?.track && !['empty', 'unsupported'].includes(result.status)) setOpen(true); }, close() { if (open) setOpen(false); }, isOpen() { return visible(); }, viewChanged() { if (!expanded()) open = false; render(); sync(); } };
}
