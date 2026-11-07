import { on } from './core.js?v=111';

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

export function createAtmosphere(layer, backdrop) {
    let source = '';
    let timer, sample = null;
    backdrop.crossOrigin = 'anonymous';
    function paint() {
        const canvas = getComputedStyle(document.documentElement).getPropertyValue('--canvas').trim();
        const match = /^#([\da-f]{6})$/i.exec(canvas);
        if (!match) return;
        const base = [0, 2, 4].map(index => parseInt(match[1].slice(index, index + 2), 16));
        const strength = sample ? (document.documentElement.dataset.theme === 'light' ? .13 : .18) : 0;
        layer.style.backgroundColor = `rgb(${base.map((value, index) => Math.round(value * (1 - strength) + (sample?.[index] || 0) * strength)).join(' ')})`;
    }
    backdrop.addEventListener('error', () => { backdrop.style.opacity = '0'; sample = null; paint(); });
    backdrop.addEventListener('load', () => {
        const url = new URL(source || fallbackArtwork, location.origin);
        sample = null;
        if (url.origin === location.origin && !url.pathname.endsWith('.svg') && !backdrop.currentSrc.startsWith('data:')) {
            try {
                const canvas = document.createElement('canvas'); canvas.width = canvas.height = 16;
                const context = canvas.getContext('2d', { willReadFrequently: true });
                context.drawImage(backdrop, 0, 0, 16, 16);
                const pixels = context.getImageData(0, 0, 16, 16).data;
                const totals = [0, 0, 0]; let count = 0;
                for (let index = 0; index < pixels.length; index += 16) {
                    if (pixels[index + 3] < 128) continue;
                    for (let channel = 0; channel < 3; channel++) totals[channel] += pixels[index + channel];
                    count++;
                }
                if (count) sample = totals.map(total => total / count);
            } catch { /* The blurred artwork still supplies color when sampling is unavailable. */ }
        }
        paint(); backdrop.style.opacity = '';
    });
    paint();
    on('dispose', () => { clearTimeout(timer); layer.hidden = true; });
    return {
        setArtwork(url) {
            if (url === source) return;
            const hadArtwork = Boolean(source);
            source = url;
            clearTimeout(timer);
            backdrop.style.opacity = '0';
            timer = setTimeout(() => setTrackArtwork(backdrop, url), hadArtwork && !matchMedia('(prefers-reduced-motion: reduce)').matches ? 180 : 0);
        },
        setExpanded(expanded) { layer.hidden = !expanded; },
        refresh: paint
    };
}
