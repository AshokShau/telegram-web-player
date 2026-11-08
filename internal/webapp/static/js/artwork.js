import { on } from './core.js?v=119';

export const fallbackArtwork = '/static/assets/artwork.svg?v=92';
const artworkRequests = new WeakMap();
const retryDelays = [1200, 3500, 10000];

function themedFallback() {
    const light = document.documentElement.dataset.theme === 'light';
    const [sleeve, record, groove, shine, label, spindle] = light
        ? ['#e8edf3', '#d4dde9', '#9fb1c6', '#ffffff', '#1676b9', '#e8edf3']
        : ['#272c35', '#1c2129', '#435061', '#8393a7', '#339fec', '#20242c'];
    const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="400" height="400" viewBox="0 0 400 400"><rect width="400" height="400" fill="${sleeve}"/><circle cx="200" cy="200" r="139" fill="${record}"/><g fill="none" stroke="${groove}" stroke-width="2"><circle cx="200" cy="200" r="111"/><circle cx="200" cy="200" r="94"/><circle cx="200" cy="200" r="77"/></g><path d="M105 99a139 139 0 0 1 180-4" fill="none" stroke="${shine}" stroke-width="3" opacity="${light ? '.7' : '.32'}"/><circle cx="200" cy="200" r="30" fill="${label}"/><circle cx="200" cy="200" r="7" fill="${spindle}"/></svg>`;
    return `data:image/svg+xml,${encodeURIComponent(svg)}`;
}

export function refreshFallbackArtwork() {
    const fallback = themedFallback();
    for (const image of document.querySelectorAll('img[data-artwork-source], img[src^="/static/assets/artwork.svg"]')) {
        if (image.dataset.artworkSource === fallbackArtwork || image.getAttribute('src')?.startsWith('data:image/svg+xml') || image.getAttribute('src')?.startsWith('/static/assets/artwork.svg')) image.src = fallback;
    }
}

export function setTrackArtwork(image, source, refresh = false) {
    const previous = artworkRequests.get(image);
    if (previous?.source === source && !refresh) return;
    if (previous) clearTimeout(previous.timer);
    const request = { source, attempt: 0, timer: null };
    artworkRequests.set(image, request);
    image.dataset.artworkSource = source;
    function load(retry = false) {
        if (artworkRequests.get(image) !== request) return;
        let url = source;
        if ((retry || refresh) && new URL(source, location.origin).pathname.startsWith('/thumbnail/telegram/')) {
            const fresh = new URL(source, location.origin);
            fresh.searchParams.set('artwork_retry', String(Date.now())); url = fresh.href;
        }
        image.src = source === fallbackArtwork ? themedFallback() : url;
    }
    image.onerror = () => {
        if (artworkRequests.get(image) !== request || image.getAttribute('src')?.startsWith('data:')) return;
        image.src = themedFallback();
        if (request.attempt < retryDelays.length) {
            request.timer = setTimeout(() => {
                if (image.isConnected) load(true);
            }, retryDelays[request.attempt++]);
        }
    };
    load();
}

function refreshArtwork() {
    if (document.hidden) return;
    for (const image of document.querySelectorAll('img[data-artwork-source]')) {
        const source = image.dataset.artworkSource;
        if (new URL(source, location.origin).pathname.startsWith('/thumbnail/telegram/') || image.getAttribute('src') === fallbackArtwork) {
            setTrackArtwork(image, source, true);
        }
    }
}
window.addEventListener('online', refreshArtwork);
window.addEventListener('pageshow', refreshArtwork);
document.addEventListener('visibilitychange', refreshArtwork);

export function createAtmosphere(layer, backdrop, onColor) {
    let source = '', generation = 0, sample = null, paletteLoading = false;
    let retryTimer;
    function paletteSource(raw) {
        if (raw === fallbackArtwork) return '';
        const url = new URL(raw, location.origin);
        if (url.pathname.toLowerCase().endsWith('.svg')) return '';
        return ['http:', 'https:'].includes(url.protocol) ? url.href : '';
    }
    function sampleColor(image) {
        try {
            const canvas = document.createElement('canvas'); canvas.width = canvas.height = 16;
            const context = canvas.getContext('2d', { willReadFrequently: true });
            context.drawImage(image, 0, 0, 16, 16);
            const pixels = context.getImageData(0, 0, 16, 16).data;
            const totals = [0, 0, 0]; let count = 0;
            for (let index = 0; index < pixels.length; index += 4) {
                if (pixels[index + 3] < 128) continue;
                for (let channel = 0; channel < 3; channel++) totals[channel] += pixels[index + channel];
                count++;
            }
            return count ? totals.map(total => total / count) : null;
        } catch { return null; }
    }
    function paint() {
        const average = sample ? sample.reduce((sum, value) => sum + value, 0) / 3 : 0;
        let color = sample ? sample.map(value => Math.round(Math.max(28, Math.min(255, average + (value - average) * 1.25) * .68))) : [36, 39, 49];
        const contrast = values => {
            const channels = values.map(value => value / 255).map(value => value <= .04045 ? value / 12.92 : ((value + .055) / 1.055) ** 2.4);
            return 1.05 / (.05 + channels[0] * .2126 + channels[1] * .7152 + channels[2] * .0722);
        };
        while (contrast(color) < 7.5) color = color.map(value => Math.round(value * .9));
        const hex = `#${color.map(value => value.toString(16).padStart(2, '0')).join('')}`;
        layer.style.backgroundColor = hex;
        onColor?.(hex);
    }
    function loadPalette(id, attempt = 0) {
        clearTimeout(retryTimer);
        const url = paletteSource(source);
        if (!url) { sample = null; paint(); return; }
        paletteLoading = true;
        const image = new Image(); image.crossOrigin = 'anonymous';
        image.onload = () => {
            if (id !== generation) return;
            paletteLoading = false; clearTimeout(retryTimer);
            sample = sampleColor(image); paint();
        };
        image.onerror = () => {
            if (id !== generation) return;
            paletteLoading = false; sample = null; paint();
            if (attempt < retryDelays.length) retryTimer = setTimeout(() => loadPalette(id, attempt + 1), retryDelays[attempt]);
        };
        image.src = url;
    }
    paint();
    on('dispose', () => { generation++; clearTimeout(retryTimer); layer.hidden = true; });
    const refresh = () => { paint(); if (source && !sample && !paletteLoading) loadPalette(generation); };
    window.addEventListener('online', refresh);
    window.addEventListener('pageshow', refresh);
    document.addEventListener('visibilitychange', () => { if (!document.hidden) refresh(); });
    return {
        setArtwork(url) {
            if (url === source) return;
            source = url; generation++; clearTimeout(retryTimer);
            const id = generation;
            if (url === fallbackArtwork) { backdrop.src = themedFallback(); sample = null; paint(); return; }
            const image = new Image();
            image.onload = () => { if (id === generation) backdrop.src = image.src; };
            image.onerror = () => { if (id === generation) backdrop.src = themedFallback(); };
            image.src = url;
            loadPalette(id);
        },
        setExpanded(expanded) {
            const wasHidden = layer.hidden;
            layer.hidden = !expanded;
            if (expanded) { paint(); if (wasHidden && source && !paletteLoading) loadPalette(generation); }
        },
        refresh
    };
}
